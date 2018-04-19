// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/go-zeromq/zmq4/zmtp"
	"github.com/go-zeromq/zmq4/zmtp/security/null"
	"github.com/pkg/errors"
)

const (
	defaultRetry   = 250 * time.Millisecond
	defaultTimeout = 5 * time.Second
)

var (
	errInvalidAddress = errors.New("zmq4: invalid address")
	errInvalidSocket  = errors.New("zmq4: invalid socket")

	ErrBadProperty = errors.New("zmq4: bad property")
)

// socket implements the ZeroMQ socket interface
type socket struct {
	ready chan struct{} // ready when at least 1 connection is live
	once  *sync.Once
	ep    string // socket end-point
	typ   zmtp.SocketType
	id    zmtp.SocketIdentity
	retry time.Duration
	sec   zmtp.Security
	conn  net.Conn               // transport connection
	zmtp  *zmtp.Conn             // ZMTP connection
	props map[string]interface{} // properties of this socket

	ctx      context.Context // life-line of socket
	cancel   context.CancelFunc
	listener net.Listener
	dialer   net.Dialer
}

func newDefaultSocket(sockType zmtp.SocketType) *socket {
	return &socket{
		once:   new(sync.Once),
		ready:  make(chan struct{}),
		typ:    sockType,
		retry:  defaultRetry,
		sec:    null.Security(),
		props:  make(map[string]interface{}),
		ctx:    nil,
		cancel: nil,
		dialer: net.Dialer{Timeout: defaultTimeout},
	}
}

func newSocket(sockType zmtp.SocketType, opts ...Option) *socket {
	sck := newDefaultSocket(sockType)
	for _, opt := range opts {
		opt(sck)
	}
	if sck.ctx == nil {
		sck.ctx, sck.cancel = context.WithCancel(context.Background())
	}
	return sck
}

// Close closes the open Socket
func (sck *socket) Close() error {
	sck.cancel()
	if sck.listener != nil {
		defer sck.listener.Close()
	}

	if sck.conn == nil {
		return errInvalidSocket
	}
	return sck.conn.Close()
}

// Send puts the message on the outbound send queue.
// Send blocks until the message can be queued or the send deadline expires.
func (sck *socket) Send(msg zmtp.Msg) error {
	sck.isReady()
	return sck.zmtp.SendMsg(msg)
}

// Recv receives a complete message.
func (sck *socket) Recv() (zmtp.Msg, error) {
	sck.isReady()
	return sck.zmtp.RecvMsg()
}

// Listen connects a local endpoint to the Socket.
func (sck *socket) Listen(endpoint string) error {
	sck.ep = endpoint
	network, addr, err := splitAddr(endpoint)
	if err != nil {
		return err
	}

	var l net.Listener

	switch network {
	case "ipc":
		l, err = net.Listen("unix", addr)
	case "tcp":
		l, err = net.Listen("tcp", addr)
	case "udp":
		l, err = net.Listen("udp", addr)
	case "inproc":
		panic("zmq4: inproc not implemented")
	default:
		panic("zmq4: unknown protocol " + network)
	}

	if err != nil {
		return errors.Wrapf(err, "could not listen to %q", endpoint)
	}
	sck.listener = l

	go sck.accept()

	return nil
}

func (sck *socket) accept() {
	ctx, cancel := context.WithCancel(sck.ctx)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return
		default:
			conn, err := sck.listener.Accept()
			if err != nil {
				// log.Printf("zmq4: error accepting connection from %q: %v", sck.ep, err)
				continue
			}
			// FIXME(sbinet): multiple connections...
			sck.conn = conn

			sck.zmtp, err = zmtp.Open(sck.conn, sck.sec, sck.typ, sck.id, true)
			if err != nil {
				panic(err)
				//		return errors.Wrapf(err, "could not open a ZMTP connection")
			}
			go func() {
				select {
				case sck.ready <- struct{}{}:
				}
			}()
		}
	}
}

// Dial connects a remote endpoint to the Socket.
func (sck *socket) Dial(endpoint string) error {
	sck.ep = endpoint

	network, addr, err := splitAddr(endpoint)
	if err != nil {
		return err
	}

	retries := 0
	var conn net.Conn
connect:
	switch network {
	case "ipc":
		conn, err = sck.dialer.DialContext(sck.ctx, "unix", addr)
	case "tcp":
		conn, err = sck.dialer.DialContext(sck.ctx, "tcp", addr)
	case "udp":
		conn, err = sck.dialer.DialContext(sck.ctx, "udp", addr)
	case "inproc":
		panic("zmq4: inproc not implemented")
	default:
		panic("zmq4: unknown protocol " + network)
	}

	if err != nil {
		if retries < 10 {
			retries++
			time.Sleep(sck.retry)
			goto connect
		}
		return errors.Wrapf(err, "could not dial to %q", endpoint)
	}

	if conn == nil {
		return errors.Wrapf(err, "got a nil dial-conn to %q", endpoint)
	}

	sck.conn = conn

	sck.zmtp, err = zmtp.Open(sck.conn, sck.sec, sck.typ, sck.id, false)
	if err != nil {
		return errors.Wrapf(err, "could not open a ZMTP connection")
	}
	if sck.zmtp == nil {
		return errors.Wrapf(err, "got a nil ZMTP connection to %q", endpoint)
	}

	go func() {
		select {
		case sck.ready <- struct{}{}:
		}
	}()
	return nil
}

// Type returns the type of this Socket (PUB, SUB, ...)
func (sck *socket) Type() zmtp.SocketType {
	return sck.typ
}

// Conn returns the underlying net.Conn the socket is bound to.
func (sck *socket) Conn() net.Conn {
	return sck.conn
}

// GetOption is used to retrieve an option for a socket.
func (sck *socket) GetOption(name string) (interface{}, error) {
	v, ok := sck.props[name]
	if !ok {
		return nil, ErrBadProperty
	}
	return v, nil
}

// SetOption is used to set an option for a socket.
func (sck *socket) SetOption(name string, value interface{}) error {
	// FIXME(sbinet) different socket types support different options.
	sck.props[name] = value
	return nil
}

func (sck *socket) isReady() {
	sck.once.Do(func() {
		<-sck.ready
		sck.ready = nil
	})
}

var (
	_ Socket = (*socket)(nil)
)

// splitAddr returns the triplet (network, addr, error)
func splitAddr(v string) (network, addr string, err error) {
	ep := strings.Split(v, "://")
	if len(ep) != 2 {
		err = errInvalidAddress
		return network, addr, err
	}
	var (
		host string
		port string
	)
	network = ep[0]
	switch network {
	case "tcp", "udp":
		host, port, err = net.SplitHostPort(ep[1])
		if err != nil {
			return network, addr, err
		}
		switch port {
		case "0", "*", "":
			port = "0"
		}
		switch host {
		case "", "*":
			host = "0.0.0.0"
		}
		addr = host + ":" + port
		return network, addr, err

	case "ipc":
		host = ep[1]
		port = ""
		return network, host, nil
	case "inproc":
		err = fmt.Errorf("zmq4: protocol %q not implemented", network)
	default:
		err = fmt.Errorf("zmq4: unknown protocol %q", network)
	}

	return network, addr, err
}
