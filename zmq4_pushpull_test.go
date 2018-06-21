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
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

var (
	pushpulls = []testCasePushPull{
		{
			name:     "tcp-push-pull",
			endpoint: must(EndPoint("tcp")),
			push:     zmq4.NewPush(bkg),
			pull:     zmq4.NewPull(bkg),
		},
		{
			name:     "ipc-push-pull",
			endpoint: "ipc://ipc-push-pull",
			push:     zmq4.NewPush(bkg),
			pull:     zmq4.NewPull(bkg),
		},
		{
			name:     "inproc-push-pull",
			endpoint: "inproc://push-pull",
			push:     zmq4.NewPush(bkg),
			pull:     zmq4.NewPull(bkg),
		},
	}
)

type testCasePushPull struct {
	name     string
	skip     bool
	endpoint string
	push     zmq4.Socket
	pull     zmq4.Socket
}

func TestPushPull(t *testing.T) {
	var (
		hello = zmq4.NewMsg([]byte("HELLO WORLD"))
		bye   = zmq4.NewMsgFrom([]byte("GOOD"), []byte("BYE"))
	)

	for _, tc := range pushpulls {
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

			defer tc.pull.Close()
			defer tc.push.Close()

			grp, ctx := errgroup.WithContext(ctx)
			grp.Go(func() error {

				err := tc.push.Listen(ep)
				if err != nil {
					return errors.Wrapf(err, "could not listen")
				}

				err = tc.push.Send(hello)
				if err != nil {
					return errors.Wrapf(err, "could not send %v", hello)
				}

				err = tc.push.Send(bye)
				if err != nil {
					return errors.Wrapf(err, "could not send %v", bye)
				}
				return err
			})
			grp.Go(func() error {

				err := tc.pull.Dial(ep)
				if err != nil {
					return errors.Wrapf(err, "could not dial")
				}

				msg, err := tc.pull.Recv()
				if err != nil {
					return errors.Wrapf(err, "could not recv %v", hello)
				}

				if got, want := msg, hello; !reflect.DeepEqual(got, want) {
					return errors.Errorf("recv1: got = %v, want= %v", got, want)
				}

				msg, err = tc.pull.Recv()
				if err != nil {
					return errors.Wrapf(err, "could not recv %v", bye)
				}

				if got, want := msg, bye; !reflect.DeepEqual(got, want) {
					return errors.Errorf("recv2: got = %v, want= %v", got, want)
				}

				return err
			})
			if err := grp.Wait(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
