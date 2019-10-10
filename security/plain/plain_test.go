// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plain_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"net"
	"testing"
	"time"

	"github.com/go-zeromq/zmq4"
	"github.com/go-zeromq/zmq4/security/plain"
	"golang.org/x/sync/errgroup"
	"golang.org/x/xerrors"
)

var (
	reqQuit = zmq4.NewMsgString("QUIT")
	repQuit = zmq4.NewMsgString("bye")
)

func TestSecurity(t *testing.T) {
	sec := plain.Security("user", "secret")
	if got, want := sec.Type(), zmq4.PlainSecurity; got != want {
		t.Fatalf("got=%v, want=%v", got, want)
	}

	data := []byte("hello world")
	wenc := new(bytes.Buffer)
	if _, err := sec.Encrypt(wenc, data); err != nil {
		t.Fatalf("error encrypting data: %+v", err)
	}

	if !bytes.Equal(wenc.Bytes(), data) {
		t.Fatalf("error encrypted data.\ngot = %q\nwant= %q\n", wenc.Bytes(), data)
	}

	wdec := new(bytes.Buffer)
	if _, err := sec.Decrypt(wdec, wenc.Bytes()); err != nil {
		t.Fatalf("error decrypting data: %+v", err)
	}

	if !bytes.Equal(wdec.Bytes(), data) {
		t.Fatalf("error decrypted data.\ngot = %q\nwant= %q\n", wdec.Bytes(), data)
	}
}

func TestHandshakeReqRep(t *testing.T) {
	sec := plain.Security("user", "secret")
	if got, want := sec.Type(), zmq4.PlainSecurity; got != want {
		t.Fatalf("got=%v, want=%v", got, want)
	}

	ctx, timeout := context.WithTimeout(context.Background(), 10*time.Second)
	defer timeout()

	ep := must(EndPoint("tcp"))

	req := zmq4.NewReq(ctx, zmq4.WithSecurity(sec))
	defer req.Close()

	rep := zmq4.NewRep(ctx, zmq4.WithSecurity(sec))
	defer rep.Close()

	grp, ctx := errgroup.WithContext(ctx)
	grp.Go(func() error {
		err := rep.Listen(ep)
		if err != nil {
			return xerrors.Errorf("could not listen: %w", err)
		}

		msg, err := rep.Recv()
		if err != nil {
			return xerrors.Errorf("could not recv REQ message: %w", err)
		}
		if string(msg.Frames[0]) != "QUIT" {
			return xerrors.Errorf("received wrong REQ message: %#v", msg)
		}
		return nil
	})

	grp.Go(func() error {
		err := req.Dial(ep)
		if err != nil {
			return xerrors.Errorf("could not dial: %w", err)
		}

		err = req.Send(reqQuit)
		if err != nil {
			return xerrors.Errorf("could not send REQ message: %w", err)
		}
		return nil
	})

	if err := grp.Wait(); err != nil {
		t.Fatalf("error: %+v", err)
	}
}

func must(str string, err error) string {
	if err != nil {
		panic(err)
	}
	return str
}

func EndPoint(transport string) (string, error) {
	switch transport {
	case "tcp":
		addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
		if err != nil {
			return "", xerrors.Errorf("could not resolve TCP address: %w", err)
		}
		l, err := net.ListenTCP("tcp", addr)
		if err != nil {
			return "", xerrors.Errorf("could not listen to TCP addr=%q: %w", addr, err)
		}
		defer l.Close()
		return fmt.Sprintf("tcp://%s", l.Addr()), nil
	case "ipc":
		return "ipc://tmp-" + newUUID(), nil
	case "inproc":
		return "inproc://tmp-" + newUUID(), nil
	default:
		panic("invalid transport: [" + transport + "]")
	}
}

func newUUID() string {
	var uuid [16]byte
	if _, err := io.ReadFull(rand.Reader, uuid[:]); err != nil {
		log.Fatalf("cannot generate random data for UUID: %v", err)
	}
	uuid[8] = uuid[8]&^0xc0 | 0x80
	uuid[6] = uuid[6]&^0xf0 | 0x40
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:])
}
