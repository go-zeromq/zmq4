// Copyright 2020 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/go-zeromq/zmq4"
	"github.com/go-zeromq/zmq4/transport"
	"golang.org/x/sync/errgroup"
)

func TestInvalidConn(t *testing.T) {
	// t.Parallel()

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
			return fmt.Errorf("could not dial %q: %w", ep, err)
		}
		defer conn.Close()
		var reply = make([]byte, 64)
		_, err = io.ReadFull(conn, reply)
		if err != nil {
			return fmt.Errorf("could not read reply bytes...: %w", err)
		}
		_, err = conn.Write(make([]byte, 64))
		if err != nil {
			return fmt.Errorf("could not send bytes...: %w", err)
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
	// t.Parallel()

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
			want := fmt.Errorf("zmq4: could not open a ZMTP connection: zmq4: could not initialize ZMTP connection: zmq4: peer=%q not compatible with %q", tc.srv.Type(), tc.wrong.Type())
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

func TestConnReaperDeadlock(t *testing.T) {
	// Should avoid deadlock when multiple clients are closed rapidly.

	ep := must(EndPoint("tcp"))
	defer cleanUp(ep)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Bind the server.
	srv := zmq4.NewRouter(ctx, zmq4.WithLogger(zmq4.Devnull))
	if err := srv.Listen(ep); err != nil {
		t.Fatalf("could not listen on %q: %+v", ep, err)
	}
	defer srv.Close()

	// Connect 10 clients.
	var clients []zmq4.Socket
	for i := 0; i < 10; i++ {
		id := fmt.Sprint("client-", i)
		c := zmq4.NewReq(ctx, zmq4.WithLogger(zmq4.Devnull), zmq4.WithID(zmq4.SocketIdentity(id)))
		if err := c.Dial(ep); err != nil {
			t.Fatalf("could not dial %q: %+v", ep, err)
		}
		clients = append(clients, c)
	}

	// Disconnect 5 of them _from the client side_. The server does not know
	// the client is gone until it tries to send a message below.
	for i := 0; i < 5; i++ {
		clients[i].Close()
	}

	// Now try to send a message from the server to all 10 clients.
	msg := zmq4.NewMsgFrom(nil, nil, []byte("payload"))
	for i := range clients {
		id := fmt.Sprint("client-", i)
		msg.Frames[0] = []byte(id)
		if err := srv.Send(msg); err != nil {
			t.Logf("Send to %s failed: %+v\n", id, err)
		}
	}

	for i := 5; i < 10; i++ {
		clients[i].Close()
	}
}

func TestSocketSendSubscriptionOnConnect(t *testing.T) {
	endpoint := "inproc://test-resub"
	message := "test"

	sub := zmq4.NewSub(context.Background())
	defer sub.Close()
	pub := zmq4.NewPub(context.Background())
	defer pub.Close()
	sub.SetOption(zmq4.OptionSubscribe, message)
	if err := sub.Listen(endpoint); err != nil {
		t.Fatalf("Sub Dial failed: %v", err)
	}
	if err := pub.Dial(endpoint); err != nil {
		t.Fatalf("Pub Dial failed: %v", err)
	}
	wg := new(sync.WaitGroup)
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			pub.Send(zmq4.NewMsgFromString([]string{message}))
			if ctx.Err() != nil {
				return
			}
			time.Sleep(1 * time.Millisecond)
		}
	}()
	msg, err := sub.Recv()
	if err != nil {
		t.Fatalf("Recv failed: %v", err)
	}
	if string(msg.Frames[0]) != message {
		t.Fatalf("invalid message received: got '%s', wanted '%s'", msg.Frames[0], message)
	}
}

type transportMock struct {
	dialCalledCount int
	errOnDial       bool
	conn            net.Conn
}

func (t *transportMock) Dial(ctx context.Context, dialer transport.Dialer, addr string) (net.Conn, error) {
	t.dialCalledCount++
	if t.errOnDial {
		return nil, errors.New("test error")
	}
	return t.conn, nil
}

func (t *transportMock) Listen(ctx context.Context, addr string) (net.Listener, error) {
	return nil, nil
}

func (t *transportMock) Addr(ep string) (addr string, err error) {
	return "", nil
}

func TestConnMaxRetries(t *testing.T) {
	retryCount := 123
	socket := zmq4.NewSub(context.Background(), zmq4.WithDialerRetry(time.Microsecond), zmq4.WithDialerMaxRetries(retryCount))
	transport := &transportMock{errOnDial: true}
	transportName := "test-maxretries"
	zmq4.RegisterTransport(transportName, transport)
	err := socket.Dial(transportName + "://test")

	if err == nil {
		t.Fatal("expected error")
	}

	if transport.dialCalledCount != retryCount+1 {
		t.Fatalf("Dial called %d times, expected %d", transport.dialCalledCount, retryCount+1)
	}
}

func TestConnMaxRetriesInfinite(t *testing.T) {
	timeout := time.Millisecond
	retryTime := time.Nanosecond

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	socket := zmq4.NewSub(ctx, zmq4.WithDialerRetry(retryTime), zmq4.WithDialerMaxRetries(-1))
	transport := &transportMock{errOnDial: true}
	transportName := "test-infiniteretries"
	zmq4.RegisterTransport(transportName, transport)
	err := socket.Dial(transportName + "://test")
	if err == nil {
		t.Fatal("expected error")
	}

	atLeastExpectedRetries := 100
	if transport.dialCalledCount < atLeastExpectedRetries {
		t.Fatalf("Dial called %d times, expected  at least %d", transport.dialCalledCount, atLeastExpectedRetries)
	}
}
