// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"context"
	"net"
)

// NewDealer returns a new DEALER ZeroMQ socket.
// The returned socket value is initially unbound.
func NewDealer(ctx context.Context, opts ...Option) Socket {
	dealer := &dealerSocket{newSocket(ctx, Dealer, opts...)}
	return dealer
}

// dealerSocket is a DEALER ZeroMQ socket.
type dealerSocket struct {
	sck *socket
}

// Close closes the open Socket
func (dealer *dealerSocket) Close() error {
	return dealer.sck.Close()
}

// Send puts the message on the outbound send queue.
// Send blocks until the message can be queued or the send deadline expires.
func (dealer *dealerSocket) Send(msg Msg) error {
	return dealer.sck.Send(msg)
}

// SendMulti puts the message on the outbound send queue.
// SendMulti blocks until the message can be queued or the send deadline expires.
// The message will be sent as a multipart message.
func (dealer *dealerSocket) SendMulti(msg Msg) error {
	return dealer.sck.SendMulti(msg)
}

// Recv receives a complete message.
func (dealer *dealerSocket) Recv() (Msg, error) {
	return dealer.sck.Recv()
}

// Listen connects a local endpoint to the Socket.
func (dealer *dealerSocket) Listen(ep string) error {
	return dealer.sck.Listen(ep)
}

// Dial connects a remote endpoint to the Socket.
func (dealer *dealerSocket) Dial(ep string) error {
	return dealer.sck.Dial(ep)
}

// Type returns the type of this Socket (PUB, SUB, ...)
func (dealer *dealerSocket) Type() SocketType {
	return dealer.sck.Type()
}

// Addr returns the listener's address.
// Addr returns nil if the socket isn't a listener.
func (dealer *dealerSocket) Addr() net.Addr {
	return dealer.sck.Addr()
}

// GetOption is used to retrieve an option for a socket.
func (dealer *dealerSocket) GetOption(name string) (interface{}, error) {
	return dealer.sck.GetOption(name)
}

// SetOption is used to set an option for a socket.
func (dealer *dealerSocket) SetOption(name string, value interface{}) error {
	return dealer.sck.SetOption(name, value)
}

var (
	_ Socket = (*dealerSocket)(nil)
)
