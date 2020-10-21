// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"context"
	"fmt"
	"net"
	"sync"
)

// NewReq returns a new REQ ZeroMQ socket.
// The returned socket value is initially unbound.
func NewReq(ctx context.Context, opts ...Option) Socket {
	state := &reqState{}
	req := &reqSocket{newSocket(ctx, Req, opts...), state}
	req.sck.r = newReqReader(req.sck.ctx, state)
	req.sck.w = newReqWriter(req.sck.ctx, state)
	return req
}

// reqSocket is a REQ ZeroMQ socket.
type reqSocket struct {
	sck   *socket
	state *reqState
}

// Close closes the open Socket
func (req *reqSocket) Close() error {
	return req.sck.Close()
}

// Send puts the message on the outbound send queue.
// Send blocks until the message can be queued or the send deadline expires.
func (req *reqSocket) Send(msg Msg) error {
	ctx, cancel := context.WithTimeout(req.sck.ctx, req.sck.timeout())
	defer cancel()
	return req.sck.w.write(ctx, msg)
}

// SendMulti puts the message on the outbound send queue.
// SendMulti blocks until the message can be queued or the send deadline expires.
// The message will be sent as a multipart message.
func (req *reqSocket) SendMulti(msg Msg) error {
	msg.multipart = true
	ctx, cancel := context.WithTimeout(req.sck.ctx, req.sck.timeout())
	defer cancel()
	return req.sck.w.write(ctx, msg)
}

// Recv receives a complete message.
func (req *reqSocket) Recv() (Msg, error) {
	ctx, cancel := context.WithCancel(req.sck.ctx)
	defer cancel()
	var msg Msg
	err := req.sck.r.read(ctx, &msg)
	return msg, err
}

// Listen connects a local endpoint to the Socket.
func (req *reqSocket) Listen(ep string) error {
	return req.sck.Listen(ep)
}

// Dial connects a remote endpoint to the Socket.
func (req *reqSocket) Dial(ep string) error {
	return req.sck.Dial(ep)
}

// Type returns the type of this Socket (PUB, SUB, ...)
func (req *reqSocket) Type() SocketType {
	return req.sck.Type()
}

// Addr returns the listener's address.
// Addr returns nil if the socket isn't a listener.
func (req *reqSocket) Addr() net.Addr {
	return req.sck.Addr()
}

// GetOption is used to retrieve an option for a socket.
func (req *reqSocket) GetOption(name string) (interface{}, error) {
	return req.sck.GetOption(name)
}

// SetOption is used to set an option for a socket.
func (req *reqSocket) SetOption(name string, value interface{}) error {
	return req.sck.SetOption(name, value)
}

type reqWriter struct {
	mu       sync.Mutex
	conns    []*Conn
	nextConn int
	state    *reqState
}

func newReqWriter(ctx context.Context, state *reqState) *reqWriter {
	return &reqWriter{
		state: state,
	}
}

func (r *reqWriter) write(ctx context.Context, msg Msg) error {
	msg.Frames = append([][]byte{nil}, msg.Frames...)

	r.mu.Lock()
	defer r.mu.Unlock()
	var err error
	for i := 0; i < len(r.conns); i++ {
		cur := i + r.nextConn%len(r.conns)
		conn := r.conns[cur]
		err = conn.SendMsg(msg)
		if err == nil {
			r.nextConn = cur + 1%len(r.conns)
			r.state.Set(conn)
			return nil
		}
	}
	return fmt.Errorf("zmq4: no connections available: %w", err)
}

func (r *reqWriter) addConn(c *Conn) {
	r.mu.Lock()
	r.conns = append(r.conns, c)
	r.mu.Unlock()
}

func (r *reqWriter) rmConn(conn *Conn) {
	r.mu.Lock()
	defer r.mu.Unlock()

	cur := -1
	for i := range r.conns {
		if r.conns[i] == conn {
			cur = i
			break
		}
	}
	if cur >= 0 {
		r.conns = append(r.conns[:cur], r.conns[cur+1:]...)
	}

	r.state.Reset(conn)
}

func (r *reqWriter) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var err error
	for _, conn := range r.conns {
		e := conn.Close()
		if e != nil && err == nil {
			err = e
		}
	}
	r.conns = nil
	return err
}

type reqReader struct {
	state *reqState
}

func newReqReader(ctx context.Context, state *reqState) *reqReader {
	return &reqReader{
		state: state,
	}
}

func (r *reqReader) addConn(c *Conn) {}
func (r *reqReader) rmConn(c *Conn)  {}

func (r *reqReader) Close() error {
	return nil
}

func (r *reqReader) read(ctx context.Context, msg *Msg) error {
	curConn := r.state.Get()
	if curConn == nil {
		return fmt.Errorf("zmq4: no connections available")
	}
	*msg = curConn.read()
	if msg.err != nil {
		return msg.err
	}
	if len(msg.Frames) > 1 {
		msg.Frames = msg.Frames[1:]
	}
	return nil
}

type reqState struct {
	mu       sync.Mutex
	lastConn *Conn
}

func (r *reqState) Set(conn *Conn) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastConn = conn
}

// Reset resets the state iff c matches the resident connection
func (r *reqState) Reset(c *Conn) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.lastConn == c {
		r.lastConn = nil
	}
}

func (r *reqState) Get() *Conn {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.lastConn
}

var (
	_ Socket = (*reqSocket)(nil)
)
