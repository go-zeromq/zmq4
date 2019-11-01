// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"context"
	"net"
)

// NewRep returns a new REP ZeroMQ socket.
// The returned socket value is initially unbound.
func NewRep(ctx context.Context, opts ...Option) Socket {
	rep := &repSocket{newSocket(ctx, Rep, opts...)}
	return rep
}

// repSocket is a REP ZeroMQ socket.
type repSocket struct {
	sck *socket
}

// Close closes the open Socket
func (rep *repSocket) Close() error {
	return rep.sck.Close()
}

// Send puts the message on the outbound send queue.
// Send blocks until the message can be queued or the send deadline expires.
func (rep *repSocket) Send(msg Msg) error {
	msg.Frames = append([][]byte{nil}, msg.Frames...)
	return rep.sck.Send(msg)
}

// SendMulti puts the message on the outbound send queue.
// SendMulti blocks until the message can be queued or the send deadline expires.
// The message will be sent as a multipart message.
func (rep *repSocket) SendMulti(msg Msg) error {
	msg.Frames = append([][]byte{nil}, msg.Frames...)
	return rep.sck.SendMulti(msg)
}

// Recv receives a complete message.
func (rep *repSocket) Recv() (Msg, error) {
	msg, err := rep.sck.Recv()
	if len(msg.Frames) > 1 {
		msg.Frames = msg.Frames[1:]
	}
	return msg, err
}

// Listen connects a local endpoint to the Socket.
func (rep *repSocket) Listen(ep string) error {
	return rep.sck.Listen(ep)
}

// Dial connects a remote endpoint to the Socket.
func (rep *repSocket) Dial(ep string) error {
	return rep.sck.Dial(ep)
}

// Type returns the type of this Socket (PUB, SUB, ...)
func (rep *repSocket) Type() SocketType {
	return rep.sck.Type()
}

// Addr returns the listener's address.
// Addr returns nil if the socket isn't a listener.
func (rep *repSocket) Addr() net.Addr {
	return rep.sck.Addr()
}

// GetOption is used to retrieve an option for a socket.
func (rep *repSocket) GetOption(name string) (interface{}, error) {
	return rep.sck.GetOption(name)
}

// SetOption is used to set an option for a socket.
func (rep *repSocket) SetOption(name string, value interface{}) error {
	return rep.sck.SetOption(name, value)
}

var (
	_ Socket = (*repSocket)(nil)
)
