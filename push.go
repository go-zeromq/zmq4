// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"context"
	"fmt"
	"net"
)

// NewPush returns a new PUSH ZeroMQ socket.
// The returned socket value is initially unbound.
func NewPush(ctx context.Context, opts ...Option) Socket {
	push := &pushSocket{newSocket(ctx, Push, opts...)}
	push.sck.r = nil
	return push
}

// pushSocket is a PUSH ZeroMQ socket.
type pushSocket struct {
	sck *socket
}

// Close closes the open Socket
func (push *pushSocket) Close() error {
	return push.sck.Close()
}

// Send puts the message on the outbound send queue.
// Send blocks until the message can be queued or the send deadline expires.
func (push *pushSocket) Send(msg Msg) error {
	return push.sck.Send(msg)
}

// SendMulti puts the message on the outbound send queue.
// SendMulti blocks until the message can be queued or the send deadline expires.
// The message will be sent as a multipart message.
func (push *pushSocket) SendMulti(msg Msg) error {
	return push.sck.SendMulti(msg)
}

// Recv receives a complete message.
func (*pushSocket) Recv() (Msg, error) {
	return Msg{}, fmt.Errorf("zmq4: PUSH sockets can't recv messages")
}

// Listen connects a local endpoint to the Socket.
func (push *pushSocket) Listen(ep string) error {
	return push.sck.Listen(ep)
}

// Dial connects a remote endpoint to the Socket.
func (push *pushSocket) Dial(ep string) error {
	return push.sck.Dial(ep)
}

// Type returns the type of this Socket (PUB, SUB, ...)
func (push *pushSocket) Type() SocketType {
	return push.sck.Type()
}

// Addr returns the listener's address.
// Addr returns nil if the socket isn't a listener.
func (push *pushSocket) Addr() net.Addr {
	return push.sck.Addr()
}

// GetOption is used to retrieve an option for a socket.
func (push *pushSocket) GetOption(name string) (interface{}, error) {
	return push.sck.GetOption(name)
}

// SetOption is used to set an option for a socket.
func (push *pushSocket) SetOption(name string, value interface{}) error {
	return push.sck.SetOption(name, value)
}

var (
	_ Socket = (*pushSocket)(nil)
)
