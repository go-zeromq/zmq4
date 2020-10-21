// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"context"
	"fmt"
	"net"
)

// NewPull returns a new PULL ZeroMQ socket.
// The returned socket value is initially unbound.
func NewPull(ctx context.Context, opts ...Option) Socket {
	pull := &pullSocket{newSocket(ctx, Pull, opts...)}
	pull.sck.w = nil
	return pull
}

// pullSocket is a PULL ZeroMQ socket.
type pullSocket struct {
	sck *socket
}

// Close closes the open Socket
func (pull *pullSocket) Close() error {
	return pull.sck.Close()
}

// Send puts the message on the outbound send queue.
// Send blocks until the message can be queued or the send deadline expires.
func (*pullSocket) Send(msg Msg) error {
	return fmt.Errorf("zmq4: PULL sockets can't send messages")
}

// SendMulti puts the message on the outbound send queue.
// SendMulti blocks until the message can be queued or the send deadline expires.
// The message will be sent as a multipart message.
func (pull *pullSocket) SendMulti(msg Msg) error {
	return fmt.Errorf("zmq4: PULL sockets can't send messages")
}

// Recv receives a complete message.
func (pull *pullSocket) Recv() (Msg, error) {
	return pull.sck.Recv()
}

// Listen connects a local endpoint to the Socket.
func (pull *pullSocket) Listen(ep string) error {
	return pull.sck.Listen(ep)
}

// Dial connects a remote endpoint to the Socket.
func (pull *pullSocket) Dial(ep string) error {
	return pull.sck.Dial(ep)
}

// Type returns the type of this Socket (PUB, SUB, ...)
func (pull *pullSocket) Type() SocketType {
	return pull.sck.Type()
}

// Addr returns the listener's address.
// Addr returns nil if the socket isn't a listener.
func (pull *pullSocket) Addr() net.Addr {
	return pull.sck.Addr()
}

// GetOption is used to retrieve an option for a socket.
func (pull *pullSocket) GetOption(name string) (interface{}, error) {
	return pull.sck.GetOption(name)
}

// SetOption is used to set an option for a socket.
func (pull *pullSocket) SetOption(name string, value interface{}) error {
	return pull.sck.SetOption(name, value)
}

var (
	_ Socket = (*pullSocket)(nil)
)
