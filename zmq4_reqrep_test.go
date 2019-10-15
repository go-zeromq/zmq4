// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/go-zeromq/zmq4"
	"golang.org/x/sync/errgroup"
	"golang.org/x/xerrors"
)

var (
	reqreps = []testCaseReqRep{
		{
			name:     "tcp-req-rep",
			endpoint: must(EndPoint("tcp")),
			req:      zmq4.NewReq(bkg),
			rep:      zmq4.NewRep(bkg),
		},
		{
			name:     "ipc-req-rep",
			endpoint: "ipc://ipc-req-rep",
			req:      zmq4.NewReq(bkg),
			rep:      zmq4.NewRep(bkg),
		},
		{
			name:     "inproc-req-rep",
			endpoint: "inproc://inproc-req-rep",
			req:      zmq4.NewReq(bkg),
			rep:      zmq4.NewRep(bkg),
		},
	}
)

type testCaseReqRep struct {
	name     string
	skip     bool
	endpoint string
	req      zmq4.Socket
	rep      zmq4.Socket
}

func TestReqRep(t *testing.T) {
	var (
		reqName = zmq4.NewMsgString("NAME")
		reqLang = zmq4.NewMsgString("LANG")
		reqQuit = zmq4.NewMsgString("QUIT")
		repName = zmq4.NewMsgString("zmq4")
		repLang = zmq4.NewMsgString("Go")
		repQuit = zmq4.NewMsgString("bye")
	)

	for i := range reqreps {
		tc := reqreps[i]
		t.Run(tc.name, func(t *testing.T) {
			defer tc.req.Close()
			defer tc.rep.Close()

			if tc.skip {
				t.Skipf(tc.name)
			}
			t.Parallel()

			ep := tc.endpoint
			cleanUp(ep)

			ctx, timeout := context.WithTimeout(context.Background(), 20*time.Second)
			defer timeout()

			grp, ctx := errgroup.WithContext(ctx)
			grp.Go(func() error {

				err := tc.rep.Listen(ep)
				if err != nil {
					return xerrors.Errorf("could not listen: %w", err)
				}

				if addr := tc.rep.Addr(); addr == nil {
					return xerrors.Errorf("listener with nil Addr")
				}

				loop := true
				for loop {
					msg, err := tc.rep.Recv()
					if err != nil {
						return xerrors.Errorf("could not recv REQ message: %w", err)
					}
					var rep zmq4.Msg
					switch string(msg.Frames[0]) {
					case "NAME":
						rep = repName
					case "LANG":
						rep = repLang
					case "QUIT":
						rep = repQuit
						loop = false
					}

					err = tc.rep.Send(rep)
					if err != nil {
						return xerrors.Errorf("could not send REP message to %v: %w", msg, err)
					}
				}

				return err
			})
			grp.Go(func() error {

				err := tc.req.Dial(ep)
				if err != nil {
					return xerrors.Errorf("could not dial: %w", err)
				}

				if addr := tc.req.Addr(); addr != nil {
					return xerrors.Errorf("dialer with non-nil Addr")
				}

				for _, msg := range []struct {
					req zmq4.Msg
					rep zmq4.Msg
				}{
					{reqName, repName},
					{reqLang, repLang},
					{reqQuit, repQuit},
				} {
					err = tc.req.Send(msg.req)
					if err != nil {
						return xerrors.Errorf("could not send REQ message %v: %w", msg.req, err)
					}
					rep, err := tc.req.Recv()
					if err != nil {
						return xerrors.Errorf("could not recv REP message %v: %w", msg.req, err)
					}

					if got, want := rep, msg.rep; !reflect.DeepEqual(got, want) {
						return xerrors.Errorf("got = %v, want= %v", got, want)
					}
				}

				return err
			})
			if err := grp.Wait(); err != nil {
				t.Fatalf("error: %+v", err)
			}
		})
	}
}
