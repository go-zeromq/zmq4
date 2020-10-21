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

	addConn(r *Conn)
	rmConn(r *Conn)
	read(ctx context.Context, msg *Msg) error
}

// wpool is the interface that writes ZMQ messages to a pool of connections.
type wpool interface {
	io.Closer

	addConn(w *Conn)
	rmConn(r *Conn)
	write(ctx context.Context, msg Msg) error
}

// qreader is a queued-message reader.
type qreader struct {
	ctx context.Context
	mu  sync.RWMutex
	rs  []*Conn
	c   chan Msg

	sem *semaphore // ready when a connection is live.
}

func newQReader(ctx context.Context) *qreader {
	const qrsize = 10
	return &qreader{
		ctx: ctx,
		c:   make(chan Msg, qrsize),
		sem: newSemaphore(),
	}
}

func (q *qreader) Close() error {
	q.mu.RLock()
	var err error
	var grp errgroup.Group
	for i := range q.rs {
		grp.Go(q.rs[i].Close)
	}
	err = grp.Wait()
	q.rs = nil
	q.mu.RUnlock()
	return err
}

func (q *qreader) addConn(r *Conn) {
	go q.listen(q.ctx, r)
	q.mu.Lock()
	q.sem.enable()
	q.rs = append(q.rs, r)
	q.mu.Unlock()
}

func (q *qreader) rmConn(r *Conn) {
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

func (q *qreader) read(ctx context.Context, msg *Msg) error {
	q.sem.lock()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case *msg = <-q.c:
	}
	return msg.err
}

func (q *qreader) listen(ctx context.Context, r *Conn) {
	defer q.rmConn(r)
	defer r.Close()

	for {
		msg := r.read()
		select {
		case <-ctx.Done():
			return
		default:
			q.c <- msg
			if msg.err != nil {
				return
			}
		}
	}
}

type mwriter struct {
	ctx context.Context
	mu  sync.Mutex
	ws  []*Conn
	sem *semaphore
}

func newMWriter(ctx context.Context) *mwriter {
	return &mwriter{
		ctx: ctx,
		sem: newSemaphore(),
	}
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

func (mw *mwriter) addConn(w *Conn) {
	mw.mu.Lock()
	mw.sem.enable()
	mw.ws = append(mw.ws, w)
	mw.mu.Unlock()
}

func (mw *mwriter) rmConn(w *Conn) {
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

func (w *mwriter) write(ctx context.Context, msg Msg) error {
	w.sem.lock()
	grp, _ := errgroup.WithContext(ctx)
	w.mu.Lock()
	for i := range w.ws {
		ww := w.ws[i]
		grp.Go(func() error {
			return ww.SendMsg(msg)
		})
	}
	err := grp.Wait()
	w.mu.Unlock()
	return err
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
)
