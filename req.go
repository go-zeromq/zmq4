// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"context"
	"net"

	"golang.org/x/xerrors"
)

// NewReq returns a new REQ ZeroMQ socket.
// The returned socket value is initially unbound.
func NewReq(ctx context.Context, opts ...Option) Socket {
	req := &reqSocket{newSocket(ctx, Req, opts...)}
	return req
}

// reqSocket is a REQ ZeroMQ socket.
type reqSocket struct {
	sck *socket
}

// Close closes the open Socket
func (req *reqSocket) Close() error {
	return req.sck.Close()
}

// Send puts the message on the outbound send queue.
// Send blocks until the message can be queued or the send deadline expires.
func (req *reqSocket) Send(msg Msg) error {
	msg.Frames = append([][]byte{nil}, msg.Frames...)
	return req.sck.Send(msg)
}

// Recv receives a complete message.
func (req *reqSocket) Recv() (Msg, error) {
	msg, err := req.sck.Recv()
	if len(msg.Frames) > 1 {
		msg.Frames = msg.Frames[1:]
	}
	return msg, err
}

// Listen connects a local endpoint to the Socket.
func (req *reqSocket) Listen(ep string) error {
	return req.sck.Listen(ep)
}

// Dial connects a remote endpoint to the Socket.
func (req *reqSocket) Dial(ep string) error {
	return req.sck.Dial(ep)
}

// Type returns the type of this Socket (PUB, SUB, ...)
func (req *reqSocket) Type() SocketType {
	return req.sck.Type()
}

// Addr returns the listener's address.
// Addr returns nil if the socket isn't a listener.
func (req *reqSocket) Addr() net.Addr {
	return req.sck.Addr()
}

// GetOption is used to retrieve an option for a socket.
func (req *reqSocket) GetOption(name string) (interface{}, error) {
	return req.sck.GetOption(name)
}

// SetOption is used to set an option for a socket.
func (req *reqSocket) SetOption(name string, value interface{}) error {
	return req.sck.SetOption(name, value)
}

// GetTopics is used to retrieve subscribed topics for a pub socket.
func (req *reqSocket) GetTopics(filter bool) ([]string, error) {
	err := xerrors.Errorf("zmq4: Only available for PUB sockets")
	return nil, err
}

var (
	_ Socket = (*reqSocket)(nil)
)
