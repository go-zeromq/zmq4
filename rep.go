// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"context"

	"github.com/go-zeromq/zmq4/zmtp"
)

// NewRep returns a new REP ZeroMQ socket.
// The returned socket value is initially unbound.
func NewRep(ctx context.Context, opts ...Option) Socket {
	return &repSocket{newSocket(ctx, zmtp.Rep, opts...)}
}

// repSocket is a REP ZeroMQ socket.
type repSocket struct {
	*socket
}

// Send puts the message on the outbound send queue.
// Send blocks until the message can be queued or the send deadline expires.
func (rep *repSocket) Send(msg zmtp.Msg) error {
	msg.Frames = append([][]byte{nil}, msg.Frames...)
	return rep.socket.Send(msg)
}

// Recv receives a complete message.
func (rep *repSocket) Recv() (zmtp.Msg, error) {
	msg, err := rep.socket.Recv()
	if len(msg.Frames) > 1 {
		msg.Frames = msg.Frames[1:]
	}
	return msg, err
}

var (
	_ Socket = (*repSocket)(nil)
)
