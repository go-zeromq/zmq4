// Copyright 2024 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"context"
	"fmt"
	"io"
	"net"
	"runtime"
	"testing"
	"time"
)

func TestConnReaperDeadlock2(t *testing.T) {
	ep := must(EndPoint("tcp"))
	defer cleanUp(ep)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Bind the server.
	srv := NewRouter(ctx, WithLogger(Devnull)).(*routerSocket)
	if err := srv.Listen(ep); err != nil {
		t.Fatalf("could not listen on %q: %+v", ep, err)
	}
	defer srv.Close()

	// Connect clients.
	// Use same client ID so the srv.Send will hold mutex
	// for longer time
	var clients []Socket
	id := "client-x"
	for i := 0; i < 10; i++ {
		c := NewReq(ctx, WithLogger(Devnull), WithID(SocketIdentity(id)))
		if err := c.Dial(ep); err != nil {
			t.Fatalf("could not dial %q: %+v", ep, err)
		}
		clients = append(clients, c)
	}

	// Modify clients connection sockets in server
	// so any send to client will trigger context switch
	// and be failing.
	// Idea is that while srv.Send is progresing,
	// the connection will be closed and assigned
	// for connection reaper.
	rmw := srv.sck.w.(*routerMWriter)
	for i := range rmw.ws {
		rmw.ws[i].rw = &sockSendEof{rmw.ws[i].rw}
	}

	// Now try to send a message from the server to all clients.
	msg := NewMsgFrom(nil, nil, []byte("payload"))
	msg.Frames[0] = []byte(id)
	if err := srv.Send(msg); err != nil {
		t.Logf("Send to %s failed: %+v\n", id, err)
	}

	for i := range clients {
		clients[i].Close()
	}
}

type sockSendEof struct {
	net.Conn
}

func (r *sockSendEof) Write(b []byte) (n int, err error) {
	runtime.Gosched()
	time.Sleep(1 * time.Second)
	return 0, io.EOF
}

func must(str string, err error) string {
	if err != nil {
		panic(err)
	}
	return str
}

func EndPoint(transport string) (string, error) {
	switch transport {
	case "tcp":
		addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
		if err != nil {
			return "", err
		}
		l, err := net.ListenTCP("tcp", addr)
		if err != nil {
			return "", err
		}
		defer l.Close()
		return fmt.Sprintf("tcp://%s", l.Addr()), nil
	case "ipc":
		return "ipc://tmp-" + newUUID(), nil
	case "inproc":
		return "inproc://tmp-" + newUUID(), nil
	default:
		panic("invalid transport: [" + transport + "]")
	}
}
