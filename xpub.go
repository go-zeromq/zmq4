// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"context"

	"net"

	"golang.org/x/xerrors"
)

// NewXPub returns a new XPUB ZeroMQ socket.
// The returned socket value is initially unbound.
func NewXPub(ctx context.Context, opts ...Option) Socket {
	xpub := &xpubSocket{newSocket(ctx, XPub, opts...)}
	return xpub
}

// xpubSocket is a XPUB ZeroMQ socket.
type xpubSocket struct {
	sck *socket
}

// Close closes the open Socket
func (xpub *xpubSocket) Close() error {
	return xpub.sck.Close()
}

// Send puts the message on the outbound send queue.
// Send blocks until the message can be queued or the send deadline expires.
func (xpub *xpubSocket) Send(msg Msg) error {
	return xpub.sck.Send(msg)
}

// Recv receives a complete message.
func (xpub *xpubSocket) Recv() (Msg, error) {
	return xpub.sck.Recv()
}

// Listen connects a local endpoint to the Socket.
func (xpub *xpubSocket) Listen(ep string) error {
	return xpub.sck.Listen(ep)
}

// Dial connects a remote endpoint to the Socket.
func (xpub *xpubSocket) Dial(ep string) error {
	return xpub.sck.Dial(ep)
}

// Type returns the type of this Socket (PUB, SUB, ...)
func (xpub *xpubSocket) Type() SocketType {
	return xpub.sck.Type()
}

// Addr returns the listener's address.
// Addr returns nil if the socket isn't a listener.
func (xpub *xpubSocket) Addr() net.Addr {
	return xpub.sck.Addr()
}

// GetOption is used to retrieve an option for a socket.
func (xpub *xpubSocket) GetOption(name string) (interface{}, error) {
	return xpub.sck.GetOption(name)
}

// SetOption is used to set an option for a socket.
func (xpub *xpubSocket) SetOption(name string, value interface{}) error {
	return xpub.sck.SetOption(name, value)
}

// GetTopics is used to retrieve subscribed topics for a pub socket.
func (xpub *xpubSocket) GetTopics(filter bool) ([]string, error) {
	err := xerrors.Errorf("zmq4: Only available for PUB sockets")
	return nil, err
}

var (
	_ Socket = (*xpubSocket)(nil)
)
