// Copyright 2026 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"bufio"
	"bytes"
	"net"
	"reflect"
	"testing"
)

// recordConn is a net.Conn that records what is written to it.
type recordConn struct {
	net.Conn
	buf bytes.Buffer
}

func (c *recordConn) Write(p []byte) (int, error) { return c.buf.Write(p) }

func batchMsgs() []Msg {
	return []Msg{
		NewMsg([]byte("one")),
		NewMsgFrom([]byte("topic"), bytes.Repeat([]byte("x"), 300)),
		NewMsgFrom([]byte("topic"), []byte{}, []byte("three")),
	}
}

// sendMsgs writes the same bytes as SendMsg would for each message in turn,
// short and long frames alike.
func TestSendMsgsWritesWhatSendMsgWrites(t *testing.T) {
	one, batch := &recordConn{}, &recordConn{}
	single := &Conn{rw: one, sec: nullSecurity{}}
	batched := &Conn{rw: batch, sec: nullSecurity{}}

	for _, msg := range batchMsgs() {
		if err := single.SendMsg(msg); err != nil {
			t.Fatalf("SendMsg: %+v", err)
		}
	}
	if err := batched.sendMsgs(batchMsgs()); err != nil {
		t.Fatalf("sendMsgs: %+v", err)
	}
	if !bytes.Equal(one.buf.Bytes(), batch.buf.Bytes()) {
		t.Fatalf("sendMsgs wrote\n%q\nSendMsg wrote\n%q", batch.buf.Bytes(), one.buf.Bytes())
	}
}

// Messages written together are read back one at a time, through the
// connection's read buffer, as they were sent.
func TestSendMsgsReadBack(t *testing.T) {
	w := &recordConn{}
	if err := (&Conn{rw: w, sec: nullSecurity{}}).sendMsgs(batchMsgs()); err != nil {
		t.Fatalf("sendMsgs: %+v", err)
	}
	r := &Conn{r: bufio.NewReaderSize(&w.buf, readBufferSize), sec: nullSecurity{}}
	for i, want := range batchMsgs() {
		got := r.read()
		if got.err != nil {
			t.Fatalf("message %d: %+v", i, got.err)
		}
		if !reflect.DeepEqual(got.Frames, want.Frames) {
			t.Fatalf("message %d: got %q, want %q", i, got.Frames, want.Frames)
		}
	}
}
