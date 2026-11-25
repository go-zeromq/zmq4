// Copyright 2026 Nebius. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"context"
	"net"
	"runtime"
	"strings"
	"testing"
	"time"
)

// The full queue represents a consumer that exited before its peer stopped sending.
func TestQueuedReaderCancellation(t *testing.T) {
	for _, kind := range []string{"queue", "router", "publisher", "reply"} {
		t.Run(kind, func(t *testing.T) {
			for attempt := 0; attempt < 8; attempt++ {
				testQueuedReaderCancellation(t, kind)
			}
		})
	}
}

func testQueuedReaderCancellation(t *testing.T, kind string) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var queue chan Msg
	var listen func(context.Context, *Conn)
	var release func()
	switch kind {
	case "queue":
		q := newQReader(ctx)
		queue, listen = q.c, q.listen
	case "router":
		q := newRouterQReader(ctx)
		queue, listen = q.c, q.listen
	case "publisher":
		q := newPubQReader(ctx)
		queue, listen = q.c, q.listen
	case "reply":
		q := newRepReader(ctx, newRepState())
		listen = q.listen
		for len(q.msgCh) < cap(q.msgCh) {
			q.msgCh <- repMsg{msg: NewMsg(make([]byte, 64<<10))}
		}
		release = func() { <-q.msgCh }
	}
	if queue != nil {
		for len(queue) < cap(queue) {
			queue <- NewMsg(make([]byte, 64<<10))
		}
		release = func() { <-queue }
	}
	local, remote := net.Pipe()
	defer local.Close()
	defer remote.Close()
	receiver := &Conn{rw: local, sec: nullSecurity{}}
	sender := &Conn{rw: remote, sec: nullSecurity{}}
	done := make(chan struct{})
	go func() {
		defer close(done)
		listen(ctx, receiver)
	}()
	sent := make(chan error, 1)
	payload := make([]byte, 64<<10)
	payload[0] = 2 // PUB treats leading 0/1 as subscription updates.
	go func() { sent <- sender.SendMsg(NewMsg(payload)) }()
	select {
	case err := <-sent:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("reader did not receive the queued message")
	}
	// Wait for the enqueue to block before cancellation; otherwise the old reader
	// can observe cancellation before entering its unsafe default branch.
	deadline := time.Now().Add(time.Second)
	for !blockedQueueReader(kind) {
		if time.Now().After(deadline) {
			t.Fatal("reader did not block on the full queue")
		}
		runtime.Gosched()
	}
	cancel()
	_ = local.Close()
	select {
	case <-done:
	case <-time.After(time.Second):
		// Release the old implementation too, so a failing regression does not leak.
		release()
		<-done
		t.Fatal("reader stayed blocked on enqueue after cancellation and connection close")
	}
}

func blockedQueueReader(kind string) bool {
	name := map[string]string{"queue": "qreader", "router": "routerQReader", "publisher": "pubQReader", "reply": "repReader"}[kind]
	buf := make([]byte, 64<<10)
	var stacks string
	for {
		n := runtime.Stack(buf, true)
		if n < len(buf) {
			stacks = string(buf[:n])
			break
		}
		buf = make([]byte, 2*len(buf))
	}
	for _, stack := range strings.Split(stacks, "\n\n") {
		lines := strings.SplitN(stack, "\n", 3)
		if len(lines) >= 2 && strings.Contains(lines[1], "zmq4.(*"+name+").listen(") &&
			(strings.Contains(lines[0], "[select]") || strings.Contains(lines[0], "[chan send]")) {
			return true
		}
	}
	return false
}
