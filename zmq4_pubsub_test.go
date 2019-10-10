// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4_test

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/go-zeromq/zmq4"
	"golang.org/x/sync/errgroup"
	"golang.org/x/xerrors"
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
		{
			name:     "inproc-pub-sub",
			endpoint: "inproc://inproc-pub-sub",
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

func TestNotBlockingSendOnPub(t *testing.T) {

	pub := zmq4.NewPub(context.Background())
	defer pub.Close()

	err := pub.Listen(must(EndPoint("tcp")))
	if err != nil {
		t.Fatalf("could not listen on end point: %+v", err)
	}

	errc := make(chan error)
	go func() {
		errc <- pub.Send(zmq4.NewMsg([]byte("blocked?")))
	}()

	select {
	case <-time.After(5 * time.Second):
		t.Fatalf("pub socket should not block!")
	case err := <-errc:
		if err != nil {
			t.Fatalf("unexpected error: %+v", err)
		}
	}
}

func TestPubSub(t *testing.T) {
	var (
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

	for i := range pubsubs {
		tc := pubsubs[i]
		t.Run(tc.name, func(t *testing.T) {
			defer tc.pub.Close()
			defer tc.sub0.Close()
			defer tc.sub1.Close()
			defer tc.sub2.Close()

			if tc.skip {
				t.Skipf(tc.name)
			}
			t.Parallel()

			ep := tc.endpoint
			cleanUp(ep)

			ctx, timeout := context.WithTimeout(context.Background(), 20*time.Second)
			defer timeout()

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
					return xerrors.Errorf("could not listen: %w", err)
				}

				wg1.Wait()
				wg2.Wait()

				time.Sleep(1 * time.Second)

				for _, msg := range msgs[0] {
					err = tc.pub.Send(msg)
					if err != nil {
						return xerrors.Errorf("could not send message %v: %w", msg, err)
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
							return xerrors.Errorf("could not dial: %w", err)
						}
						wg1.Done()
						wg1.Wait()

						err = sub.SetOption(zmq4.OptionSubscribe, topics[isub])
						if err != nil {
							return xerrors.Errorf("could not subscribe to topic %q: %w", topics[isub], err)
						}

						wg2.Done()
						wg2.Wait()

						msgs := msgs[isub]
						for imsg, want := range msgs {
							msg, err := sub.Recv()
							if err != nil {
								return xerrors.Errorf("could not recv message %v: %w", want, err)
							}
							if !reflect.DeepEqual(msg, want) {
								return xerrors.Errorf("sub[%d][msg=%d]: got = %v, want= %v", isub, imsg, msg, want)
							}
							nmsgs[isub]++
						}

						return err
					})
				}(isub, subs[isub])
			}

			if err := grp.Wait(); err != nil {
				t.Fatalf("error: %+v", err)
			}

			for i, want := range wantNumMsgs {
				if want != nmsgs[i] {
					t.Errorf("sub[%d]: got %d messages, want %d msgs=%v", i, nmsgs[i], want, nmsgs)
				}
			}
		})
	}
}
