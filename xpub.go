// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"context"
)

// NewXPub returns a new XPUB ZeroMQ socket.
// The returned socket value is initially unbound.
func NewXPub(ctx context.Context, opts ...Option) Socket {
	return &xpubSocket{newSocket(ctx, XPub, opts...)}
}

// xpubSocket is a XPUB ZeroMQ socket.
type xpubSocket struct {
	*socket
}

var (
	_ Socket = (*xpubSocket)(nil)
)
