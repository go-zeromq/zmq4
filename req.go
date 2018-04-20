// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"context"

	"github.com/go-zeromq/zmq4/zmtp"
)

// NewReq returns a new REQ ZeroMQ socket.
// The returned socket value is initially unbound.
func NewReq(ctx context.Context, opts ...Option) Socket {
	return &reqSocket{newSocket(ctx, zmtp.Req, opts...)}
}

// reqSocket is a REQ ZeroMQ socket.
type reqSocket struct {
	*socket
}

// Send puts the message on the outbound send queue.
// Send blocks until the message can be queued or the send deadline expires.
func (req *reqSocket) Send(msg zmtp.Msg) error {
	msg.Frames = append([][]byte{nil}, msg.Frames...)
	return req.socket.Send(msg)
}

// Recv receives a complete message.
func (req *reqSocket) Recv() (zmtp.Msg, error) {
	msg, err := req.socket.Recv()
	if len(msg.Frames) > 1 {
		msg.Frames = msg.Frames[1:]
	}
	return msg, err
}

var (
	_ Socket = (*reqSocket)(nil)
)
