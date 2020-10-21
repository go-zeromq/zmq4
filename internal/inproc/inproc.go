// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package inproc provides tools to implement an in-process asynchronous pipe of net.Conns.
package inproc

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
)

var (
	mgr = contextType{db: make(map[string]*Listener)}

	ErrClosed      = errors.New("inproc: connection closed")
	ErrConnRefused = errors.New("inproc: connection refused")
)

func init() {
	mgr.cv.L = &mgr.mu
}

type contextType struct {
	mu sync.Mutex
	cv sync.Cond
	db map[string]*Listener
}

// A Listener is an in-process listener for stream-oriented protocols.
// Listener implements net.Listener.
//
// Multiple goroutines may invoke methods on a Listener simultaneously.
type Listener struct {
	addr Addr

	pipes  []*pipe
	closed bool
}

type pipe struct {
	p1 *conn
	p2 *conn
}

func newPipe(addr Addr) *pipe {
	const sz = 8
	ch1 := make(chan []byte, sz)
	ch2 := make(chan []byte, sz)
	done1 := make(chan struct{})
	done2 := make(chan struct{})

	p1 := &conn{
		addr:       addr,
		r:          ch1,
		w:          ch2,
		localDone:  done1,
		remoteDone: done2,
		rdeadline:  makePipeDeadline(),
		wdeadline:  makePipeDeadline(),
	}
	p2 := &conn{
		addr:       addr,
		r:          ch2,
		w:          ch1,
		localDone:  done2,
		remoteDone: done1,
		rdeadline:  makePipeDeadline(),
		wdeadline:  makePipeDeadline(),
	}
	return &pipe{p1, p2}
}

func (p *pipe) Close() error {
	e1 := p.p1.Close()
	e2 := p.p2.Close()
	if e1 != nil {
		return e1
	}
	if e2 != nil {
		return e2
	}
	return nil
}

// Listen announces on the given address.
func Listen(addr string) (*Listener, error) {
	mgr.mu.Lock()
	_, dup := mgr.db[addr]
	if dup {
		mgr.mu.Unlock()
		return nil, fmt.Errorf("inproc: address %q already in use", addr)
	}

	l := &Listener{
		addr: Addr(addr),
	}
	mgr.db[addr] = l
	mgr.cv.Broadcast()
	mgr.mu.Unlock()

	return l, nil
}

// Addr returns the listener's netword address.
func (l *Listener) Addr() net.Addr {
	return l.addr
}

// Close closes the listener.
// Any blocked Accept operations will be unblocked and return errors.
func (l *Listener) Close() error {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	if l.closed {
		return nil
	}
	var err error
	for i := range l.pipes {
		p := l.pipes[i]
		e := p.Close()
		if e != nil && err == nil {
			err = e
		}
	}
	l.closed = true
	delete(mgr.db, string(l.addr))
	return err
}

// Accept waits for and returns the next connection to the listener.
func (l *Listener) Accept() (net.Conn, error) {
	mgr.mu.Lock()
	p := newPipe(l.addr)
	l.pipes = append(l.pipes, p)
	closed := l.closed
	mgr.cv.Broadcast()
	mgr.mu.Unlock()

	if closed {
		return nil, ErrClosed
	}
	return p.p1, nil
}

// Dial connects to the given address.
func Dial(addr string) (net.Conn, error) {
	mgr.mu.Lock()

	for {
		var (
			l  *Listener
			ok bool
		)
		if l, ok = mgr.db[addr]; !ok || l == nil {
			mgr.mu.Unlock()
			return nil, ErrConnRefused
		}
		if n := len(l.pipes); n != 0 {
			p := l.pipes[n-1]
			l.pipes = l.pipes[:n-1]
			mgr.mu.Unlock()
			return p.p2, nil
		}
		mgr.cv.Wait()
	}
}

// Addr represents an in-process "network" end-point address.
type Addr string

// String implements net.Addr.String
func (a Addr) String() string {
	return strings.TrimPrefix(string(a), "inproc://")
}

// Network returns the name of the network.
func (a Addr) Network() string {
	return "inproc"
}

var (
	_ net.Addr     = (*Addr)(nil)
	_ net.Listener = (*Listener)(nil)
)
