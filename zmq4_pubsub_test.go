// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4_test

import (
	"context"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-zeromq/zmq4"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

var (
	pubsubs = []testCasePubSub{
		{
			name:     "tcp-pub-sub",
			endpoint: must(EndPoint("tcp")),
			pub:      zmq4.NewPub(bkg),
			sub0:     zmq4.NewSub(bkg, zmq4.WithID(zmq4.SocketIdentity("sub0"))),
			sub1:     zmq4.NewSub(bkg, zmq4.WithID(zmq4.SocketIdentity("sub1"))),
			sub2:     zmq4.NewSub(bkg, zmq4.WithID(zmq4.SocketIdentity("sub2"))),
		},
		{
			name:     "ipc-pub-sub",
			endpoint: "ipc://ipc-pub-sub",
			pub:      zmq4.NewPub(bkg),
			sub0:     zmq4.NewSub(bkg, zmq4.WithID(zmq4.SocketIdentity("sub0"))),
			sub1:     zmq4.NewSub(bkg, zmq4.WithID(zmq4.SocketIdentity("sub1"))),
			sub2:     zmq4.NewSub(bkg, zmq4.WithID(zmq4.SocketIdentity("sub2"))),
		},
	}
)

type testCasePubSub struct {
	name     string
	skip     bool
	endpoint string
	pub      zmq4.Socket
	sub0     zmq4.Socket
	sub1     zmq4.Socket
	sub2     zmq4.Socket
}

func TestPubSub(t *testing.T) {
	var (
		//topics = []string{"", "MSG ", "MSG "}
		topics      = []string{"", "MSG", "msg"}
		wantNumMsgs = []int{3, 1, 1}
		msg0        = zmq4.NewMsgString("anything")
		msg1        = zmq4.NewMsgString("MSG 1")
		msg2        = zmq4.NewMsgString("msg 2")
		msgs        = [][]zmq4.Msg{
			0: {msg0, msg1, msg2},
			1: {msg1},
			2: {msg2},
		}
	)

	for _, tc := range pubsubs {
		t.Run(tc.name, func(t *testing.T) {
			if tc.skip {
				t.Skipf(tc.name)
			}

			// FIXME(sbinet): we should probably do this at the zmq4.Socket.Close level
			if strings.HasPrefix(tc.endpoint, "ipc://") {
				defer os.Remove(tc.endpoint[len("ipc://"):])
			}

			ep := tc.endpoint

			ctx, timeout := context.WithTimeout(context.Background(), 20*time.Second)
			defer timeout()

			defer tc.pub.Close()
			defer tc.sub0.Close()
			defer tc.sub1.Close()
			defer tc.sub2.Close()

			nmsgs := []int{0, 0, 0}
			subs := []zmq4.Socket{tc.sub0, tc.sub1, tc.sub2}

			var wg1 sync.WaitGroup
			var wg2 sync.WaitGroup
			wg1.Add(len(subs))
			wg2.Add(len(subs))

			grp, ctx := errgroup.WithContext(ctx)
			grp.Go(func() error {

				err := tc.pub.Listen(ep)
				if err != nil {
					return errors.Wrapf(err, "could not listen")
				}

				wg1.Wait()
				wg2.Wait()

				time.Sleep(1 * time.Second)

				for _, msg := range msgs[0] {
					err = tc.pub.Send(msg)
					if err != nil {
						return errors.Wrapf(err, "could not send message %v", msg)
					}
				}

				return err
			})

			for isub := range subs {
				func(isub int, sub zmq4.Socket) {
					grp.Go(func() error {
						var err error
						err = sub.Dial(ep)
						if err != nil {
							return errors.Wrapf(err, "could not dial")
						}
						wg1.Done()
						wg1.Wait()

						err = sub.SetOption(zmq4.OptionSubscribe, topics[isub])
						if err != nil {
							return errors.Wrapf(err, "could not subscribe to topic %q", topics[isub])
						}

						wg2.Done()
						wg2.Wait()

						msgs := msgs[isub]
						for imsg, want := range msgs {
							msg, err := sub.Recv()
							if err != nil {
								return errors.Wrapf(err, "could not recv message %v", want)
							}
							if !reflect.DeepEqual(msg, want) {
								return errors.Errorf("sub[%d][msg=%d]: got = %v, want= %v", isub, imsg, msg, want)
							}
							nmsgs[isub]++
						}

						return err
					})
				}(isub, subs[isub])
			}

			if err := grp.Wait(); err != nil {
				t.Fatal(err)
			}

			for i, want := range wantNumMsgs {
				if want != nmsgs[i] {
					t.Errorf("sub[%d]: got %d messages, want %d msgs=%v", i, nmsgs[i], want, nmsgs)
				}
			}
		})
	}
}
