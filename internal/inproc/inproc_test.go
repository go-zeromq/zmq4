// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package inproc

import (
	"bytes"
	"io"
	"math/rand"
	"testing"
)

func TestBasicIO(t *testing.T) {
	t.Skip()

	want := make([]byte, 1<<2)
	rand.New(rand.NewSource(0)).Read(want)

	pipe := newPipe(Addr("basic-io"))
	c1 := pipe.p1
	c2 := pipe.p2

	dataCh := make(chan []byte)
	go func() {
		rd := bytes.NewReader(want)
		if err := chunkedCopy(c1, rd); err != nil {
			t.Errorf("unexpected c1.Write error: %v", err)
		}
	}()

	go func() {
		wr := new(bytes.Buffer)
		if err := chunkedCopy(wr, c2); err != nil {
			t.Errorf("unexpected c2.Read error: %v", err)
		}
		dataCh <- wr.Bytes()
	}()

	if got := <-dataCh; !bytes.Equal(got, want) {
		//	t.Errorf("transmitted data differs")
		t.Errorf("transmitted data differs:\ngot= %q\nwnt= %q\n", got, want)
	}

	if err := c1.Close(); err != nil {
		t.Errorf("unexpected c1.Close error: %v", err)
	}
	if err := c2.Close(); err != nil {
		t.Errorf("unexpected c2.Close error: %v", err)
	}
}

// chunkedCopy copies from r to w in fixed-width chunks to avoid
// causing a Write that exceeds the maximum packet size for packet-based
// connections like "unixpacket".
// We assume that the maximum packet size is at least 1024.
func chunkedCopy(w io.Writer, r io.Reader) error {
	//	b := make([]byte, 1024)
	//	_, err := io.CopyBuffer(struct{ io.Writer }{w}, struct{ io.Reader }{r}, b)
	_, err := io.Copy(w, r)
	return err
}
