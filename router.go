// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"bytes"
	"context"
	"sync"

	"golang.org/x/sync/errgroup"
)

// NewRouter returns a new ROUTER ZeroMQ socket.
// The returned socket value is initially unbound.
func NewRouter(ctx context.Context, opts ...Option) Socket {
	router := &routerSocket{newSocket(ctx, Router, opts...)}
	router.sck.r = newRouterQReader(router.sck.ctx)
	router.sck.w = newRouterMWriter(router.sck.ctx)
	return router
}

// routerSocket is a ROUTER ZeroMQ socket.
type routerSocket struct {
	sck *socket
}

// Close closes the open Socket
func (router *routerSocket) Close() error {
	return router.sck.Close()
}

// Send puts the message on the outbound send queue.
// Send blocks until the message can be queued or the send deadline expires.
func (router *routerSocket) Send(msg Msg) error {
	ctx, cancel := context.WithTimeout(router.sck.ctx, router.sck.timeout())
	defer cancel()
	return router.sck.w.write(ctx, msg)
}

// Recv receives a complete message.
func (router *routerSocket) Recv() (Msg, error) {
	return router.sck.Recv()
}

// Listen connects a local endpoint to the Socket.
func (router *routerSocket) Listen(ep string) error {
	return router.sck.Listen(ep)
}

// Dial connects a remote endpoint to the Socket.
func (router *routerSocket) Dial(ep string) error {
	return router.sck.Dial(ep)
}

// Type returns the type of this Socket (PUB, SUB, ...)
func (router *routerSocket) Type() SocketType {
	return router.sck.Type()
}

// GetOption is used to retrieve an option for a socket.
func (router *routerSocket) GetOption(name string) (interface{}, error) {
	return router.sck.GetOption(name)
}

// SetOption is used to set an option for a socket.
func (router *routerSocket) SetOption(name string, value interface{}) error {
	return router.sck.SetOption(name, value)
}

// routerQReader is a queued-message reader.
type routerQReader struct {
	ctx context.Context

	mu sync.RWMutex
	rs []*msgReader
	c  chan Msg

	sem *semaphore // ready when a connection is live.
}

func newRouterQReader(ctx context.Context) *routerQReader {
	const qrsize = 10
	return &routerQReader{
		ctx: ctx,
		c:   make(chan Msg, qrsize),
		sem: newSemaphore(),
	}
}

func (q *routerQReader) Close() error {
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

func (q *routerQReader) addConn(r *msgReader) {
	go q.listen(q.ctx, r)
	q.mu.Lock()
	q.sem.enable()
	q.rs = append(q.rs, r)
	q.mu.Unlock()
}

func (q *routerQReader) read(ctx context.Context, msg *Msg) error {
	q.sem.lock()
	select {
	case <-ctx.Done():
	case *msg = <-q.c:
	}
	return msg.err
}

func (q *routerQReader) listen(ctx context.Context, r *msgReader) {
	id := []byte(r.r.Peer.Meta[sysSockID])
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
			msg.Frames = append([][]byte{id}, msg.Frames...)
			q.c <- msg
		}
	}
}

type routerMWriter struct {
	ctx context.Context
	mu  sync.Mutex
	ws  []*msgWriter
	sem *semaphore
}

func newRouterMWriter(ctx context.Context) *routerMWriter {
	return &routerMWriter{
		ctx: ctx,
		sem: newSemaphore(),
	}
}

func (w *routerMWriter) Close() error {
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

func (mw *routerMWriter) addConn(w *msgWriter) {
	mw.mu.Lock()
	mw.sem.enable()
	mw.ws = append(mw.ws, w)
	mw.mu.Unlock()
}

func (w *routerMWriter) write(ctx context.Context, msg Msg) error {
	w.sem.lock()
	grp, ctx := errgroup.WithContext(ctx)
	w.mu.Lock()
	id := msg.Frames[0]
	dmsg := NewMsgFrom(msg.Frames[1:]...)
	for i := range w.ws {
		ww := w.ws[i]
		pid := []byte(ww.w.Peer.Meta[sysSockID])
		if !bytes.Equal(pid, id) {
			continue
		}
		grp.Go(func() error {
			return ww.write(ctx, dmsg)
		})
	}
	err := grp.Wait()
	w.mu.Unlock()
	return err
}

var (
	_ rpool  = (*routerQReader)(nil)
	_ wpool  = (*routerMWriter)(nil)
	_ Socket = (*routerSocket)(nil)
)
