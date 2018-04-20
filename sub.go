// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"context"

	"github.com/go-zeromq/zmq4/zmtp"
)

// NewSub returns a new SUB ZeroMQ socket.
// The returned socket value is initially unbound.
func NewSub(ctx context.Context, opts ...Option) Socket {
	return &subSocket{newSocket(ctx, zmtp.Sub, opts...)}
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

	switch name {
	case OptionSubscribe:
		err = sub.Send(zmtp.NewMsgFrom([]byte{1}, []byte(value.(string))))
	case OptionUnsubscribe:
		err = sub.Send(zmtp.NewMsgFrom([]byte{0}, []byte(value.(string))))
	}

	return err
}

var (
	_ Socket = (*subSocket)(nil)
)
