// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"github.com/go-zeromq/zmq4/zmtp"
)

// NewSub returns a new SUB ZeroMQ socket.
// The returned socket value is initially unbound.
func NewSub(opts ...Option) *Sub {
	return &Sub{newSocket(zmtp.Sub, opts...)}
}

// Sub is a SUB ZeroMQ socket.
type Sub struct {
	*socket
}

var (
	_ Socket = (*Sub)(nil)
)
