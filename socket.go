// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"context"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
)

const (
	defaultRetry   = 250 * time.Millisecond
	defaultTimeout = 5 * time.Minute
)

var (
	errInvalidAddress = errors.New("zmq4: invalid address")
	errInvalidSocket  = errors.New("zmq4: invalid socket")

	ErrBadProperty = errors.New("zmq4: bad property")
)

// socket implements the ZeroMQ socket interface
type socket struct {
	ep    string // socket end-point
	typ   SocketType
	id    SocketIdentity
	retry time.Duration
	sec   Security

	mu    sync.RWMutex
	ids   map[string]*Conn // ZMTP connection IDs
	conns []*Conn          // ZMTP connections
	r     rpool
	w     wpool

	props map[string]interface{} // properties of this socket

	ctx      context.Context // life-line of socket
	cancel   context.CancelFunc
	listener net.Listener
	dialer   net.Dialer
}

func newDefaultSocket(ctx context.Context, sockType SocketType) *socket {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithCancel(ctx)
	return &socket{
		typ:    sockType,
		retry:  defaultRetry,
		sec:    nullSecurity{},
		ids:    make(map[string]*Conn),
		conns:  nil,
		r:      newQReader(ctx),
		w:      newMWriter(ctx),
		props:  make(map[string]interface{}),
		ctx:    ctx,
		cancel: cancel,
		dialer: net.Dialer{Timeout: defaultTimeout},
	}
}

func newSocket(ctx context.Context, sockType SocketType, opts ...Option) *socket {
	sck := newDefaultSocket(ctx, sockType)
	for _, opt := range opts {
		opt(sck)
	}
	if len(sck.id) == 0 {
		sck.id = SocketIdentity(newUUID())
	}

	return sck
}

// Close closes the open Socket
func (sck *socket) Close() error {
	sck.cancel()
	if sck.listener != nil {
		defer sck.listener.Close()
	}

	if sck.conns == nil {
		return errInvalidSocket
	}

	var err error
	sck.mu.RLock()
	for _, conn := range sck.conns {
		e := conn.Close()
		if e != nil && err == nil {
			err = e
		}
	}
	sck.mu.RUnlock()
	if strings.HasPrefix(sck.ep, "ipc://") {
		os.Remove(sck.ep[len("ipc://"):])
	}

	return err
}

// Send puts the message on the outbound send queue.
// Send blocks until the message can be queued or the send deadline expires.
func (sck *socket) Send(msg Msg) error {
	ctx, cancel := context.WithTimeout(sck.ctx, sck.timeout())
	defer cancel()
	return sck.w.write(ctx, msg)
}

// Recv receives a complete message.
func (sck *socket) Recv() (Msg, error) {
	ctx, cancel := context.WithCancel(sck.ctx)
	defer cancel()
	var msg Msg
	err := sck.r.read(ctx, &msg)
	return msg, err
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

			zconn, err := Open(conn, sck.sec, sck.typ, sck.id, true)
			if err != nil {
				panic(err)
				//		return errors.Wrapf(err, "could not open a ZMTP connection")
			}

			sck.addConn(zconn)
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

	zconn, err := Open(conn, sck.sec, sck.typ, sck.id, false)
	if err != nil {
		return errors.Wrapf(err, "could not open a ZMTP connection")
	}
	if zconn == nil {
		return errors.Wrapf(err, "got a nil ZMTP connection to %q", endpoint)
	}

	sck.addConn(zconn)
	return nil
}

func (sck *socket) addConn(c *Conn) {
	sck.mu.Lock()
	sck.conns = append(sck.conns, c)
	uuid, ok := c.peer.meta[sysSockID]
	if !ok {
		uuid = newUUID()
		c.peer.meta[sysSockID] = uuid
	}
	sck.ids[uuid] = c
	if sck.r != nil {
		sck.r.addConn(newMsgReader(c))
	}
	if sck.w != nil {
		sck.w.addConn(newMsgWriter(c))
	}
	sck.mu.Unlock()
}

// Type returns the type of this Socket (PUB, SUB, ...)
func (sck *socket) Type() SocketType {
	return sck.typ
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

func (sck *socket) timeout() time.Duration {
	// FIXME(sbinet): extract from options
	return defaultTimeout
}

var (
	_ Socket = (*socket)(nil)
)
