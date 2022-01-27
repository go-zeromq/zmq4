// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	defaultRetry   = 250 * time.Millisecond
	defaultTimeout = 5 * time.Minute
)

var (
	errInvalidAddress = errors.New("zmq4: invalid address")

	ErrBadProperty = errors.New("zmq4: bad property")
)

// socket implements the ZeroMQ socket interface
type socket struct {
	ep    string // socket end-point
	typ   SocketType
	id    SocketIdentity
	retry time.Duration
	sec   Security
	log   *log.Logger

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

	closedConns []*Conn
	reaperCond  *sync.Cond
}

func newDefaultSocket(ctx context.Context, sockType SocketType) *socket {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithCancel(ctx)
	return &socket{
		typ:        sockType,
		retry:      defaultRetry,
		sec:        nullSecurity{},
		ids:        make(map[string]*Conn),
		conns:      nil,
		r:          newQReader(ctx),
		w:          newMWriter(ctx),
		props:      make(map[string]interface{}),
		ctx:        ctx,
		cancel:     cancel,
		dialer:     net.Dialer{Timeout: defaultTimeout},
		reaperCond: sync.NewCond(&sync.Mutex{}),
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
	if sck.log == nil {
		sck.log = log.New(os.Stderr, "zmq4: ", 0)
	}

	return sck
}

func (sck *socket) topics() []string {
	var (
		keys   = make(map[string]struct{})
		topics []string
	)
	sck.mu.RLock()
	for _, con := range sck.conns {
		con.mu.RLock()
		for topic := range con.topics {
			if _, dup := keys[topic]; dup {
				continue
			}
			keys[topic] = struct{}{}
			topics = append(topics, topic)
		}
		con.mu.RUnlock()
	}
	sck.mu.RUnlock()
	sort.Strings(topics)
	return topics
}

// Close closes the open Socket
func (sck *socket) Close() error {
	sck.cancel()
	sck.reaperCond.Signal()

	if sck.listener != nil {
		defer sck.listener.Close()
	}

	sck.mu.RLock()
	defer sck.mu.RUnlock()

	var err error
	for _, conn := range sck.conns {
		e := conn.Close()
		if e != nil && err == nil {
			err = e
		}
	}

	// Remove the unix socket file if created by net.Listen
	if sck.listener != nil && strings.HasPrefix(sck.ep, "ipc://") {
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

// SendMulti puts the message on the outbound send queue.
// SendMulti blocks until the message can be queued or the send deadline expires.
// The message will be sent as a multipart message.
func (sck *socket) SendMulti(msg Msg) error {
	msg.multipart = true
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

	trans, ok := drivers.get(network)
	switch {
	case ok:
		l, err = trans.Listen(sck.ctx, addr)
	default:
		panic("zmq4: unknown protocol " + network)
	}

	if err != nil {
		return fmt.Errorf("zmq4: could not listen to %q: %w", endpoint, err)
	}
	sck.listener = l

	go sck.accept()
	go sck.connReaper()

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
				// FIXME(sbinet): maybe bubble up this error to application code?
				//sck.log.Printf("error accepting connection from %q: %+v", sck.ep, err)
				continue
			}

			zconn, err := Open(conn, sck.sec, sck.typ, sck.id, true, sck.scheduleRmConn)
			if err != nil {
				// FIXME(sbinet): maybe bubble up this error to application code?
				sck.log.Printf("could not open a ZMTP connection with %q: %+v", sck.ep, err)
				continue
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

	var (
		conn      net.Conn
		trans, ok = drivers.get(network)
		retries   = 0
	)
connect:
	switch {
	case ok:
		conn, err = trans.Dial(sck.ctx, &sck.dialer, addr)
	default:
		panic("zmq4: unknown protocol " + network)
	}

	if err != nil {
		if retries < 10 {
			retries++
			time.Sleep(sck.retry)
			goto connect
		}
		return fmt.Errorf("zmq4: could not dial to %q (retry=%v): %w", endpoint, sck.retry, err)
	}

	if conn == nil {
		return fmt.Errorf("zmq4: got a nil dial-conn to %q", endpoint)
	}

	zconn, err := Open(conn, sck.sec, sck.typ, sck.id, false, sck.scheduleRmConn)
	if err != nil {
		return fmt.Errorf("zmq4: could not open a ZMTP connection: %w", err)
	}
	if zconn == nil {
		return fmt.Errorf("zmq4: got a nil ZMTP connection to %q", endpoint)
	}

	go sck.connReaper()
	sck.addConn(zconn)
	return nil
}

func (sck *socket) addConn(c *Conn) {
	sck.mu.Lock()
	sck.conns = append(sck.conns, c)
	uuid, ok := c.Peer.Meta[sysSockID]
	if !ok {
		uuid = newUUID()
		c.Peer.Meta[sysSockID] = uuid
	}
	sck.ids[uuid] = c
	if sck.w != nil {
		sck.w.addConn(c)
	}
	if sck.r != nil {
		sck.r.addConn(c)
	}
	sck.mu.Unlock()
}

func (sck *socket) rmConn(c *Conn) {
	sck.mu.Lock()
	defer sck.mu.Unlock()

	cur := -1
	for i := range sck.conns {
		if sck.conns[i] == c {
			cur = i
			break
		}
	}

	if cur == -1 {
		return
	}

	sck.conns = append(sck.conns[:cur], sck.conns[cur+1:]...)
	if sck.r != nil {
		sck.r.rmConn(c)
	}
	if sck.w != nil {
		sck.w.rmConn(c)
	}
}

func (sck *socket) scheduleRmConn(c *Conn) {
	sck.reaperCond.L.Lock()
	sck.closedConns = append(sck.closedConns, c)
	sck.reaperCond.Signal()
	sck.reaperCond.L.Unlock()
}

// Type returns the type of this Socket (PUB, SUB, ...)
func (sck *socket) Type() SocketType {
	return sck.typ
}

// Addr returns the listener's address.
// Addr returns nil if the socket isn't a listener.
func (sck *socket) Addr() net.Addr {
	if sck.listener == nil {
		return nil
	}
	return sck.listener.Addr()
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

func (sck *socket) connReaper() {
	sck.reaperCond.L.Lock()
	defer sck.reaperCond.L.Unlock()

	for {
		for len(sck.closedConns) == 0 && sck.ctx.Err() == nil {
			sck.reaperCond.Wait()
		}

		if sck.ctx.Err() != nil {
			return
		}

		for _, c := range sck.closedConns {
			sck.rmConn(c)
		}
		sck.closedConns = nil
	}
}

var (
	_ Socket = (*socket)(nil)
)
