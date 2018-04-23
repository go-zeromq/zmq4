// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"context"
)

// NewPair returns a new PAIR ZeroMQ socket.
// The returned socket value is initially unbound.
func NewPair(ctx context.Context, opts ...Option) Socket {
	return &pairSocket{newSocket(ctx, Pair, opts...)}
}

// pairSocket is a PAIR ZeroMQ socket.
type pairSocket struct {
	*socket
}

var (
	_ Socket = (*pairSocket)(nil)
)
