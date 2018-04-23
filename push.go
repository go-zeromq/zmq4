// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"context"

	"github.com/pkg/errors"
)

// NewPush returns a new PUSH ZeroMQ socket.
// The returned socket value is initially unbound.
func NewPush(ctx context.Context, opts ...Option) Socket {
	return &pushSocket{newSocket(ctx, Push, opts...)}
}

// pushSocket is a PUSH ZeroMQ socket.
type pushSocket struct {
	*socket
}

// Recv receives a complete message.
func (*pushSocket) Recv() (Msg, error) {
	return Msg{}, errors.Errorf("zmq4: PUSH sockets can't recv messages")
}

var (
	_ Socket = (*pushSocket)(nil)
)
