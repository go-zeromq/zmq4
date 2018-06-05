// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"context"
)

// NewRouter returns a new ROUTER ZeroMQ socket.
// The returned socket value is initially unbound.
func NewRouter(ctx context.Context, opts ...Option) Socket {
	router := &routerSocket{newSocket(ctx, Router, opts...)}
	return router
}

// routerSocket is a ROUTER ZeroMQ socket.
type routerSocket struct {
	sck *socket
}

// Close closes the open Socket
func (router *routerSocket) Close() error {
	return router.sck.Close()
}

// Send puts the message on the outbound send queue.
// Send blocks until the message can be queued or the send deadline expires.
func (router *routerSocket) Send(msg Msg) error {
	return router.sck.Send(msg)
}

// Recv receives a complete message.
func (router *routerSocket) Recv() (Msg, error) {
	return router.sck.Recv()
}

// Listen connects a local endpoint to the Socket.
func (router *routerSocket) Listen(ep string) error {
	return router.sck.Listen(ep)
}

// Dial connects a remote endpoint to the Socket.
func (router *routerSocket) Dial(ep string) error {
	return router.sck.Dial(ep)
}

// Type returns the type of this Socket (PUB, SUB, ...)
func (router *routerSocket) Type() SocketType {
	return router.sck.Type()
}

// GetOption is used to retrieve an option for a socket.
func (router *routerSocket) GetOption(name string) (interface{}, error) {
	return router.sck.GetOption(name)
}

// SetOption is used to set an option for a socket.
func (router *routerSocket) SetOption(name string, value interface{}) error {
	return router.sck.SetOption(name, value)
}

var (
	_ Socket = (*routerSocket)(nil)
)
