// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package zmq4 implements the ØMQ sockets and protocol for ZeroMQ-4.
//
// For more informations, see http://zeromq.org.
package zmq4

import "net"

// Socket represents a ZeroMQ socket.
type Socket interface {
	// Close closes the open Socket
	Close() error

	// Send puts the message on the outbound send queue.
	// Send blocks until the message can be queued or the send deadline expires.
	Send(msg Msg) error

	// SendMulti puts the message on the outbound send queue.
	// SendMulti blocks until the message can be queued or the send deadline expires.
	// The message will be sent as a multipart message.
	SendMulti(msg Msg) error

	// Recv receives a complete message.
	Recv() (Msg, error)

	// Listen connects a local endpoint to the Socket.
	Listen(ep string) error

	// Dial connects a remote endpoint to the Socket.
	Dial(ep string) error

	// Type returns the type of this Socket (PUB, SUB, ...)
	Type() SocketType

	// Addr returns the listener's address.
	// Addr returns nil if the socket isn't a listener.
	Addr() net.Addr

	// GetOption is used to retrieve an option for a socket.
	GetOption(name string) (interface{}, error)

	// SetOption is used to set an option for a socket.
	SetOption(name string, value interface{}) error
}
