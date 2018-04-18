// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"github.com/go-zeromq/zmq4/zmtp"
)

// NewPull returns a new PULL ZeroMQ socket.
// The returned socket value is initially unbound.
func NewPull() *Pull {
	return &Pull{newSocket(zmtp.Pull)}
}

// Pull is a PULL ZeroMQ socket.
type Pull struct {
	*socket
}

var (
	_ Socket = (*Pull)(nil)
)
