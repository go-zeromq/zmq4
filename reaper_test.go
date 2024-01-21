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

	// Add modified clients connection to server
	// so any send to client will trigger context switch
	// and be failing.
	// Idea is that while srv.Send is progressing,
	// the connection will be closed and assigned
	// for connection reaper, and reaper will try to remove those
	id := "client-x"
	srv.sck.mu.Lock()
	rmw := srv.sck.w.(*routerMWriter)
	for i := 0; i < 10; i++ {
		w := &Conn{}
		w.Peer.Meta = make(Metadata)
		w.Peer.Meta[sysSockID] = id
		w.rw = &sockSendEof{}
		w.onCloseErrorCB = srv.sck.scheduleRmConn
		// Do not to call srv.addConn as we dont want to have listener on this fake socket
		rmw.addConn(w)
		srv.sck.conns = append(srv.sck.conns, w)
	}
	srv.sck.mu.Unlock()

	// Now try to send a message from the server to all clients.
	msg := NewMsgFrom(nil, nil, []byte("payload"))
	msg.Frames[0] = []byte(id)
	if err := srv.Send(msg); err != nil {
		t.Logf("Send to %s failed: %+v\n", id, err)
	}
}

type sockSendEof struct {
}

func (r *sockSendEof) Write(b []byte) (n int, err error) {
	runtime.Gosched()
	time.Sleep(1 * time.Second)
	return 0, io.EOF
}

func (r *sockSendEof) Read(b []byte) (int, error) {
	return 0, nil
}

func (r *sockSendEof) Close() error {
	return nil
}

func (r *sockSendEof) LocalAddr() net.Addr {
	return nil
}

func (r *sockSendEof) RemoteAddr() net.Addr {
	return nil
}

func (r *sockSendEof) SetDeadline(t time.Time) error {
	return nil
}

func (r *sockSendEof) SetReadDeadline(t time.Time) error {
	return nil
}

func (r *sockSendEof) SetWriteDeadline(t time.Time) error {
	return nil
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
