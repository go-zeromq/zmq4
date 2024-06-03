// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
)

const (
	DefaultSendHwm = 1000
)

// Topics is an interface that wraps the basic Topics method.
type Topics interface {
	// Topics returns the sorted list of topics a socket is subscribed to.
	Topics() []string
}

// NewPub returns a new PUB ZeroMQ socket.
// The returned socket value is initially unbound.
func NewPub(ctx context.Context, opts ...Option) Socket {
	pub := &pubSocket{sck: newSocket(ctx, Pub, opts...)}
	pub.sck.w = newPubMWriter(pub.sck.ctx)
	pub.sck.r = newPubQReader(pub.sck.ctx)
	return pub
}

// pubSocket is a PUB ZeroMQ socket.
type pubSocket struct {
	sck *socket
}

// Close closes the open Socket
func (pub *pubSocket) Close() error {
	return pub.sck.Close()
}

// Send puts the message on the outbound send queue.
// Send blocks until the message can be queued or the send deadline expires.
func (pub *pubSocket) Send(msg Msg) error {
	ctx, cancel := context.WithTimeout(pub.sck.ctx, pub.sck.Timeout())
	defer cancel()
	return pub.sck.w.write(ctx, msg)
}

// SendMulti puts the message on the outbound send queue.
// SendMulti blocks until the message can be queued or the send deadline expires.
// The message will be sent as a multipart message.
func (pub *pubSocket) SendMulti(msg Msg) error {
	msg.multipart = true
	ctx, cancel := context.WithTimeout(pub.sck.ctx, pub.sck.Timeout())
	defer cancel()
	return pub.sck.w.write(ctx, msg)
}

// Recv receives a complete message.
func (*pubSocket) Recv() (Msg, error) {
	msg := Msg{err: fmt.Errorf("zmq4: PUB sockets can't recv messages")}
	return msg, msg.err
}

// Listen connects a local endpoint to the Socket.
func (pub *pubSocket) Listen(ep string) error {
	return pub.sck.Listen(ep)
}

// Dial connects a remote endpoint to the Socket.
func (pub *pubSocket) Dial(ep string) error {
	return pub.sck.Dial(ep)
}

// Type returns the type of this Socket (PUB, SUB, ...)
func (pub *pubSocket) Type() SocketType {
	return pub.sck.Type()
}

// Addr returns the listener's address.
// Addr returns nil if the socket isn't a listener.
func (pub *pubSocket) Addr() net.Addr {
	return pub.sck.Addr()
}

// GetOption is used to retrieve an option for a socket.
func (pub *pubSocket) GetOption(name string) (interface{}, error) {
	return pub.sck.GetOption(name)
}

// SetOption is used to set an option for a socket.
func (pub *pubSocket) SetOption(name string, value interface{}) error {
	err := pub.sck.SetOption(name, value)
	if err != nil {
		return err
	}

	if name != OptionHWM {
		return ErrBadProperty
	}

	hwm, ok := value.(int)
	if !ok {
		return ErrBadProperty
	}

	w := pub.sck.w.(*pubMWriter)
	w.hwm.Store(int64(hwm))
	return nil
}

// Topics returns the sorted list of topics a socket is subscribed to.
func (pub *pubSocket) Topics() []string {
	return pub.sck.topics()
}

// pubQReader is a queued-message reader.
type pubQReader struct {
	ctx context.Context

	mu sync.RWMutex
	rs []*Conn
	c  chan Msg

	sem *semaphore // ready when a connection is live.
}

func newPubQReader(ctx context.Context) *pubQReader {
	const qrsize = 10
	return &pubQReader{
		ctx: ctx,
		c:   make(chan Msg, qrsize),
		sem: newSemaphore(),
	}
}

func (q *pubQReader) Close() error {
	q.mu.RLock()
	var err error
	for _, r := range q.rs {
		e := r.Close()
		if e != nil && err == nil {
			err = e
		}
	}
	q.rs = nil
	q.mu.RUnlock()
	return err
}

func (q *pubQReader) addConn(r *Conn) {
	q.mu.Lock()
	q.sem.enable()
	q.rs = append(q.rs, r)
	q.mu.Unlock()
	go q.listen(q.ctx, r)
}

func (q *pubQReader) rmConn(r *Conn) {
	q.mu.Lock()
	defer q.mu.Unlock()

	cur := -1
	for i := range q.rs {
		if q.rs[i] == r {
			cur = i
			break
		}
	}
	if cur >= 0 {
		q.rs = append(q.rs[:cur], q.rs[cur+1:]...)
	}
}

func (q *pubQReader) read(ctx context.Context, msg *Msg) error {
	q.sem.lock(ctx)
	select {
	case <-ctx.Done():
	case *msg = <-q.c:
	}
	return msg.err
}

func (q *pubQReader) listen(ctx context.Context, r *Conn) {
	defer q.rmConn(r)
	defer r.Close()

	for {
		msg := r.read()
		select {
		case <-ctx.Done():
			return
		default:
			if msg.err != nil {
				return
			}
			switch {
			case q.topic(msg):
				r.subscribe(msg)
			default:
				q.c <- msg
			}
		}
	}
}

func (q *pubQReader) topic(msg Msg) bool {
	if len(msg.Frames) != 1 {
		return false
	}
	frame := msg.Frames[0]
	if len(frame) == 0 {
		return false
	}
	topic := frame[0]
	return topic == 0 || topic == 1
}

type pubMWriter struct {
	ctx         context.Context
	mu          sync.RWMutex
	subscribers map[*Conn]chan Msg

	hwm atomic.Int64
}

func newPubMWriter(ctx context.Context) *pubMWriter {
	p := &pubMWriter{
		ctx:         ctx,
		subscribers: map[*Conn]chan Msg{},
	}
	p.hwm.Store(DefaultSendHwm)
	return p
}

func (w *pubMWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	for conn := range w.subscribers {
		_ = conn.Close()
	}
	w.subscribers = nil
	return nil
}

func (mw *pubMWriter) addConn(w *Conn) {
	mw.mu.Lock()
	defer mw.mu.Unlock()

	c := make(chan Msg, mw.hwm.Load())
	mw.subscribers[w] = c
	go func() {
		for {
			msg, ok := <-c
			if !ok {
				break
			}
			topic := string(msg.Frames[0])
			if w.subscribed(topic) {
				_ = w.SendMsg(msg)
			}
		}
	}()
}

func (mw *pubMWriter) rmConn(w *Conn) {
	mw.mu.Lock()
	defer mw.mu.Unlock()

	if channel, ok := mw.subscribers[w]; ok {
		_ = w.Close()
		delete(mw.subscribers, w)
		close(channel)
	}
}

func (w *pubMWriter) write(ctx context.Context, msg Msg) error {
	w.mu.RLock()
	defer w.mu.RUnlock()

	for _, channel := range w.subscribers {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case channel <- msg: // proceeds to default case if the channel is full (msg will be discarded)
		default:
		}
	}
	return nil
}

var (
	_ rpool  = (*pubQReader)(nil)
	_ wpool  = (*pubMWriter)(nil)
	_ Socket = (*pubSocket)(nil)
	_ Topics = (*pubSocket)(nil)
)
