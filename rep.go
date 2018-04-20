// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"github.com/go-zeromq/zmq4/zmtp"
)

// NewRep returns a new REP ZeroMQ socket.
// The returned socket value is initially unbound.
func NewRep(opts ...Option) Socket {
	return &repSocket{newSocket(zmtp.Rep, opts...)}
}

// repSocket is a REP ZeroMQ socket.
type repSocket struct {
	*socket
}

var (
	_ Socket = (*repSocket)(nil)
)
