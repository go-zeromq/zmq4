// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package inproc

import (
	"bytes"
	"io"
	"math/rand"
	"reflect"
	"testing"

	"golang.org/x/sync/errgroup"
	"golang.org/x/xerrors"
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

func TestRW(t *testing.T) {
	const ep = "inproc://rw-srv"
	lst, err := Listen(ep)
	if err != nil {
		t.Fatalf("could not create server: %+v", err)
	}
	defer lst.Close()

	if addr := lst.Addr(); addr == nil {
		t.Fatalf("listener with nil address")
	}
	if got, want := lst.Addr().String(), ep[len("inproc://"):]; got != want {
		t.Fatalf("invalid listener address: got=%q, want=%q", got, want)
	}

	var grp errgroup.Group
	grp.Go(func() error {
		conn, err := lst.Accept()
		if err != nil {
			return xerrors.Errorf("could not accept connection: %w", err)
		}
		defer conn.Close()

		if addr := conn.LocalAddr(); addr == nil {
			t.Fatalf("accept-conn with nil address")
		}
		if got, want := conn.LocalAddr().String(), ep[len("inproc://"):]; got != want {
			t.Fatalf("invalid accept-con address: got=%q, want=%q", got, want)
		}
		if got, want := conn.RemoteAddr().String(), ep[len("inproc://"):]; got != want {
			t.Fatalf("invalid accept-con address: got=%q, want=%q", got, want)
		}

		raw := make([]byte, len("HELLO"))
		_, err = io.ReadFull(conn, raw)
		if err != nil {
			return xerrors.Errorf("could not read request: %w", err)
		}

		if got, want := raw, []byte("HELLO"); !reflect.DeepEqual(got, want) {
			return xerrors.Errorf("invalid request: got=%v, want=%v", got, want)
		}

		_, err = conn.Write([]byte("HELLO"))
		if err != nil {
			return xerrors.Errorf("could not write reply: %w", err)
		}

		raw = make([]byte, len("QUIT"))
		_, err = io.ReadFull(conn, raw)
		if err != nil {
			return xerrors.Errorf("could not read final request: %w", err)
		}

		if got, want := raw, []byte("QUIT"); !reflect.DeepEqual(got, want) {
			return xerrors.Errorf("invalid request: got=%v, want=%v", got, want)
		}

		return nil
	})

	grp.Go(func() error {
		conn, err := Dial("inproc://rw-srv")
		if err != nil {
			return xerrors.Errorf("could not dial server: %w", err)
		}
		defer conn.Close()

		if addr := conn.LocalAddr(); addr == nil {
			t.Fatalf("dial-conn with nil address")
		}
		if got, want := conn.LocalAddr().String(), ep[len("inproc://"):]; got != want {
			t.Fatalf("invalid dial-con address: got=%q, want=%q", got, want)
		}
		if got, want := conn.RemoteAddr().String(), ep[len("inproc://"):]; got != want {
			t.Fatalf("invalid dial-con address: got=%q, want=%q", got, want)
		}

		_, err = conn.Write([]byte("HELLO"))
		if err != nil {
			return xerrors.Errorf("could not send request: %w", err)
		}

		raw := make([]byte, len("HELLO"))
		_, err = io.ReadFull(conn, raw)
		if err != nil {
			return xerrors.Errorf("could not read reply: %w", err)
		}

		if got, want := raw, []byte("HELLO"); !reflect.DeepEqual(got, want) {
			return xerrors.Errorf("invalid reply: got=%v, want=%v", got, want)
		}

		_, err = conn.Write([]byte("QUIT"))
		if err != nil {
			return xerrors.Errorf("could not write final request: %w", err)
		}

		return nil
	})

	err = grp.Wait()
	if err != nil {
		t.Fatalf("error: %+v", err)
	}
}
