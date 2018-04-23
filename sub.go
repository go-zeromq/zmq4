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
	return &subSocket{newSocket(ctx, Sub, opts...)}
}

// subSocket is a SUB ZeroMQ socket.
type subSocket struct {
	*socket
}

// SetOption is used to set an option for a socket.
func (sub *subSocket) SetOption(name string, value interface{}) error {
	err := sub.socket.SetOption(name, value)
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

	sub.socket.mu.RLock()
	msg := NewMsg(topic)
	for _, conn := range sub.conns {
		e := conn.SendMsg(msg)
		if e != nil && err == nil {
			err = e
		}
	}
	sub.socket.mu.RUnlock()
	return err
}

var (
	_ Socket = (*subSocket)(nil)
)
