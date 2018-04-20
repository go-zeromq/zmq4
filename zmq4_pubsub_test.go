// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4_test

import (
	"context"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-zeromq/zmq4"
	"github.com/go-zeromq/zmq4/zmtp"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

var (
	pubsubs = []testCasePubSub{
		{
			name:     "tcp-pub-sub",
			endpoint: must(EndPoint("tcp")),
			pub:      zmq4.NewPub(bkg),
			sub1:     zmq4.NewSub(bkg),
			sub2:     zmq4.NewSub(bkg),
		},
		{
			name:     "ipc-pub-sub",
			endpoint: "ipc://ipc-pub-sub",
			pub:      zmq4.NewPub(bkg),
			sub1:     zmq4.NewSub(bkg),
			sub2:     zmq4.NewSub(bkg),
		},
	}
)

type testCasePubSub struct {
	name     string
	skip     bool
	endpoint string
	pub      zmq4.Socket
	sub1     zmq4.Socket
	sub2     zmq4.Socket
}

func TestPubSub(t *testing.T) {
	var (
		topic = "MSG"
		msg1  = zmtp.NewMsgString("MSG 1")
		msg2  = zmtp.NewMsgString("MSG 2")
		msgs  = []zmtp.Msg{msg1, msg2}
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
			defer tc.sub1.Close()
			defer tc.sub2.Close()

			ready := make(chan int)
			grp, ctx := errgroup.WithContext(ctx)
			grp.Go(func() error {

				err := tc.pub.Listen(ep)
				if err != nil {
					return errors.Wrapf(err, "could not listen")
				}

				<-ready
				<-ready

				time.Sleep(1 * time.Second)

				for _, msg := range msgs {
					err = tc.pub.Send(msg)
					if err != nil {
						return errors.Wrapf(err, "could not send message %v", msg)
					}
				}

				return err
			})

			for isub, sub := range []zmq4.Socket{tc.sub1, tc.sub2} {
				func(isub int, sub zmq4.Socket) {
					grp.Go(func() error {
						var err error
						err = sub.Dial(ep)
						if err != nil {
							return errors.Wrapf(err, "could not dial")
						}
						err = sub.SetOption(zmq4.OptionSubscribe, topic)
						if err != nil {
							return errors.Wrapf(err, "could not subscribe to topic %q", topic)
						}

						ready <- isub
						for _, want := range msgs {
							msg, err := sub.Recv()
							if err != nil {
								return errors.Wrapf(err, "could not recv message %v", want)
							}
							if !reflect.DeepEqual(msg, want) {
								return errors.Wrapf(err, "got = %v, want= %v", msg, want)
							}
						}

						return err
					})
				}(isub, sub)
			}

			if err := grp.Wait(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
