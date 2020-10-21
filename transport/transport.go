// Copyright 2020 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package transport

import (
	"context"
	"net"
)

type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// Transport is the zmq4 transport interface that wraps
// the Dial and Listen methods.
type Transport interface {
	Dial(ctx context.Context, dialer Dialer, addr string) (net.Conn, error)
	Listen(ctx context.Context, addr string) (net.Listener, error)
}

type netTransport struct {
	prot string
}

// New returns a new net-based transport with the given network (e.g "tcp").
func New(network string) Transport {
	return netTransport{prot: network}
}

func (trans netTransport) Dial(ctx context.Context, dialer Dialer, addr string) (net.Conn, error) {
	return dialer.DialContext(ctx, trans.prot, addr)
}

func (trans netTransport) Listen(ctx context.Context, addr string) (net.Listener, error) {
	return net.Listen(trans.prot, addr)
}
