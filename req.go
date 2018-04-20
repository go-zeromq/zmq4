// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"github.com/go-zeromq/zmq4/zmtp"
)

// NewReq returns a new REQ ZeroMQ socket.
// The returned socket value is initially unbound.
func NewReq(opts ...Option) Socket {
	return &reqSocket{newSocket(zmtp.Req, opts...)}
}

// reqSocket is a REQ ZeroMQ socket.
type reqSocket struct {
	*socket
}

var (
	_ Socket = (*reqSocket)(nil)
)
