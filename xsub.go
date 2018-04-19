// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"github.com/go-zeromq/zmq4/zmtp"
)

// NewXSub returns a new XSUB ZeroMQ socket.
// The returned socket value is initially unbound.
func NewXSub(opts ...Option) *XSub {
	return &XSub{newSocket(zmtp.XSub, opts...)}
}

// XSub is a XSUB ZeroMQ socket.
type XSub struct {
	*socket
}

var (
	_ Socket = (*XSub)(nil)
)
