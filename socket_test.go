// Copyright 2020 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4_test

import (
	"context"
	"io"
	"net"
	"testing"
	"time"

	"github.com/go-zeromq/zmq4"
	"golang.org/x/sync/errgroup"
	"golang.org/x/xerrors"
)

func TestInvalidConn(t *testing.T) {
	t.Parallel()

	ep := must(EndPoint("tcp"))
	cleanUp(ep)

	ctx, timeout := context.WithTimeout(context.Background(), 20*time.Second)
	defer timeout()

	pub := zmq4.NewPub(ctx)
	defer pub.Close()

	err := pub.Listen(ep)
	if err != nil {
		t.Fatalf("could not listen on end-point: %+v", err)
	}

	grp, ctx := errgroup.WithContext(ctx)
	grp.Go(func() error {
		conn, err := net.Dial("tcp", ep[len("tcp://"):])
		if err != nil {
			return xerrors.Errorf("could not dial %q: %w", ep, err)
		}
		defer conn.Close()
		var reply = make([]byte, 64)
		_, err = io.ReadFull(conn, reply)
		if err != nil {
			return xerrors.Errorf("could not read reply bytes...: %w", err)
		}
		_, err = conn.Write(make([]byte, 64))
		if err != nil {
			return xerrors.Errorf("could not send bytes...: %w", err)
		}
		time.Sleep(1 * time.Second) // FIXME(sbinet): hugly.
		return nil
	})

	if err := grp.Wait(); err != nil {
		t.Fatalf("error: %+v", err)
	}

	if err := ctx.Err(); err != nil && err != context.Canceled {
		t.Fatalf("error: %+v", err)
	}
}

func TestConnPairs(t *testing.T) {
	t.Parallel()

	bkg := context.Background()

	for _, tc := range []struct {
		name  string
		srv   zmq4.Socket
		wrong zmq4.Socket
		cli   zmq4.Socket
	}{
		{
			name:  "pair",
			srv:   zmq4.NewPair(bkg, zmq4.WithLogger(zmq4.Devnull)),
			wrong: zmq4.NewSub(bkg, zmq4.WithLogger(zmq4.Devnull)),
			cli:   zmq4.NewPair(bkg, zmq4.WithLogger(zmq4.Devnull)),
		},
		{
			name:  "pub",
			srv:   zmq4.NewPub(bkg, zmq4.WithLogger(zmq4.Devnull)),
			wrong: zmq4.NewPair(bkg, zmq4.WithLogger(zmq4.Devnull)),
			cli:   zmq4.NewSub(bkg, zmq4.WithLogger(zmq4.Devnull)),
		},
		{
			name:  "sub",
			srv:   zmq4.NewSub(bkg, zmq4.WithLogger(zmq4.Devnull)),
			wrong: zmq4.NewPair(bkg, zmq4.WithLogger(zmq4.Devnull)),
			cli:   zmq4.NewPub(bkg, zmq4.WithLogger(zmq4.Devnull)),
		},
		{
			name:  "req",
			srv:   zmq4.NewReq(bkg, zmq4.WithLogger(zmq4.Devnull)),
			wrong: zmq4.NewPair(bkg, zmq4.WithLogger(zmq4.Devnull)),
			cli:   zmq4.NewRep(bkg, zmq4.WithLogger(zmq4.Devnull)),
		},
		{
			name:  "rep",
			srv:   zmq4.NewRep(bkg, zmq4.WithLogger(zmq4.Devnull)),
			wrong: zmq4.NewPair(bkg, zmq4.WithLogger(zmq4.Devnull)),
			cli:   zmq4.NewReq(bkg, zmq4.WithLogger(zmq4.Devnull)),
		},
		{
			name:  "dealer",
			srv:   zmq4.NewDealer(bkg, zmq4.WithLogger(zmq4.Devnull)),
			wrong: zmq4.NewPair(bkg, zmq4.WithLogger(zmq4.Devnull)),
			cli:   zmq4.NewRouter(bkg, zmq4.WithLogger(zmq4.Devnull)),
		},
		{
			name:  "router",
			srv:   zmq4.NewRouter(bkg, zmq4.WithLogger(zmq4.Devnull)),
			wrong: zmq4.NewPair(bkg, zmq4.WithLogger(zmq4.Devnull)),
			cli:   zmq4.NewDealer(bkg, zmq4.WithLogger(zmq4.Devnull)),
		},
		{
			name:  "pull",
			srv:   zmq4.NewPull(bkg, zmq4.WithLogger(zmq4.Devnull)),
			wrong: zmq4.NewPair(bkg, zmq4.WithLogger(zmq4.Devnull)),
			cli:   zmq4.NewPush(bkg, zmq4.WithLogger(zmq4.Devnull)),
		},
		{
			name:  "push",
			srv:   zmq4.NewPush(bkg, zmq4.WithLogger(zmq4.Devnull)),
			wrong: zmq4.NewPair(bkg, zmq4.WithLogger(zmq4.Devnull)),
			cli:   zmq4.NewPull(bkg, zmq4.WithLogger(zmq4.Devnull)),
		},
		{
			name:  "xpub",
			srv:   zmq4.NewXPub(bkg, zmq4.WithLogger(zmq4.Devnull)),
			wrong: zmq4.NewPair(bkg, zmq4.WithLogger(zmq4.Devnull)),
			cli:   zmq4.NewXSub(bkg, zmq4.WithLogger(zmq4.Devnull)),
		},
		{
			name:  "xsub",
			srv:   zmq4.NewXSub(bkg, zmq4.WithLogger(zmq4.Devnull)),
			wrong: zmq4.NewPair(bkg, zmq4.WithLogger(zmq4.Devnull)),
			cli:   zmq4.NewXPub(bkg, zmq4.WithLogger(zmq4.Devnull)),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ep := must(EndPoint("tcp"))
			cleanUp(ep)

			_, timeout := context.WithTimeout(bkg, 20*time.Second)
			defer timeout()

			defer tc.srv.Close()
			defer tc.wrong.Close()
			defer tc.cli.Close()

			err := tc.srv.Listen(ep)
			if err != nil {
				t.Fatalf("could not listen on %q: %+v", ep, err)
			}

			err = tc.wrong.Dial(ep)
			if err == nil {
				t.Fatalf("dialed %q", ep)
			}
			want := xerrors.Errorf("zmq4: could not open a ZMTP connection: zmq4: could not initialize ZMTP connection: zmq4: peer=%q not compatible with %q", tc.srv.Type(), tc.wrong.Type())
			if got, want := err.Error(), want.Error(); got != want {
				t.Fatalf("invalid error:\ngot = %v\nwant= %v", got, want)
			}

			err = tc.cli.Dial(ep)
			if err != nil {
				t.Fatalf("could not dial %q: %+v", ep, err)
			}
		})
	}
}
