// Copyright 2024 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"context"
	"io"
	"net"
	"sync/atomic"
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
	for i := 0; i < 2; i++ {
		w := &Conn{}
		w.Peer.Meta = make(Metadata)
		w.Peer.Meta[sysSockID] = id
		w.rw = &sockSendEOF{}
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

type sockSendEOF struct {
}

var a atomic.Int32

func (r *sockSendEOF) Write(b []byte) (n int, err error) {
	// Each odd write fails asap.
	// Each even write fails after sleep.
	// Such a way we ensure the short write failure
	// will cause socket be assinged to connection reaper
	// while srv.Send is still in progress due to long writes.
	if x := a.Add(1); x&1 == 0 {
		time.Sleep(1 * time.Second)
	}
	return 0, io.EOF
}

func (r *sockSendEOF) Read(b []byte) (int, error) {
	return 0, nil
}

func (r *sockSendEOF) Close() error {
	return nil
}

func (r *sockSendEOF) LocalAddr() net.Addr {
	return nil
}

func (r *sockSendEOF) RemoteAddr() net.Addr {
	return nil
}

func (r *sockSendEOF) SetDeadline(t time.Time) error {
	return nil
}

func (r *sockSendEOF) SetReadDeadline(t time.Time) error {
	return nil
}

func (r *sockSendEOF) SetWriteDeadline(t time.Time) error {
	return nil
}
