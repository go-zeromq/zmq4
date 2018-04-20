// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"context"

	"github.com/go-zeromq/zmq4/zmtp"
)

// NewSub returns a new SUB ZeroMQ socket.
// The returned socket value is initially unbound.
func NewSub(ctx context.Context, opts ...Option) Socket {
	return &subSocket{newSocket(ctx, zmtp.Sub, opts...)}
}

// subSocket is a SUB ZeroMQ socket.
type subSocket struct {
	*socket
}

var (
	_ Socket = (*subSocket)(nil)
)
