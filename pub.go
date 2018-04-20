// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"context"

	"github.com/go-zeromq/zmq4/zmtp"
)

// NewPub returns a new PUB ZeroMQ socket.
// The returned socket value is initially unbound.
func NewPub(ctx context.Context, opts ...Option) Socket {
	return &pubSocket{newSocket(ctx, zmtp.Pub, opts...)}
}

// pubSocket is a PUB ZeroMQ socket.
type pubSocket struct {
	*socket
}

var (
	_ Socket = (*pubSocket)(nil)
)
