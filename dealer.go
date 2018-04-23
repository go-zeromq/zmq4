// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"context"
)

// NewDealer returns a new DEALER ZeroMQ socket.
// The returned socket value is initially unbound.
func NewDealer(ctx context.Context, opts ...Option) Socket {
	return &dealerSocket{newSocket(ctx, Dealer, opts...)}
}

// dealerSocket is a DEALER ZeroMQ socket.
type dealerSocket struct {
	*socket
}

var (
	_ Socket = (*dealerSocket)(nil)
)
