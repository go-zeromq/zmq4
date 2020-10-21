// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package inproc

import (
	"io"
	"net"
	"sync"
	"time"
)

type conn struct {
	addr Addr
	r    <-chan []byte
	w    chan<- []byte

	once       sync.Once // Protects closing localDone
	localDone  chan struct{}
	remoteDone <-chan struct{}

	rdeadline pipeDeadline
	wdeadline pipeDeadline
}

func (c *conn) Write(data []byte) (int, error) {
	n, err := c.write(data)
	if err != nil && err != io.ErrClosedPipe {
		err = &net.OpError{Op: "write", Net: "pipe", Err: err}
	}
	return n, err
}

func (c *conn) write(data []byte) (int, error) {
	switch {
	case isClosedChan(c.localDone):
		return 0, io.ErrClosedPipe
	case isClosedChan(c.remoteDone):
		return 0, io.ErrClosedPipe
	case isClosedChan(c.wdeadline.wait()):
		return 0, timeoutError{}
	}

	var n int
	select {
	case c.w <- data:
		n = len(data)
		return n, nil
	case <-c.localDone:
		return n, io.ErrClosedPipe
	case <-c.remoteDone:
		return n, io.ErrClosedPipe
	case <-c.wdeadline.wait():
		return n, timeoutError{}
	}
}

func (c *conn) Read(data []byte) (int, error) {
	n, err := c.read(data)
	if err != nil && err != io.EOF && err != io.ErrClosedPipe {
		err = &net.OpError{Op: "read", Net: "pipe", Err: err}
	}
	return n, err
}

func (c *conn) read(data []byte) (int, error) {
	switch {
	case isClosedChan(c.localDone):
		return 0, io.ErrClosedPipe
	case isClosedChan(c.remoteDone):
		return 0, io.EOF
	case isClosedChan(c.rdeadline.wait()):
		return 0, timeoutError{}
	}

	select {
	case bw := <-c.r:
		nr := copy(data, bw)
		if len(data) < len(bw) {
			return nr, io.ErrShortBuffer
		}
		return nr, nil
	case <-c.rdeadline.wait():
		return 0, timeoutError{}
	}
}

func (c *conn) LocalAddr() net.Addr  { return c.addr }
func (c *conn) RemoteAddr() net.Addr { return c.addr }

func (c *conn) SetDeadline(t time.Time) error {
	if isClosedChan(c.localDone) || isClosedChan(c.remoteDone) {
		return io.ErrClosedPipe
	}
	c.rdeadline.set(t)
	c.wdeadline.set(t)
	return nil
}

func (c *conn) SetReadDeadline(t time.Time) error {
	if isClosedChan(c.localDone) || isClosedChan(c.remoteDone) {
		return io.ErrClosedPipe
	}
	c.rdeadline.set(t)
	return nil
}

func (c *conn) SetWriteDeadline(t time.Time) error {
	if isClosedChan(c.localDone) || isClosedChan(c.remoteDone) {
		return io.ErrClosedPipe
	}
	c.wdeadline.set(t)
	return nil
}

func (c *conn) Close() error {
	c.once.Do(func() {
		close(c.localDone)
	})
	return nil
}

// pipeDeadline is an abstraction for handling timeouts.
type pipeDeadline struct {
	mu     sync.Mutex // Guards timer and cancel
	timer  *time.Timer
	cancel chan struct{} // Must be non-nil
}

func makePipeDeadline() pipeDeadline {
	return pipeDeadline{cancel: make(chan struct{})}
}

// set sets the point in time when the deadline will time out.
// A timeout event is signaled by closing the channel returned by waiter.
// Once a timeout has occurred, the deadline can be refreshed by specifying a
// t value in the future.
//
// A zero value for t prevents timeout.
func (d *pipeDeadline) set(t time.Time) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.timer != nil && !d.timer.Stop() {
		<-d.cancel // Wait for the timer callback to finish and close cancel
	}
	d.timer = nil

	// Time is zero, then there is no deadline.
	closed := isClosedChan(d.cancel)
	if t.IsZero() {
		if closed {
			d.cancel = make(chan struct{})
		}
		return
	}

	// Time in the future, setup a timer to cancel in the future.
	if dur := time.Until(t); dur > 0 {
		if closed {
			d.cancel = make(chan struct{})
		}
		d.timer = time.AfterFunc(dur, func() {
			close(d.cancel)
		})
		return
	}

	// Time in the past, so close immediately.
	if !closed {
		close(d.cancel)
	}
}

// wait returns a channel that is closed when the deadline is exceeded.
func (d *pipeDeadline) wait() chan struct{} {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.cancel
}

func isClosedChan(c <-chan struct{}) bool {
	select {
	case <-c:
		return true
	default:
		return false
	}
}

type timeoutError struct{}

func (timeoutError) Error() string   { return "deadline exceeded" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }
