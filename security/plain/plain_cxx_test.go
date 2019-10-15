// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build czmq4

package plain_test

import (
	"context"
	"os"
	"reflect"
	"testing"
	"time"

	czmq4 "github.com/go-zeromq/goczmq/v4"
	"github.com/go-zeromq/zmq4"
	"github.com/go-zeromq/zmq4/security/plain"
	"golang.org/x/sync/errgroup"
	"golang.org/x/xerrors"
)

func TestMain(m *testing.M) {
	auth := czmq4.NewAuth()

	err := auth.Allow("127.0.0.1")
	if err != nil {
		auth.Destroy()
		panic(err)
	}

	err = auth.Plain("./testdata/password.txt")
	if err != nil {
		auth.Destroy()
		panic(err)
	}

	// call flag.Parse() here if TestMain uses flags

	exit := m.Run()

	auth.Destroy()
	os.Exit(exit)
}

func TestHandshakeReqCRep(t *testing.T) {
	t.Skipf("REQ-CREP")

	sec := plain.Security("user", "secret")
	if got, want := sec.Type(), zmq4.PlainSecurity; got != want {
		t.Fatalf("got=%v, want=%v", got, want)
	}

	ctx, timeout := context.WithTimeout(context.Background(), 10*time.Second)
	defer timeout()

	ep := must(EndPoint("tcp"))

	req := zmq4.NewReq(ctx, zmq4.WithSecurity(sec))
	defer req.Close()

	rep := zmq4.NewCRep(ctx, czmq4.SockSetZapDomain("global"), czmq4.SockSetPlainServer(1))
	defer rep.Close()

	grp, ctx := errgroup.WithContext(ctx)
	grp.Go(func() error {
		err := rep.Listen(ep)
		if err != nil {
			return xerrors.Errorf("could not listen: %w", err)
		}

		msg, err := rep.Recv()
		if err != nil {
			return xerrors.Errorf("could not recv REQ message: %w", err)
		}

		if !reflect.DeepEqual(msg, reqQuit) {
			return xerrors.Errorf("got = %v, want = %v", msg, repQuit)
		}

		err = rep.Send(repQuit)
		if err != nil {
			return xerrors.Errorf("could not send REP message: %w", err)
		}

		return nil
	})

	grp.Go(func() error {
		err := req.Dial(ep)
		if err != nil {
			return xerrors.Errorf("could not dial: %w", err)
		}

		err = req.Send(reqQuit)
		if err != nil {
			return xerrors.Errorf("could not send REQ message: %w", err)
		}
		msg, err := req.Recv()
		if err != nil {
			return xerrors.Errorf("could not recv REQ message: %w", err)
		}

		if !reflect.DeepEqual(msg, repQuit) {
			return xerrors.Errorf("got = %v, want = %v", msg, repQuit)
		}
		return nil
	})

	if err := grp.Wait(); err != nil {
		t.Fatalf("error: %+v", err)
	}
}

func TestHandshakeCReqRep(t *testing.T) {
	t.Skipf("CREQ-REP")

	sec := plain.Security("user", "secret")
	if got, want := sec.Type(), zmq4.PlainSecurity; got != want {
		t.Fatalf("got=%v, want=%v", got, want)
	}

	ctx, timeout := context.WithTimeout(context.Background(), 10*time.Second)
	defer timeout()

	ep := must(EndPoint("tcp"))

	req := zmq4.NewCReq(ctx, czmq4.SockSetPlainUsername("user"), czmq4.SockSetPlainPassword("secret"))
	defer req.Close()

	rep := zmq4.NewRep(ctx, zmq4.WithSecurity(sec))
	defer rep.Close()

	grp, ctx := errgroup.WithContext(ctx)
	grp.Go(func() error {
		err := rep.Listen(ep)
		if err != nil {
			return xerrors.Errorf("could not listen: %w", err)
		}

		msg, err := rep.Recv()
		if err != nil {
			return xerrors.Errorf("could not recv REQ message: %w", err)
		}

		if !reflect.DeepEqual(msg, reqQuit) {
			return xerrors.Errorf("got = %v, want = %v", msg, repQuit)
		}

		err = rep.Send(repQuit)
		if err != nil {
			return xerrors.Errorf("could not send REP message: %w", err)
		}

		return nil
	})

	grp.Go(func() error {
		err := req.Dial(ep)
		if err != nil {
			return xerrors.Errorf("could not dial: %w", err)
		}

		err = req.Send(reqQuit)
		if err != nil {
			return xerrors.Errorf("could not send REQ message: %w", err)
		}
		msg, err := req.Recv()
		if err != nil {
			return xerrors.Errorf("could not recv REQ message: %w", err)
		}

		if !reflect.DeepEqual(msg, repQuit) {
			return xerrors.Errorf("got = %v, want = %v", msg, repQuit)
		}
		return nil
	})

	if err := grp.Wait(); err != nil {
		t.Fatalf("error: %+v", err)
	}
}

func TestHandshakeCReqCRep(t *testing.T) {
	t.Skipf("CREQ-CREP")

	sec := plain.Security("user", "secret")
	if got, want := sec.Type(), zmq4.PlainSecurity; got != want {
		t.Fatalf("got=%v, want=%v", got, want)
	}

	ctx, timeout := context.WithTimeout(context.Background(), 10*time.Second)
	defer timeout()

	ep := must(EndPoint("tcp"))

	req := zmq4.NewCReq(ctx, czmq4.SockSetPlainUsername("user"), czmq4.SockSetPlainPassword("secret"))
	defer req.Close()

	rep := zmq4.NewCRep(ctx, czmq4.SockSetZapDomain("global"), czmq4.SockSetPlainServer(1))
	defer rep.Close()

	grp, ctx := errgroup.WithContext(ctx)
	grp.Go(func() error {
		err := rep.Listen(ep)
		if err != nil {
			return xerrors.Errorf("could not listen: %w", err)
		}

		msg, err := rep.Recv()
		if err != nil {
			return xerrors.Errorf("could not recv REQ message: %w", err)
		}

		if !reflect.DeepEqual(msg, reqQuit) {
			return xerrors.Errorf("got = %v, want = %v", msg, repQuit)
		}

		err = rep.Send(repQuit)
		if err != nil {
			return xerrors.Errorf("could not send REP message: %w", err)
		}

		return nil
	})

	grp.Go(func() error {
		err := req.Dial(ep)
		if err != nil {
			return xerrors.Errorf("could not dial: %w", err)
		}

		err = req.Send(reqQuit)
		if err != nil {
			return xerrors.Errorf("could not send REQ message: %w", err)
		}
		msg, err := req.Recv()
		if err != nil {
			return xerrors.Errorf("could not recv REQ message: %w", err)
		}

		if !reflect.DeepEqual(msg, repQuit) {
			return xerrors.Errorf("got = %v, want = %v", msg, repQuit)
		}
		return nil
	})

	if err := grp.Wait(); err != nil {
		t.Fatalf("error: %+v", err)
	}
}
