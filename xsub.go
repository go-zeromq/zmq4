// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"context"
	"net"
)

// NewXSub returns a new XSUB ZeroMQ socket.
// The returned socket value is initially unbound.
func NewXSub(ctx context.Context, opts ...Option) Socket {
	xsub := &xsubSocket{newSocket(ctx, XSub, opts...)}
	return xsub
}

// xsubSocket is a XSUB ZeroMQ socket.
type xsubSocket struct {
	sck *socket
}

// Close closes the open Socket
func (xsub *xsubSocket) Close() error {
	return xsub.sck.Close()
}

// Send puts the message on the outbound send queue.
// Send blocks until the message can be queued or the send deadline expires.
func (xsub *xsubSocket) Send(msg Msg) error {
	return xsub.sck.Send(msg)
}

// SendMulti puts the message on the outbound send queue.
// SendMulti blocks until the message can be queued or the send deadline expires.
// The message will be sent as a multipart message.
func (xsub *xsubSocket) SendMulti(msg Msg) error {
	return xsub.sck.SendMulti(msg)
}

// Recv receives a complete message.
func (xsub *xsubSocket) Recv() (Msg, error) {
	return xsub.sck.Recv()
}

// Listen connects a local endpoint to the Socket.
func (xsub *xsubSocket) Listen(ep string) error {
	return xsub.sck.Listen(ep)
}

// Dial connects a remote endpoint to the Socket.
func (xsub *xsubSocket) Dial(ep string) error {
	return xsub.sck.Dial(ep)
}

// Type returns the type of this Socket (PUB, SUB, ...)
func (xsub *xsubSocket) Type() SocketType {
	return xsub.sck.Type()
}

// Addr returns the listener's address.
// Addr returns nil if the socket isn't a listener.
func (xsub *xsubSocket) Addr() net.Addr {
	return xsub.sck.Addr()
}

// GetOption is used to retrieve an option for a socket.
func (xsub *xsubSocket) GetOption(name string) (interface{}, error) {
	return xsub.sck.GetOption(name)
}

// SetOption is used to set an option for a socket.
func (xsub *xsubSocket) SetOption(name string, value interface{}) error {
	return xsub.sck.SetOption(name, value)
}

var (
	_ Socket = (*xsubSocket)(nil)
)
