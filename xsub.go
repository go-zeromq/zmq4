// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"context"
)

// NewXSub returns a new XSUB ZeroMQ socket.
// The returned socket value is initially unbound.
func NewXSub(ctx context.Context, opts ...Option) Socket {
	return &xsubSocket{newSocket(ctx, XSub, opts...)}
}

// xsubSocket is a XSUB ZeroMQ socket.
type xsubSocket struct {
	*socket
}

var (
	_ Socket = (*xsubSocket)(nil)
)
