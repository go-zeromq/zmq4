// Copyright 2026 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestQReaderCancellationWithFullQueue(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	reader, writer := net.Pipe()
	defer writer.Close()
	conn := &Conn{rw: reader, sec: nullSecurity{}}
	q := newQReader(ctx)
	// Fill the queue without a consumer, as happens while replay handoff is blocked.
	for i := 0; i < cap(q.c); i++ {
		q.c <- NewMsg([]byte("queued payload"))
	}
	q.rs = append(q.rs, conn)
	done := make(chan struct{})
	go func() {
		q.listen(ctx, conn)
		close(done)
	}()
	read := make(chan error, 1)
	go func() {
		// net.Pipe writes complete only after Conn.read consumes the frame.
		_, err := writer.Write([]byte{0, 1, 'a'})
		read <- err
	}()
	select {
	case err := <-read:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("reader did not consume the frame")
	}
	// Let listen enter the queue send before cancelling it.
	time.Sleep(50 * time.Millisecond)
	cancel()
	if err := q.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("cancelled reader retained the full queue and blocked message")
	}
	q.mu.RLock()
	defer q.mu.RUnlock()
	if len(q.rs) != 0 {
		t.Fatal("cancelled reader did not remove its connection")
	}
}
