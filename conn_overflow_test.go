// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"testing"
)

// TestConnReadOversizeFrameGuarded locks the maxFrameSize guard added to
// Conn.read. A truncated/corrupt frame header can decode to a uint64 body size
// that is above runtime.maxAlloc (~2^48) but below maxInt64 (~9.2e18). The
// upstream guard only rejected sizes > maxInt64, so such a size passed the guard
// and make([]byte, size) panicked with "makeslice: len out of range", crashing
// the whole process (the reader runs in a goroutine with no recover). The fix
// caps frames at maxFrameSize (1<<30, well below maxAlloc so make cannot panic)
// and returns errOverflow instead. This test feeds a frame whose body size is
// 1<<49 (in the panic window) and asserts errOverflow with no panic.
func TestConnReadOversizeFrameGuarded(t *testing.T) {
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	conn := &Conn{rw: c2, sec: nullSecurity{}}

	// Long-frame wire layout: [flag byte][8-byte big-endian size].
	// flag = isLongBitFlag (0x2); hasMore/command bits clear -> single data frame.
	var buf bytes.Buffer
	buf.WriteByte(byte(isLongBitFlag))
	var sz [8]byte
	binary.BigEndian.PutUint64(sz[:], 1<<49) // > maxAlloc (~2^48), < maxInt64
	buf.Write(sz[:])

	go func() { _, _ = io.Copy(c1, &buf) }()

	msg := conn.read()
	if !errors.Is(msg.err, errOverflow) {
		t.Fatalf("expected errOverflow for oversize frame, got err=%v (the maxFrameSize guard is missing — make() would panic the process)", msg.err)
	}
}

// TestConnReadLegitFrameStillAccepted ensures the tightened guard does not
// reject frames within maxFrameSize.
func TestConnReadLegitFrameStillAccepted(t *testing.T) {
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	conn := &Conn{rw: c2, sec: nullSecurity{}}

	body := []byte("hello")
	var buf bytes.Buffer
	buf.WriteByte(byte(isLongBitFlag))
	var sz [8]byte
	binary.BigEndian.PutUint64(sz[:], uint64(len(body)))
	buf.Write(sz[:])
	buf.Write(body)

	go func() { _, _ = io.Copy(c1, &buf) }()

	msg := conn.read()
	if msg.err != nil {
		t.Fatalf("unexpected err reading legit frame: %v", msg.err)
	}
	if len(msg.Frames) != 1 || string(msg.Frames[0]) != "hello" {
		t.Fatalf("unexpected frames: %#v", msg.Frames)
	}
}
