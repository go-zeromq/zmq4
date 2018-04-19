// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"github.com/go-zeromq/zmq4/zmtp"
	"github.com/pkg/errors"
)

// NewPull returns a new PULL ZeroMQ socket.
// The returned socket value is initially unbound.
func NewPull(opts ...Option) *Pull {
	return &Pull{newSocket(zmtp.Pull, opts...)}
}

// Pull is a PULL ZeroMQ socket.
type Pull struct {
	*socket
}

// Send puts the message on the outbound send queue.
// Send blocks until the message can be queued or the send deadline expires.
func (*Pull) Send(msg zmtp.Msg) error {
	return errors.Errorf("zmq4: PULL sockets can't send messages")
}

var (
	_ Socket = (*Pull)(nil)
)
