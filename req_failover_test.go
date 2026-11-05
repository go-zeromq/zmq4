// Copyright 2026 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestReqFailoverWrap(t *testing.T) {
	for _, multipart := range []bool{false, true} {
		for _, peers := range []int{2, 3} {
			t.Run(fmt.Sprintf("multipart=%v/peers=%d", multipart, peers), func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				req := NewReq(ctx).(*reqSocket)
				defer req.Close()

				for i := 0; i < peers; i++ {
					rep := NewRep(ctx)
					defer rep.Close()
					if err := rep.Listen("tcp://127.0.0.1:0"); err != nil {
						t.Fatal(err)
					}
					go func() {
						for {
							msg, err := rep.Recv()
							if err != nil {
								return
							}
							if err := rep.Send(msg); err != nil {
								return
							}
						}
					}()
					if err := req.Dial("tcp://" + rep.Addr().String()); err != nil {
						t.Fatal(err)
					}
				}

				send := req.Send
				if multipart {
					send = req.SendMulti
				}
				msg := NewMsgFrom([]byte("request"), []byte("body"))
				exchange := func() {
					t.Helper()
					if err := send(msg); err != nil {
						t.Fatal(err)
					}
					got, err := req.Recv()
					if err != nil {
						t.Fatal(err)
					}
					if !reflect.DeepEqual(got.Frames, msg.Frames) {
						t.Fatalf("reply frames = %q, want %q", got.Frames, msg.Frames)
					}
				}

				// Advance the round-robin cursor to peer 1, then reproduce
				// failed peers which have not yet been removed by the reaper.
				exchange()
				w := req.sck.w.(*reqWriter)
				// Hold removal until the sends finish, as can happen while the
				// reaper waits for another socket operation to release this lock.
				req.sck.mu.Lock()
				defer req.sck.mu.Unlock()
				for _, conn := range w.conns[1:] {
					if err := conn.Close(); err != nil {
						t.Fatal(err)
					}
				}
				exchange()
				// A successful fallback must retain the correct reply peer and
				// leave the cursor usable on subsequent requests as well.
				exchange()

				w.conns[0].SetClosed()
				if err := send(msg); !errors.Is(err, ErrClosedConn) {
					t.Fatalf("all peers closed: got %v, want ErrClosedConn", err)
				}
			})
		}
	}
}
