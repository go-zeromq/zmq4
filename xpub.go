// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"github.com/go-zeromq/zmq4/zmtp"
)

// NewXPub returns a new XPUB ZeroMQ socket.
// The returned socket value is initially unbound.
func NewXPub(opts ...Option) *XPub {
	return &XPub{newSocket(zmtp.XPub, opts...)}
}

// XPub is a XPUB ZeroMQ socket.
type XPub struct {
	*socket
}

var (
	_ Socket = (*XPub)(nil)
)
