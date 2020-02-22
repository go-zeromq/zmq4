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
			req1:     zmq4.NewReq(bkg),
			rep:      zmq4.NewRep(bkg),
		},
		{
			name:     "ipc-req-rep",
			endpoint: "ipc://ipc-req-rep",
			req1:     zmq4.NewReq(bkg),
			rep:      zmq4.NewRep(bkg),
		},
		{
			name:     "inproc-req-rep",
			endpoint: "inproc://inproc-req-rep",
			req1:     zmq4.NewReq(bkg),
			rep:      zmq4.NewRep(bkg),
		},
	}
)

type testCaseReqRep struct {
	name     string
	skip     bool
	endpoint string
	req1     zmq4.Socket
	req2     zmq4.Socket
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
			defer tc.req1.Close()
			defer tc.rep.Close()

			ep := tc.endpoint
			cleanUp(ep)

			if tc.skip {
				t.Skipf(tc.name)
			}
			t.Parallel()

			ctx, timeout := context.WithTimeout(context.Background(), 20*time.Second)
			defer timeout()

			grp, _ := errgroup.WithContext(ctx)
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

				err := tc.req1.Dial(ep)
				if err != nil {
					return xerrors.Errorf("could not dial: %w", err)
				}

				if addr := tc.req1.Addr(); addr != nil {
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
					err = tc.req1.Send(msg.req)
					if err != nil {
						return xerrors.Errorf("could not send REQ message %v: %w", msg.req, err)
					}
					rep, err := tc.req1.Recv()
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

func TestMultiReqRepIssue70(t *testing.T) {
	var (
		reqName1 = zmq4.NewMsgString("NAME")
		reqLang1 = zmq4.NewMsgString("LANG")
		reqQuit1 = zmq4.NewMsgString("QUIT")
		reqName2 = zmq4.NewMsgString("NAME2")
		reqLang2 = zmq4.NewMsgString("LANG2")
		reqQuit2 = zmq4.NewMsgString("QUIT2")
		repName1 = zmq4.NewMsgString("zmq4")
		repLang1 = zmq4.NewMsgString("Go")
		repQuit1 = zmq4.NewMsgString("bye")
		repName2 = zmq4.NewMsgString("zmq42")
		repLang2 = zmq4.NewMsgString("Go2")
		repQuit2 = zmq4.NewMsgString("bye2")
	)

	reqreps := []testCaseReqRep{
		{
			name:     "tcp-req-rep",
			endpoint: must(EndPoint("tcp")),
			req1:     zmq4.NewReq(bkg),
			req2:     zmq4.NewReq(bkg),
			rep:      zmq4.NewRep(bkg),
		},
		{
			name:     "ipc-req-rep",
			endpoint: "ipc://ipc-req-rep",
			req1:     zmq4.NewReq(bkg),
			req2:     zmq4.NewReq(bkg),
			rep:      zmq4.NewRep(bkg),
		},
		{
			name:     "inproc-req-rep",
			endpoint: "inproc://inproc-req-rep",
			req1:     zmq4.NewReq(bkg),
			req2:     zmq4.NewReq(bkg),
			rep:      zmq4.NewRep(bkg),
		},
	}

	for i := range reqreps {
		tc := reqreps[i]
		t.Run(tc.name, func(t *testing.T) {
			defer tc.req1.Close()
			defer tc.req2.Close()
			defer tc.rep.Close()

			if tc.skip {
				t.Skipf(tc.name)
			}
			t.Parallel()

			ep := tc.endpoint
			cleanUp(ep)

			ctx, timeout := context.WithTimeout(context.Background(), 20*time.Second)
			defer timeout()

			grp, _ := errgroup.WithContext(ctx)
			grp.Go(func() error {
				err := tc.rep.Listen(ep)
				if err != nil {
					return xerrors.Errorf("could not listen: %w", err)
				}

				if addr := tc.rep.Addr(); addr == nil {
					return xerrors.Errorf("listener with nil Addr")
				}

				loop1, loop2 := true, true
				for loop1 || loop2 {
					msg, err := tc.rep.Recv()
					if err != nil {
						return xerrors.Errorf("could not recv REQ message: %w", err)
					}
					var rep zmq4.Msg
					switch string(msg.Frames[0]) {
					case "NAME":
						rep = repName1
					case "LANG":
						rep = repLang1
					case "QUIT":
						rep = repQuit1
						loop1 = false
					case "NAME2":
						rep = repName2
					case "LANG2":
						rep = repLang2
					case "QUIT2":
						rep = repQuit2
						loop2 = false
					}

					err = tc.rep.Send(rep)
					if err != nil {
						return xerrors.Errorf("could not send REP message to %v: %w", msg, err)
					}
				}
				return err
			})
			grp.Go(func() error {

				err := tc.req2.Dial(ep)
				if err != nil {
					return xerrors.Errorf("could not dial: %w", err)
				}

				if addr := tc.req2.Addr(); addr != nil {
					return xerrors.Errorf("dialer with non-nil Addr")
				}

				for _, msg := range []struct {
					req zmq4.Msg
					rep zmq4.Msg
				}{
					{reqName2, repName2},
					{reqLang2, repLang2},
					{reqQuit2, repQuit2},
				} {
					err = tc.req2.Send(msg.req)
					if err != nil {
						return xerrors.Errorf("could not send REQ message %v: %w", msg.req, err)
					}
					rep, err := tc.req2.Recv()
					if err != nil {
						return xerrors.Errorf("could not recv REP message %v: %w", msg.req, err)
					}

					if got, want := rep, msg.rep; !reflect.DeepEqual(got, want) {
						return xerrors.Errorf("got = %v, want= %v", got, want)
					}
				}
				return err
			})
			grp.Go(func() error {

				err := tc.req1.Dial(ep)
				if err != nil {
					return xerrors.Errorf("could not dial: %w", err)
				}

				if addr := tc.req1.Addr(); addr != nil {
					return xerrors.Errorf("dialer with non-nil Addr")
				}

				for _, msg := range []struct {
					req zmq4.Msg
					rep zmq4.Msg
				}{
					{reqName1, repName1},
					{reqLang1, repLang1},
					{reqQuit1, repQuit1},
				} {
					err = tc.req1.Send(msg.req)
					if err != nil {
						return xerrors.Errorf("could not send REQ message %v: %w", msg.req, err)
					}
					rep, err := tc.req1.Recv()
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
