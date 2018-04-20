// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"github.com/go-zeromq/zmq4/zmtp"
)

// NewRouter returns a new ROUTER ZeroMQ socket.
// The returned socket value is initially unbound.
func NewRouter(opts ...Option) Socket {
	return &routerSocket{newSocket(zmtp.Router, opts...)}
}

// routerSocket is a ROUTER ZeroMQ socket.
type routerSocket struct {
	*socket
}

var (
	_ Socket = (*routerSocket)(nil)
)
