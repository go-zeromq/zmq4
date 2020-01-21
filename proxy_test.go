// Copyright 2020 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/go-zeromq/zmq4"
	"golang.org/x/sync/errgroup"
	"golang.org/x/xerrors"
)

func TestProxy(t *testing.T) {
	bkg := context.Background()
	ctx, timeout := context.WithTimeout(bkg, 20*time.Second)
	defer timeout()

	var (
		frontIn = zmq4.NewPush(ctx, zmq4.WithLogger(zmq4.Devnull))
		front   = zmq4.NewPull(ctx, zmq4.WithLogger(zmq4.Devnull))
		back    = zmq4.NewPush(ctx, zmq4.WithLogger(zmq4.Devnull))
		backOut = zmq4.NewPull(ctx, zmq4.WithLogger(zmq4.Devnull))
		capt    = zmq4.NewPush(ctx, zmq4.WithLogger(zmq4.Devnull))
		captOut = zmq4.NewPull(ctx, zmq4.WithLogger(zmq4.Devnull))

		proxy *zmq4.Proxy

		epFront = "ipc://proxy-front"
		epBack  = "ipc://proxy-back"
		epCapt  = "ipc://proxy-capt"

		wg1 sync.WaitGroup // all sockets ready
		wg2 sync.WaitGroup // proxy setup
		wg3 sync.WaitGroup // all messages received
		wg4 sync.WaitGroup // all capture messages received
		wg5 sync.WaitGroup // terminate sent
		wg6 sync.WaitGroup // all sockets done
	)

	wg1.Add(6) // number of sockets
	wg2.Add(1) // proxy ready
	wg3.Add(1) // messages received at backout
	wg4.Add(1) // capture messages received at capt-out
	wg5.Add(1) // terminate
	wg6.Add(6) // number of sockets

	cleanUp(epFront)
	cleanUp(epBack)
	cleanUp(epCapt)

	var (
		msgs = []zmq4.Msg{
			zmq4.NewMsgFrom([]byte("msg1")),
			zmq4.NewMsgFrom([]byte("msg2")),
			zmq4.NewMsgFrom([]byte("msg3")),
			zmq4.NewMsgFrom([]byte("msg4")),
		}
	)

	grp, ctx := errgroup.WithContext(ctx)
	grp.Go(func() error {
		defer frontIn.Close()
		err := frontIn.Dial(epFront)
		if err != nil {
			return xerrors.Errorf("front-in could not dial %q: %w", epFront, err)
		}

		wg1.Done()
		t.Logf("front-in ready")
		wg1.Wait() // sockets
		wg2.Wait() // proxy

		for _, msg := range msgs {
			t.Logf("front-in sending %v...", msg)
			err = frontIn.Send(msg)
			if err != nil {
				return xerrors.Errorf("could not send front-in %q: %w", msg, err)
			}
			t.Logf("front-in sending %v... [done]", msg)
		}

		wg3.Wait() // all messages received
		wg4.Wait() // all capture messages received
		t.Logf("front-in waiting for terminate signal")
		wg5.Wait() // terminate

		wg6.Done() // all sockets done
		wg6.Wait()
		return nil
	})

	grp.Go(func() error {
		defer front.Close()
		err := front.Listen(epFront)
		if err != nil {
			return xerrors.Errorf("front could not listen %q: %w", epFront, err)
		}

		wg1.Done()
		t.Logf("front ready")
		wg1.Wait() // sockets
		wg2.Wait() // proxy
		wg3.Wait() // all messages received
		wg4.Wait() // all capture messages received
		t.Logf("front waiting for terminate signal")
		wg5.Wait() // terminate

		wg6.Done() // all sockets done
		wg6.Wait()
		return nil
	})

	grp.Go(func() error {
		defer back.Close()
		err := back.Listen(epBack)
		if err != nil {
			return xerrors.Errorf("back could not listen %q: %w", epBack, err)
		}

		wg1.Done()
		t.Logf("back ready")
		wg1.Wait() // sockets
		wg2.Wait() // proxy
		wg3.Wait() // all messages received
		wg4.Wait() // all capture messages received
		t.Logf("back waiting for terminate signal")
		wg5.Wait() // terminate

		wg6.Done() // all sockets done
		wg6.Wait()
		return nil
	})

	grp.Go(func() error {
		defer backOut.Close()
		err := backOut.Dial(epBack)
		if err != nil {
			return xerrors.Errorf("back-out could not dial %q: %w", epBack, err)
		}

		wg1.Done()
		t.Logf("back-out ready")
		wg1.Wait() // sockets
		wg2.Wait() // proxy

		for _, want := range msgs {
			t.Logf("back-out recving %v...", want)
			msg, err := backOut.Recv()
			if err != nil {
				return xerrors.Errorf("back-out could not recv: %w", err)
			}
			if msg.String() != want.String() {
				return xerrors.Errorf("invalid message: got=%v, want=%v", msg, want)
			}
			t.Logf("back-out recving %v... [done]", msg)
		}

		wg3.Done() // all messages received
		wg3.Wait() // all messages received
		wg4.Wait() // all capture messages received
		t.Logf("back-out waiting for terminate signal")
		wg5.Wait() // terminate

		wg6.Done() // all sockets done
		wg6.Wait()
		return nil
	})

	grp.Go(func() error {
		defer captOut.Close()
		err := captOut.Listen(epCapt)
		if err != nil {
			return xerrors.Errorf("capt-out could not listen %q: %w", epCapt, err)
		}

		wg1.Done()
		t.Logf("capt-out ready")
		wg1.Wait() // sockets
		wg2.Wait() // proxy
		wg3.Wait() // all messages received

		for _, want := range msgs {
			t.Logf("capt-out recving %v...", want)
			msg, err := captOut.Recv()
			if err != nil {
				return xerrors.Errorf("capt-out could not recv msg: %w", err)
			}
			if msg.String() != want.String() {
				return xerrors.Errorf("capt-out: invalid message: got=%v, want=%v", msg, want)
			}
			t.Logf("capt-out recving %v... [done]", msg)
		}

		wg4.Done() // all capture messages received
		wg4.Wait() // all capture messages received
		t.Logf("capt-out waiting for terminate signal")
		wg5.Wait() // terminate

		wg6.Done() // all sockets done
		wg6.Wait()
		return nil
	})

	grp.Go(func() error {
		defer capt.Close()
		err := capt.Dial(epCapt)
		if err != nil {
			return xerrors.Errorf("capt could not dial %q: %w", epCapt, err)
		}

		wg1.Done()
		t.Logf("capt ready")
		wg1.Wait() // sockets
		wg2.Wait() // proxy
		wg3.Wait() // all messages received
		wg4.Wait() // all capture messages received
		t.Logf("capt waiting for terminate signal")
		wg5.Wait() // terminate

		wg6.Done() // all sockets done
		wg6.Wait()
		return nil
	})

	grp.Go(func() error {
		t.Logf("ctrl ready")
		wg1.Wait() // sockets
		wg2.Wait() // proxy
		for _, cmd := range []struct {
			name string
			fct  func()
		}{
			{"pause", proxy.Pause},
			{"resume", proxy.Resume},
			{"stats", proxy.Stats},
		} {
			t.Logf("ctrl sending %v...", cmd.name)
			cmd.fct()
			t.Logf("ctrl sending %v... [done]", cmd.name)
		}
		wg3.Wait() // all messages received
		wg4.Wait() // all capture messages received

		t.Logf("ctrl sending kill...")
		proxy.Kill()
		t.Logf("ctrl sending kill... [done]")

		wg5.Done()
		t.Logf("ctrl waiting for terminate signal")
		wg5.Wait() // terminate

		wg6.Wait()
		return nil
	})

	grp.Go(func() error {
		wg1.Wait() // sockets ready
		proxy = zmq4.NewProxy(ctx, front, back, capt)
		t.Logf("proxy ready")
		wg2.Done()
		err := proxy.Run()
		t.Logf("proxy done: err=%+v", err)
		return err
	})

	if err := grp.Wait(); err != nil {
		t.Fatalf("error: %+v", err)
	}

	if err := ctx.Err(); err != nil && err != context.Canceled {
		t.Fatalf("error: %+v", err)
	}
}

func TestProxyStop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	var (
		epFront = "ipc://proxy-stop-front"
		epBack  = "ipc://proxy-stop-back"

		frontIn = zmq4.NewPush(ctx, zmq4.WithLogger(zmq4.Devnull))
		front   = zmq4.NewPull(ctx, zmq4.WithLogger(zmq4.Devnull))
		back    = zmq4.NewPush(ctx, zmq4.WithLogger(zmq4.Devnull))
		backOut = zmq4.NewPull(ctx, zmq4.WithLogger(zmq4.Devnull))
	)

	cleanUp(epFront)
	cleanUp(epBack)

	defer front.Close()
	defer back.Close()

	if err := front.Listen(epFront); err != nil {
		t.Fatalf("could not listen: %+v", err)
	}

	if err := frontIn.Dial(epFront); err != nil {
		t.Fatalf("could not dial: %+v", err)
	}

	if err := back.Listen(epBack); err != nil {
		t.Fatalf("could not listen: %+v", err)
	}

	if err := backOut.Dial(epBack); err != nil {
		t.Fatalf("could not dial: %+v", err)
	}

	var errc = make(chan error)
	go func() {
		errc <- zmq4.NewProxy(ctx, front, back, nil).Run()
	}()

	go func() {
		_ = frontIn.Send(zmq4.NewMsgString("msg1"))
	}()
	go func() {
		_, _ = backOut.Recv()
	}()
	cancel()

	err := <-errc
	if err != context.Canceled {
		t.Fatalf("error: %+v", err)
	}

	if err := ctx.Err(); err != nil && err != context.Canceled {
		t.Fatalf("error: %+v", err)
	}
}
