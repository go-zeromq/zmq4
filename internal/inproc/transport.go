// Copyright 2020 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package inproc

import (
	"context"
	"net"

	"github.com/go-zeromq/zmq4/transport"
)

// Transport implements the zmq4 Transport interface for the inproc transport.
type Transport struct{}

// Dial connects to the address on the named network using the provided
// context.
func (Transport) Dial(ctx context.Context, dialer transport.Dialer, addr string) (net.Conn, error) {
	return Dial(addr)
}

// Listen announces on the provided network address.
func (Transport) Listen(ctx context.Context, addr string) (net.Listener, error) {
	return Listen(addr)
}

// Addr returns the end-point address.
func (Transport) Addr(ep string) (addr string, err error) {
	return ep, nil
}

var (
	_ transport.Transport = (*Transport)(nil)
)
