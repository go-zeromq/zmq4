// Copyright 2026 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4_test

import (
	"context"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/go-zeromq/zmq4"
)

func TestAcceptedHandshakeTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server := zmq4.NewPair(ctx, zmq4.WithTimeout(200*time.Millisecond), zmq4.WithLogger(zmq4.Devnull))
	defer server.Close()
	if err := server.Listen("tcp://127.0.0.1:0"); err != nil {
		t.Fatal(err)
	}
	endpoint := fmt.Sprintf("tcp://%s", server.Addr())
	stalled, err := net.Dial("tcp", server.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer stalled.Close()
	if err := stalled.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	// The server sends its greeting, but this peer never sends one back.
	if _, err := io.ReadFull(stalled, make([]byte, 64)); err != nil {
		t.Fatal(err)
	}
	if _, err := stalled.Read(make([]byte, 1)); err != io.EOF {
		t.Fatalf("stalled handshake connection was not closed: %v", err)
	}

	// Once the timed-out peer is gone, a regular peer can connect.
	client := zmq4.NewPair(ctx, zmq4.WithTimeout(time.Second))
	defer client.Close()
	if err := client.Dial(endpoint); err != nil {
		t.Fatal(err)
	}
	// A successful handshake must clear its deadline before normal I/O.
	time.Sleep(400 * time.Millisecond)
	if err := client.Send(zmq4.NewMsgString("after handshake deadline")); err != nil {
		t.Fatal(err)
	}
	msg, err := server.Recv()
	if err != nil {
		t.Fatal(err)
	}
	if got := string(msg.Frames[0]); got != "after handshake deadline" {
		t.Fatalf("got %q", got)
	}
}
