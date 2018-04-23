// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"context"
)

// NewPub returns a new PUB ZeroMQ socket.
// The returned socket value is initially unbound.
func NewPub(ctx context.Context, opts ...Option) Socket {
	return &pubSocket{newSocket(ctx, Pub, opts...)}
}

// pubSocket is a PUB ZeroMQ socket.
type pubSocket struct {
	*socket
}

// Send puts the message on the outbound send queue.
// Send blocks until the message can be queued or the send deadline expires.
func (pub *pubSocket) Send(msg Msg) error {
	pub.socket.isReady()
	pub.socket.mu.RLock()
	var err error
	// FIXME(sbinet): only send to correct subscribers...
	for _, conn := range pub.conns {
		e := conn.SendMsg(msg)
		if e != nil && err == nil {
			err = e
		}
	}
	pub.socket.mu.RUnlock()
	return err
}

var (
	_ Socket = (*pubSocket)(nil)
)
