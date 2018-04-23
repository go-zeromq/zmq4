// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build czmq4

package zmq4

import (
	"context"
	"net"

	czmq4 "github.com/zeromq/goczmq"
)

func NewCPair(ctx context.Context) Socket {
	return &csocket{czmq4.NewSock(czmq4.Pair)}
}

func NewCPub(ctx context.Context) Socket {
	return &csocket{czmq4.NewSock(czmq4.Pub)}
}

func NewCSub(ctx context.Context) Socket {
	return &csocket{czmq4.NewSock(czmq4.Sub)}
}

func NewCReq(ctx context.Context) Socket {
	return &csocket{czmq4.NewSock(czmq4.Req)}
}

func NewCRep(ctx context.Context) Socket {
	return &csocket{czmq4.NewSock(czmq4.Rep)}
}

func NewCDealer(ctx context.Context) Socket {
	return &csocket{czmq4.NewSock(czmq4.Dealer)}
}

func NewCRouter(ctx context.Context) Socket {
	return &csocket{czmq4.NewSock(czmq4.Router)}
}

func NewCPull(ctx context.Context) Socket {
	return &csocket{czmq4.NewSock(czmq4.Pull)}
}

func NewCPush(ctx context.Context) Socket {
	return &csocket{czmq4.NewSock(czmq4.Push)}
}

func NewCXPub(ctx context.Context) Socket {
	return &csocket{czmq4.NewSock(czmq4.XPub)}
}

func NewCXSub(ctx context.Context) Socket {
	return &csocket{czmq4.NewSock(czmq4.XSub)}
}

type csocket struct {
	sock *czmq4.Sock
}

func (sck *csocket) Close() error {
	sck.sock.Destroy()
	return nil
}

// Send puts the message on the outbound send queue.
// Send blocks until the message can be queued or the send deadline expires.
func (sck *csocket) Send(msg Msg) error {
	return sck.sock.SendMessage(msg.Frames)
}

// Recv receives a complete message.
func (sck *csocket) Recv() (Msg, error) {
	frames, err := sck.sock.RecvMessage()
	return Msg{Frames: frames}, err
}

// Listen connects a local endpoint to the Socket.
func (sck *csocket) Listen(addr string) error {
	_, err := sck.sock.Bind(addr)
	return err
}

// Dial connects a remote endpoint to the Socket.
func (sck *csocket) Dial(addr string) error {
	return sck.sock.Connect(addr)
}

// Type returns the type of this Socket (PUB, SUB, ...)
func (sck *csocket) Type() SocketType {
	switch sck.sock.GetType() {
	case czmq4.Pair:
		return Pair
	case czmq4.Pub:
		return Pub
	case czmq4.Sub:
		return Sub
	case czmq4.Req:
		return Req
	case czmq4.Rep:
		return Rep
	case czmq4.Dealer:
		return Dealer
	case czmq4.Router:
		return Router
	case czmq4.Pull:
		return Pull
	case czmq4.Push:
		return Push
	case czmq4.XPub:
		return XPub
	case czmq4.XSub:
		return XSub
	}
	panic("invalid C-socket type")
}

// Conn returns the underlying net.Conn the socket is bound to.
func (sck *csocket) Conn() net.Conn {
	panic("not implemented")
}

// GetOption is used to retrieve an option for a socket.
func (sck *csocket) GetOption(name string) (interface{}, error) {
	panic("not implemented")
}

// SetOption is used to set an option for a socket.
func (sck *csocket) SetOption(name string, value interface{}) error {
	switch name {
	case OptionSubscribe:
		topic := value.(string)
		sck.sock.SetOption(czmq4.SockSetSubscribe(topic))
		return nil
	case OptionUnsubscribe:
		topic := value.(string)
		sck.sock.SetUnsubscribe(topic)
		return nil
	default:
		panic("unknown set option name [" + name + "]")
	}
	panic("not implemented")
}

var (
	_ Socket = (*csocket)(nil)
)
