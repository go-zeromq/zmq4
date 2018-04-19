// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"github.com/go-zeromq/zmq4/zmtp"
)

// NewPub returns a new PUB ZeroMQ socket.
// The returned socket value is initially unbound.
func NewPub(opts ...Option) *Pub {
	return &Pub{newSocket(zmtp.Pub, opts...)}
}

// Pub is a PUB ZeroMQ socket.
type Pub struct {
	*socket
}

var (
	_ Socket = (*Pub)(nil)
)
