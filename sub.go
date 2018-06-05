// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"context"
)

// NewSub returns a new SUB ZeroMQ socket.
// The returned socket value is initially unbound.
func NewSub(ctx context.Context, opts ...Option) Socket {
	sub := &subSocket{sck: newSocket(ctx, Sub, opts...)}
	sub.sck.r = newQReader()
	return sub
}

// subSocket is a SUB ZeroMQ socket.
type subSocket struct {
	sck *socket
}

// Close closes the open Socket
func (sub *subSocket) Close() error {
	return sub.sck.Close()
}

// Send puts the message on the outbound send queue.
// Send blocks until the message can be queued or the send deadline expires.
func (sub *subSocket) Send(msg Msg) error {
	return sub.sck.Send(msg)
}

// Recv receives a complete message.
func (sub *subSocket) Recv() (Msg, error) {
	return sub.sck.Recv()
}

// Listen connects a local endpoint to the Socket.
func (sub *subSocket) Listen(ep string) error {
	return sub.sck.Listen(ep)
}

// Dial connects a remote endpoint to the Socket.
func (sub *subSocket) Dial(ep string) error {
	return sub.sck.Dial(ep)
}

// Type returns the type of this Socket (PUB, SUB, ...)
func (sub *subSocket) Type() SocketType {
	return sub.sck.Type()
}

// GetOption is used to retrieve an option for a socket.
func (sub *subSocket) GetOption(name string) (interface{}, error) {
	return sub.sck.GetOption(name)
}

// SetOption is used to set an option for a socket.
func (sub *subSocket) SetOption(name string, value interface{}) error {
	err := sub.sck.SetOption(name, value)
	if err != nil {
		return err
	}

	var (
		topic []byte
	)

	switch name {
	case OptionSubscribe:
		topic = []byte{1}
		topic = append(topic, value.(string)...)

	case OptionUnsubscribe:
		topic = []byte{0}
		topic = append(topic, value.(string)...)

	default:
		return ErrBadProperty
	}

	sub.sck.mu.RLock()
	msg := NewMsg(topic)
	for _, conn := range sub.sck.conns {
		e := conn.SendMsg(msg)
		if e != nil && err == nil {
			err = e
		}
	}
	sub.sck.mu.RUnlock()
	return err
}

var (
	_ Socket = (*subSocket)(nil)
)
