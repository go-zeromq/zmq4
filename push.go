// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"github.com/go-zeromq/zmq4/zmtp"
)

// NewPush returns a new PUSH ZeroMQ socket.
// The returned socket value is initially unbound.
func NewPush() *Push {
	return &Push{newSocket(zmtp.Push)}
}

// Push is a PUSH ZeroMQ socket.
type Push struct {
	*socket
}

var (
	_ Socket = (*Push)(nil)
)
