// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"context"
	"net"
	"strings"
	"sync"
)

// NewXSub returns a new XSUB ZeroMQ socket.
// The returned socket value is initially unbound.
func NewXSub(ctx context.Context, opts ...Option) Socket {
	xsub := &xsubSocket{newSocket(ctx, XSub, opts...), sync.RWMutex{}, make(map[string]struct{})}
	xsub.sck.r = newQReader(xsub.sck.ctx)
	xsub.topics = make(map[string]struct{})
	return xsub
}

// xsubSocket is a XSUB ZeroMQ socket.
type xsubSocket struct {
	sck *socket
	mu     sync.RWMutex
	topics map[string]struct{}
}

// Close closes the open Socket
func (xsub *xsubSocket) Close() error {
	return xsub.sck.Close()
}

// Send puts the message on the outbound send queue.
// Send blocks until the message can be queued or the send deadline expires.
func (xsub *xsubSocket) Send(msg Msg) error {
	return xsub.sck.Send(msg)
}

// SendMulti puts the message on the outbound send queue.
// SendMulti blocks until the message can be queued or the send deadline expires.
// The message will be sent as a multipart message.
func (xsub *xsubSocket) SendMulti(msg Msg) error {
	return xsub.sck.SendMulti(msg)
}

// Recv receives a complete message.
func (xsub *xsubSocket) Recv() (Msg, error) {
	// If we're not subscribed to this message, we keep looping until we get one we are subscribed to, or an error or empty message.
	for {
		msg, err := xsub.sck.Recv()
		if err != nil || len(msg.Frames) == 0 || string(msg.Frames[0]) == "" {
			return msg, err
		}
		t := string(msg.Frames[0])
		if xsub.subscribed(t) {
			return msg, err
		}
	}
}

// Listen connects a local endpoint to the Socket.
func (xsub *xsubSocket) Listen(ep string) error {
	return xsub.sck.Listen(ep)
}

// Dial connects a remote endpoint to the Socket.
func (xsub *xsubSocket) Dial(ep string) error {
	return xsub.sck.Dial(ep)
}

// Type returns the type of this Socket (PUB, SUB, ...)
func (xsub *xsubSocket) Type() SocketType {
	return xsub.sck.Type()
}

// Addr returns the listener's address.
// Addr returns nil if the socket isn't a listener.
func (xsub *xsubSocket) Addr() net.Addr {
	return xsub.sck.Addr()
}

// GetOption is used to retrieve an option for a socket.
func (xsub *xsubSocket) GetOption(name string) (interface{}, error) {
	return xsub.sck.GetOption(name)
}

// SetOption is used to set an option for a socket.
func (xsub *xsubSocket) SetOption(name string, value interface{}) error {
	err := xsub.sck.SetOption(name, value)
	if err != nil {
		return err
	}

	var (
		topic []byte
	)

	switch name {
	case OptionSubscribe:
		k := value.(string)
		xsub.subscribe(k, 1)
		topic = append([]byte{1}, k...)

	case OptionUnsubscribe:
		k := value.(string)
		topic = append([]byte{0}, k...)
		xsub.subscribe(k, 0)

	default:
		return ErrBadProperty
	}

	xsub.sck.mu.RLock()
	if len(xsub.sck.conns) > 0 {
		err = xsub.Send(NewMsg(topic))
	}
	xsub.sck.mu.RUnlock()
	return err
}

func (xsub *xsubSocket) subscribe(topic string, v int) {
	xsub.mu.Lock()
	switch v {
	case 0:
		delete(xsub.topics, topic)
	case 1:
		xsub.topics[topic] = struct{}{}
	}
	xsub.mu.Unlock()
}

func (xsub *xsubSocket) subscribed(topic string) bool {
	xsub.mu.RLock()
	defer xsub.mu.RUnlock()
	if _, ok := xsub.topics[""]; ok {
		return true
	}
	if _, ok := xsub.topics[topic]; ok {
		return true
	}
	for k := range xsub.topics {
		if strings.HasPrefix(topic, k) {
			return true
		}
	}
	return false
}

var (
	_ Socket = (*xsubSocket)(nil)
)
