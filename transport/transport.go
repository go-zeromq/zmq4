// Copyright 2020 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package transport defines the Transport interface and provides a net-based
// implementation that can be used by zmq4 sockets to exchange messages.
package transport // import "github.com/go-zeromq/zmq4/transport"

import (
	"context"
	"fmt"
	"net"
)

// Dialer is the interface that wraps the DialContext method.
type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// Transport is the zmq4 transport interface that wraps
// the Dial and Listen methods.
type Transport interface {
	// Dial connects to the address on the named network using the provided
	// context.
	Dial(ctx context.Context, dialer Dialer, addr string) (net.Conn, error)

	// Listen announces on the provided network address.
	Listen(ctx context.Context, addr string) (net.Listener, error)

	// Addr returns the end-point address.
	Addr(ep string) (addr string, err error)
}

// netTransport implements the Transport interface using the net package.
type netTransport struct {
	prot string
}

// New returns a new net-based transport with the given network (e.g "tcp").
func New(network string) Transport {
	return netTransport{prot: network}
}

// Dial connects to the address on the named network using the provided
// context.
func (trans netTransport) Dial(ctx context.Context, dialer Dialer, addr string) (net.Conn, error) {
	return dialer.DialContext(ctx, trans.prot, addr)
}

// Listen announces on the provided network address.
func (trans netTransport) Listen(ctx context.Context, addr string) (net.Listener, error) {
	return net.Listen(trans.prot, addr)
}

// Addr returns the end-point address.
func (trans netTransport) Addr(ep string) (addr string, err error) {
	switch trans.prot {
	case "tcp", "udp":
		host, port, err := net.SplitHostPort(ep)
		if err != nil {
			return addr, err
		}
		switch port {
		case "0", "*", "":
			port = "0"
		}
		switch host {
		case "", "*":
			host = "0.0.0.0"
		}
		addr = net.JoinHostPort(host, port)
		return addr, err

	case "unix":
		return ep, nil

	default:
		err = fmt.Errorf("zmq4: unknown protocol %q", trans.prot)
	}

	return addr, err
}

var (
	_ Transport = (*netTransport)(nil)
)
