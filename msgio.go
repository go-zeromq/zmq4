// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"context"
	"io"
	"sync"

	"golang.org/x/sync/errgroup"
)

// rpool is the interface that reads ZMQ messages from a pool of connections.
type rpool interface {
	io.Closer

	addConn(r *msgReader)
	read(ctx context.Context, msg *Msg) error
}

// wpool is the interface that writes ZMQ messages to a pool of connections.
type wpool interface {
	io.Closer

	addConn(w *msgWriter)
	write(ctx context.Context, msg Msg) error
}

type msgReader struct {
	r *Conn
}

// read reads data over the wire and assembles it into a complete message
func (r *msgReader) read(ctx context.Context, msg *Msg) error {
	*msg = r.r.read()
	return msg.err
}

func (r *msgReader) Close() error {
	return r.r.Close()
}

type msgWriter struct {
	w *Conn
}

func (w *msgWriter) Close() error {
	return w.w.Close()
}

// write sends data over the wire.
func (w *msgWriter) write(ctx context.Context, msg Msg) error {
	err := w.w.SendMsg(msg)
	return err
}

// qreader is a queued-message reader.
type qreader struct {
	mu sync.RWMutex
	rs []*msgReader
	c  chan Msg

	sem *semaphore // ready when a connection is live.
}

func newQReader() *qreader {
	const qrsize = 10
	return &qreader{
		c:   make(chan Msg, qrsize),
		sem: newSemaphore(),
	}
}

func (q *qreader) Close() error {
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

func (q *qreader) addConn(r *msgReader) {
	go rlisten(context.Background(), r, q.c)
	q.mu.Lock()
	q.sem.enable()
	q.rs = append(q.rs, r)
	q.mu.Unlock()
}

func (q *qreader) read(ctx context.Context, msg *Msg) error {
	q.sem.lock()
	select {
	case <-ctx.Done():
	case *msg = <-q.c:
	}
	return msg.err
}

func rlisten(ctx context.Context, r *msgReader, ch chan Msg) {
	for {
		var msg Msg
		err := r.read(ctx, &msg)
		select {
		case <-ctx.Done():
			return
		default:
			ch <- msg
			if err != nil {
				return
			}
		}
	}
}

type mwriter struct {
	mu  sync.Mutex
	ws  []*msgWriter
	sem *semaphore
}

func newMWriter() *mwriter {
	return &mwriter{sem: newSemaphore()}
}

func (w *mwriter) Close() error {
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

func (mw *mwriter) addConn(w *msgWriter) {
	mw.mu.Lock()
	mw.sem.enable()
	mw.ws = append(mw.ws, w)
	mw.mu.Unlock()
}

func (w *mwriter) write(ctx context.Context, msg Msg) error {
	w.sem.lock()
	grp, ctx := errgroup.WithContext(ctx)
	w.mu.Lock()
	for i := range w.ws {
		ww := w.ws[i]
		grp.Go(func() error {
			return ww.write(ctx, msg)
		})
	}
	err := grp.Wait()
	w.mu.Unlock()
	return err
}

type lbwriter struct {
	c   chan Msg
	sem *semaphore
}

func newLBWriter() *lbwriter {
	const size = 10
	return &lbwriter{
		c:   make(chan Msg, size),
		sem: newSemaphore(),
	}
}

func (lw *lbwriter) Close() error {
	close(lw.c)
	return nil
}

func (lw *lbwriter) addConn(w *msgWriter) {
	lw.sem.enable()
	go wlisten(context.Background(), w, lw.c)
}

func (lw *lbwriter) write(ctx context.Context, msg Msg) error {
	lw.sem.lock()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case lw.c <- msg:
		return nil
	}
}

func wlisten(ctx context.Context, w *msgWriter, c chan Msg) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-c:
			if !ok {
				return
			}
			err := w.write(ctx, msg)
			if err != nil {
				// try another msg writer
				c <- msg
				break
			}
		}
	}
}

type semaphore struct {
	ready chan struct{}
}

func newSemaphore() *semaphore {
	return &semaphore{ready: make(chan struct{})}
}

func (sem *semaphore) enable() {
	select {
	case _, ok := <-sem.ready:
		if ok {
			close(sem.ready)
		}
	default:
		close(sem.ready)
	}
}

func (sem *semaphore) lock() {
	<-sem.ready
}

var (
	_ rpool = (*qreader)(nil)
	_ wpool = (*mwriter)(nil)
	_ wpool = (*lbwriter)(nil)
)
