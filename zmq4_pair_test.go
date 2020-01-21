// Copyright 2020 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4_test

import (
	"bytes"
	"context"
	"sync"
	"testing"
	"time"

	"github.com/go-zeromq/zmq4"
	"golang.org/x/sync/errgroup"
	"golang.org/x/xerrors"
)

var (
	pairs = []testCasePair{
		{
			name:     "tcp-pair-pair",
			endpoint: must(EndPoint("tcp")),
			srv:      zmq4.NewPair(bkg),
			cli:      zmq4.NewPair(bkg),
		},
		{
			name:     "ipc-pair-pair",
			endpoint: "ipc://ipc-pair-pair",
			srv:      zmq4.NewPair(bkg),
			cli:      zmq4.NewPair(bkg),
		},
		{
			name:     "inproc-pair-pair",
			endpoint: "inproc://inproc-pair-pair",
			srv:      zmq4.NewPair(bkg),
			cli:      zmq4.NewPair(bkg),
		},
	}
)

type testCasePair struct {
	name     string
	skip     bool
	endpoint string
	srv      zmq4.Socket
	cli      zmq4.Socket
}

func TestPair(t *testing.T) {
	var (
		msg0 = zmq4.NewMsgString("")
		msg1 = zmq4.NewMsgString("MSG 1")
		msg2 = zmq4.NewMsgString("msg 2")
		msgs = []zmq4.Msg{
			msg0,
			msg1,
			msg2,
		}
	)

	for i := range pairs {
		tc := pairs[i]
		t.Run(tc.name, func(t *testing.T) {
			defer tc.srv.Close()
			defer tc.cli.Close()

			ep := tc.endpoint
			cleanUp(ep)

			if tc.skip {
				t.Skipf(tc.name)
			}
			t.Parallel()

			var (
				wg1 sync.WaitGroup
				wg2 sync.WaitGroup
			)

			wg1.Add(1)
			wg2.Add(1)

			ctx, timeout := context.WithTimeout(context.Background(), 20*time.Second)
			defer timeout()

			grp, ctx := errgroup.WithContext(ctx)
			grp.Go(func() error {

				err := tc.srv.Listen(ep)
				if err != nil {
					return xerrors.Errorf("could not listen: %w", err)
				}

				if addr := tc.srv.Addr(); addr == nil {
					return xerrors.Errorf("listener with nil Addr")
				}

				wg1.Wait()
				wg2.Done()

				for _, msg := range msgs {
					err = tc.srv.Send(msg)
					if err != nil {
						return xerrors.Errorf("could not send message %v: %w", msg, err)
					}
					reply, err := tc.srv.Recv()
					if err != nil {
						return xerrors.Errorf("could not recv reply to %v: %w", msg, err)
					}

					if got, want := reply, zmq4.NewMsgString("reply: "+string(msg.Bytes())); !bytes.Equal(got.Bytes(), want.Bytes()) {
						return xerrors.Errorf("invalid cli reply for msg #%d: got=%v, want=%v", i, got, want)
					}
				}

				quit, err := tc.srv.Recv()
				if err != nil {
					return xerrors.Errorf("could not recv QUIT message: %w", err)
				}

				if got, want := quit, zmq4.NewMsgString("QUIT"); !bytes.Equal(got.Bytes(), want.Bytes()) {
					return xerrors.Errorf("invalid QUIT message from cli: got=%v, want=%v", got, want)
				}

				return err
			})

			grp.Go(func() error {

				err := tc.cli.Dial(ep)
				if err != nil {
					return xerrors.Errorf("could not dial: %w", err)
				}

				wg1.Done()
				wg2.Wait()

				for i := range msgs {
					msg, err := tc.cli.Recv()
					if err != nil {
						return xerrors.Errorf("could not recv #%d msg from srv: %w", i, err)
					}
					if !bytes.Equal(msg.Bytes(), msgs[i].Bytes()) {
						return xerrors.Errorf("invalid #%d msg from srv: got=%v, want=%v",
							msg, msgs[i],
						)
					}

					err = tc.cli.Send(zmq4.NewMsgString("reply: " + string(msg.Bytes())))
					if err != nil {
						return xerrors.Errorf("could not send message %v: %w", msg, err)
					}
				}

				err = tc.cli.Send(zmq4.NewMsgString("QUIT"))
				if err != nil {
					return xerrors.Errorf("could not send QUIT message: %w", err)
				}

				return err
			})

			if err := grp.Wait(); err != nil {
				t.Fatalf("error: %+v", err)
			}

			if err := ctx.Err(); err != nil && err != context.Canceled {
				t.Fatalf("error: %+v", err)
			}
		})
	}
}
