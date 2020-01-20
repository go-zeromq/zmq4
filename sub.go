// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"context"
	"net"
	"sort"
	"sync"
)

// NewSub returns a new SUB ZeroMQ socket.
// The returned socket value is initially unbound.
func NewSub(ctx context.Context, opts ...Option) Socket {
	sub := &subSocket{sck: newSocket(ctx, Sub, opts...)}
	sub.sck.r = newQReader(sub.sck.ctx)
	sub.topics = make(map[string]struct{})
	return sub
}

// subSocket is a SUB ZeroMQ socket.
type subSocket struct {
	sck *socket

	mu     sync.RWMutex
	topics map[string]struct{}
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

// SendMulti puts the message on the outbound send queue.
// SendMulti blocks until the message can be queued or the send deadline expires.
// The message will be sent as a multipart message.
func (sub *subSocket) SendMulti(msg Msg) error {
	return sub.sck.SendMulti(msg)
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
	err := sub.sck.Dial(ep)
	if err != nil {
		return err
	}
	return nil
}

// Type returns the type of this Socket (PUB, SUB, ...)
func (sub *subSocket) Type() SocketType {
	return sub.sck.Type()
}

// Addr returns the listener's address.
// Addr returns nil if the socket isn't a listener.
func (sub *subSocket) Addr() net.Addr {
	return sub.sck.Addr()
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
		k := value.(string)
		sub.subscribe(k, 1)
		topic = append([]byte{1}, k...)

	case OptionUnsubscribe:
		k := value.(string)
		topic = append([]byte{0}, k...)
		sub.subscribe(k, 0)

	default:
		return ErrBadProperty
	}

	sub.sck.mu.RLock()
	if len(sub.sck.conns) > 0 {
		err = sub.Send(NewMsg(topic))
	}
	sub.sck.mu.RUnlock()
	return err
}

// Topics returns the sorted list of topics a socket is subscribed to.
func (sub *subSocket) Topics() []string {
	sub.mu.RLock()
	var topics = make([]string, 0, len(sub.topics))
	for topic := range sub.topics {
		topics = append(topics, topic)
	}
	sub.mu.RUnlock()
	sort.Strings(topics)
	return topics
}

func (sub *subSocket) subscribe(topic string, v int) {
	sub.mu.Lock()
	switch v {
	case 0:
		delete(sub.topics, topic)
	case 1:
		sub.topics[topic] = struct{}{}
	}
	sub.mu.Unlock()
}

var (
	_ Socket = (*subSocket)(nil)
	_ Topics = (*subSocket)(nil)
)
