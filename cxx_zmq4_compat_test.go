// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build czmq4

package zmq4_test

import (
	"net"

	"github.com/go-zeromq/zmq4"
	"github.com/go-zeromq/zmq4/zmtp"
	czmq4 "github.com/zeromq/goczmq"
)

func NewCPair() zmq4.Socket {
	return &csocket{czmq4.NewSock(czmq4.Pair)}
}

func NewCPub() zmq4.Socket {
	return &csocket{czmq4.NewSock(czmq4.Pub)}
}

func NewCSub() zmq4.Socket {
	return &csocket{czmq4.NewSock(czmq4.Sub)}
}

func NewCReq() zmq4.Socket {
	return &csocket{czmq4.NewSock(czmq4.Req)}
}

func NewCRep() zmq4.Socket {
	return &csocket{czmq4.NewSock(czmq4.Rep)}
}

func NewCDealer() zmq4.Socket {
	return &csocket{czmq4.NewSock(czmq4.Dealer)}
}

func NewCRouter() zmq4.Socket {
	return &csocket{czmq4.NewSock(czmq4.Router)}
}

func NewCPull() zmq4.Socket {
	return &csocket{czmq4.NewSock(czmq4.Pull)}
}

func NewCPush() zmq4.Socket {
	return &csocket{czmq4.NewSock(czmq4.Push)}
}

func NewCXPub() zmq4.Socket {
	return &csocket{czmq4.NewSock(czmq4.XPub)}
}

func NewCXSub() zmq4.Socket {
	return &csocket{czmq4.NewSock(czmq4.XSub)}
}

func NewCStream() zmq4.Socket {
	return &csocket{czmq4.NewSock(czmq4.Stream)}
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
func (sck *csocket) Send(data []byte) error {
	return sck.sock.SendFrame(data, czmq4.FlagNone)
}

// Recv receives a complete message.
func (sck *csocket) Recv() ([]byte, error) {
	msg, _, err := sck.sock.RecvFrame()
	return msg, err
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
func (sck *csocket) Type() zmtp.SocketType {
	switch sck.sock.GetType() {
	case czmq4.Pair:
		return zmtp.Pair
	case czmq4.Pub:
		return zmtp.Pub
	case czmq4.Sub:
		return zmtp.Sub
	case czmq4.Req:
		return zmtp.Req
	case czmq4.Rep:
		return zmtp.Rep
	case czmq4.Dealer:
		return zmtp.Dealer
	case czmq4.Router:
		return zmtp.Router
	case czmq4.Pull:
		return zmtp.Pull
	case czmq4.Push:
		return zmtp.Push
	case czmq4.XPub:
		return zmtp.XPub
	case czmq4.XSub:
		return zmtp.XSub
	case czmq4.Stream:
		return zmtp.Stream
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
	panic("not implemented")
}

var (
	_ zmq4.Socket = (*csocket)(nil)
)
