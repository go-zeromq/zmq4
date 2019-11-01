// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"context"
	"net"
)

// NewPair returns a new PAIR ZeroMQ socket.
// The returned socket value is initially unbound.
func NewPair(ctx context.Context, opts ...Option) Socket {
	pair := &pairSocket{newSocket(ctx, Pair, opts...)}
	return pair
}

// pairSocket is a PAIR ZeroMQ socket.
type pairSocket struct {
	sck *socket
}

// Close closes the open Socket
func (pair *pairSocket) Close() error {
	return pair.sck.Close()
}

// Send puts the message on the outbound send queue.
// Send blocks until the message can be queued or the send deadline expires.
func (pair *pairSocket) Send(msg Msg) error {
	return pair.sck.Send(msg)
}

// SendMulti puts the message on the outbound send queue.
// SendMulti blocks until the message can be queued or the send deadline expires.
// The message will be sent as a multipart message.
func (pair *pairSocket) SendMulti(msg Msg) error {
	return pair.sck.SendMulti(msg)
}

// Recv receives a complete message.
func (pair *pairSocket) Recv() (Msg, error) {
	return pair.sck.Recv()
}

// Listen connects a local endpoint to the Socket.
func (pair *pairSocket) Listen(ep string) error {
	return pair.sck.Listen(ep)
}

// Dial connects a remote endpoint to the Socket.
func (pair *pairSocket) Dial(ep string) error {
	return pair.sck.Dial(ep)
}

// Type returns the type of this Socket (PUB, SUB, ...)
func (pair *pairSocket) Type() SocketType {
	return pair.sck.Type()
}

// Addr returns the listener's address.
// Addr returns nil if the socket isn't a listener.
func (pair *pairSocket) Addr() net.Addr {
	return pair.sck.Addr()
}

// GetOption is used to retrieve an option for a socket.
func (pair *pairSocket) GetOption(name string) (interface{}, error) {
	return pair.sck.GetOption(name)
}

// SetOption is used to set an option for a socket.
func (pair *pairSocket) SetOption(name string, value interface{}) error {
	return pair.sck.SetOption(name, value)
}

var (
	_ Socket = (*pairSocket)(nil)
)
