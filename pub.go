// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"context"
	"sync"

	"golang.org/x/sync/errgroup"
	"golang.org/x/xerrors"
)

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
	ctx, cancel := context.WithTimeout(pub.sck.ctx, pub.sck.timeout())
	defer cancel()
	return pub.sck.w.write(ctx, msg)
}

// Recv receives a complete message.
func (*pubSocket) Recv() (Msg, error) {
	msg := Msg{err: xerrors.Errorf("zmq4: PUB sockets can't recv messages")}
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

// GetOption is used to retrieve an option for a socket.
func (pub *pubSocket) GetOption(name string) (interface{}, error) {
	return pub.sck.GetOption(name)
}

// SetOption is used to set an option for a socket.
func (pub *pubSocket) SetOption(name string, value interface{}) error {
	return pub.sck.SetOption(name, value)
}

// pubQReader is a queued-message reader.
type pubQReader struct {
	ctx context.Context

	mu sync.RWMutex
	rs []*msgReader
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

func (q *pubQReader) addConn(r *msgReader) {
	go q.listen(q.ctx, r)
	q.mu.Lock()
	q.sem.enable()
	q.rs = append(q.rs, r)
	q.mu.Unlock()
}

func (q *pubQReader) rmConn(r *msgReader) {
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
	q.sem.lock()
	select {
	case <-ctx.Done():
	case *msg = <-q.c:
	}
	return msg.err
}

func (q *pubQReader) listen(ctx context.Context, r *msgReader) {
	defer q.rmConn(r)
	defer r.Close()

	for {
		var msg Msg
		err := r.read(ctx, &msg)
		select {
		case <-ctx.Done():
			return
		default:
			if err != nil {
				return
			}
			switch {
			case q.topic(msg):
				r.r.subscribe(msg)
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
	ctx context.Context
	mu  sync.Mutex
	ws  []*msgWriter
}

func newPubMWriter(ctx context.Context) *pubMWriter {
	return &pubMWriter{
		ctx: ctx,
	}
}

func (w *pubMWriter) Close() error {
	w.mu.Lock()
	var err error
	for _, ww := range w.ws {
		e := ww.Close()
		if e != nil && err == nil {
			err = e
		}
	}
	w.ws = nil
	w.mu.Unlock()
	return err
}

func (mw *pubMWriter) addConn(w *msgWriter) {
	mw.mu.Lock()
	mw.ws = append(mw.ws, w)
	mw.mu.Unlock()
}

func (mw *pubMWriter) rmConn(w *msgWriter) {
	mw.mu.Lock()
	defer mw.mu.Unlock()

	cur := -1
	for i := range mw.ws {
		if mw.ws[i] == w {
			cur = i
			break
		}
	}
	if cur >= 0 {
		mw.ws = append(mw.ws[:cur], mw.ws[cur+1:]...)
	}
}

func (w *pubMWriter) write(ctx context.Context, msg Msg) error {
	grp, ctx := errgroup.WithContext(ctx)
	w.mu.Lock()
	topic := string(msg.Frames[0])
	for i := range w.ws {
		ww := w.ws[i]
		grp.Go(func() error {
			if !ww.w.subscribed(topic) {
				return nil
			}
			return ww.write(ctx, msg)
		})
	}
	err := grp.Wait()
	w.mu.Unlock()
	return err
}

var (
	_ rpool  = (*pubQReader)(nil)
	_ wpool  = (*pubMWriter)(nil)
	_ Socket = (*pubSocket)(nil)
)
