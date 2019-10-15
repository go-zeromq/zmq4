// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"bytes"
	"context"
	"net"
	"sync"

	"golang.org/x/sync/errgroup"
	"golang.org/x/xerrors"
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

// Addr returns the listener's address.
// Addr returns nil if the socket isn't a listener.
func (router *routerSocket) Addr() net.Addr {
	return router.sck.Addr()
}

// GetOption is used to retrieve an option for a socket.
func (router *routerSocket) GetOption(name string) (interface{}, error) {
	return router.sck.GetOption(name)
}

// SetOption is used to set an option for a socket.
func (router *routerSocket) SetOption(name string, value interface{}) error {
	return router.sck.SetOption(name, value)
}

// GetTopics is used to retrieve subscribed topics for a pub socket.
func (router *routerSocket) GetTopics(filter bool) ([]string, error) {
	err := xerrors.Errorf("zmq4: Only available for PUB sockets")
	return nil, err
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

func (q *routerQReader) rmConn(r *msgReader) {
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

func (q *routerQReader) read(ctx context.Context, msg *Msg) error {
	q.sem.lock()
	select {
	case <-ctx.Done():
	case *msg = <-q.c:
	}
	return msg.err
}

func (q *routerQReader) listen(ctx context.Context, r *msgReader) {
	defer q.rmConn(r)
	defer r.Close()

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

func (mw *routerMWriter) rmConn(w *msgWriter) {
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
