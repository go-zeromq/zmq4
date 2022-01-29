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

// NewRep returns a new REP ZeroMQ socket.
// The returned socket value is initially unbound.
func NewRep(ctx context.Context, opts ...Option) Socket {
	rep := &repSocket{newSocket(ctx, Rep, opts...)}
	sharedState := newRepState()
	rep.sck.w = newRepWriter(rep.sck.ctx, sharedState)
	rep.sck.r = newRepReader(rep.sck.ctx, sharedState)
	return rep
}

// repSocket is a REP ZeroMQ socket.
type repSocket struct {
	sck *socket
}

// Close closes the open Socket
func (rep *repSocket) Close() error {
	return rep.sck.Close()
}

// Send puts the message on the outbound send queue.
// Send blocks until the message can be queued or the send deadline expires.
func (rep *repSocket) Send(msg Msg) error {
	ctx, cancel := context.WithTimeout(rep.sck.ctx, rep.sck.timeout())
	defer cancel()
	return rep.sck.w.write(ctx, msg)
}

// SendMulti puts the message on the outbound send queue.
// SendMulti blocks until the message can be queued or the send deadline expires.
// The message will be sent as a multipart message.
func (rep *repSocket) SendMulti(msg Msg) error {
	msg.multipart = true
	ctx, cancel := context.WithTimeout(rep.sck.ctx, rep.sck.timeout())
	defer cancel()
	return rep.sck.w.write(ctx, msg)
}

// Recv receives a complete message.
func (rep *repSocket) Recv() (Msg, error) {
	ctx, cancel := context.WithCancel(rep.sck.ctx)
	defer cancel()
	var msg Msg
	err := rep.sck.r.read(ctx, &msg)
	return msg, err
}

// Listen connects a local endpoint to the Socket.
func (rep *repSocket) Listen(ep string) error {
	return rep.sck.Listen(ep)
}

// Dial connects a remote endpoint to the Socket.
func (rep *repSocket) Dial(ep string) error {
	return rep.sck.Dial(ep)
}

// Type returns the type of this Socket (PUB, SUB, ...)
func (rep *repSocket) Type() SocketType {
	return rep.sck.Type()
}

// Addr returns the listener's address.
// Addr returns nil if the socket isn't a listener.
func (rep *repSocket) Addr() net.Addr {
	return rep.sck.Addr()
}

// GetOption is used to retrieve an option for a socket.
func (rep *repSocket) GetOption(name string) (interface{}, error) {
	return rep.sck.GetOption(name)
}

// SetOption is used to set an option for a socket.
func (rep *repSocket) SetOption(name string, value interface{}) error {
	return rep.sck.SetOption(name, value)
}

type repMsg struct {
	conn *Conn
	msg  Msg
}

type repReader struct {
	ctx   context.Context
	state *repState

	mu    sync.Mutex
	conns []*Conn

	msgCh chan repMsg
}

func newRepReader(ctx context.Context, state *repState) *repReader {
	const qsize = 10
	return &repReader{
		ctx:   ctx,
		msgCh: make(chan repMsg, qsize),
		state: state,
	}
}

func (r *repReader) addConn(c *Conn) {
	r.mu.Lock()
	r.conns = append(r.conns, c)
	r.mu.Unlock()
	go r.listen(r.ctx, c)
}

func (r *repReader) rmConn(conn *Conn) {
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
}

func (r *repReader) read(ctx context.Context, msg *Msg) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case repMsg := <-r.msgCh:
		if repMsg.msg.err != nil {
			return repMsg.msg.err
		}
		pre, innerMsg := splitReq(repMsg.msg)
		if pre == nil {
			return fmt.Errorf("zmq4: invalid REP message")
		}
		*msg = innerMsg
		r.state.Set(repMsg.conn, pre)
	}
	return nil
}

func (r *repReader) listen(ctx context.Context, conn *Conn) {
	defer r.rmConn(conn)
	defer conn.Close()

	for {
		msg := conn.read()
		select {
		case <-ctx.Done():
			return
		default:
			if msg.err != nil {
				return
			}
			r.msgCh <- repMsg{conn, msg}
		}
	}
}

func (r *repReader) Close() error {
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

func splitReq(envelope Msg) (preamble [][]byte, msg Msg) {
	for i, frame := range envelope.Frames {
		if len(frame) != 0 {
			continue
		}
		preamble = envelope.Frames[:i+1]
		if i+1 < len(envelope.Frames) {
			msg = NewMsgFrom(envelope.Frames[i+1:]...)
		}
	}
	return
}

type repSendPayload struct {
	conn     *Conn
	preamble [][]byte
	msg      Msg
}

type repWriter struct {
	ctx   context.Context
	state *repState

	mu    sync.Mutex
	conns []*Conn

	sendCh chan repSendPayload
}

func (r repSendPayload) buildReplyMsg() Msg {
	var frames = make([][]byte, 0, len(r.preamble)+len(r.msg.Frames))
	frames = append(frames, r.preamble...)
	frames = append(frames, r.msg.Frames...)
	return NewMsgFrom(frames...)
}

func newRepWriter(ctx context.Context, state *repState) *repWriter {
	r := &repWriter{
		ctx:    ctx,
		state:  state,
		sendCh: make(chan repSendPayload),
	}
	go r.run()
	return r
}

func (r *repWriter) addConn(w *Conn) {
	r.mu.Lock()
	r.conns = append(r.conns, w)
	r.mu.Unlock()
}

func (r *repWriter) rmConn(conn *Conn) {
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
}

func (r *repWriter) write(ctx context.Context, msg Msg) error {
	conn, preamble := r.state.Get()
	r.sendCh <- repSendPayload{conn, preamble, msg}
	return nil
}

func (r *repWriter) run() {
	for {
		select {
		case <-r.ctx.Done():
			return
		case payload, ok := <-r.sendCh:
			if !ok {
				return
			}
			r.sendPayload(payload)
		}
	}
}

func (r *repWriter) sendPayload(payload repSendPayload) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, conn := range r.conns {
		if conn == payload.conn {
			reply := payload.buildReplyMsg()
			// not much we can do at this point. Perhaps log the error?
			_ = conn.SendMsg(reply)
			return
		}
	}
}

func (r *repWriter) Close() error {
	close(r.sendCh)
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

type repState struct {
	mu       sync.Mutex
	conn     *Conn
	preamble [][]byte // includes delimiter
}

func newRepState() *repState {
	return &repState{}
}

func (r *repState) Get() (conn *Conn, preamble [][]byte) {
	r.mu.Lock()
	conn = r.conn
	preamble = r.preamble
	r.mu.Unlock()
	return
}

func (r *repState) Set(conn *Conn, pre [][]byte) {
	r.mu.Lock()
	r.conn = conn
	r.preamble = pre
	r.mu.Unlock()
}

var (
	_ Socket = (*repSocket)(nil)
)
