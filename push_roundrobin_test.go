// Copyright 2026 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestPushRoundRobin(t *testing.T) {
	for _, multipart := range []bool{false, true} {
		for _, peers := range []int{1, 2, 3} {
			for _, bindPush := range []bool{false, true} {
				t.Run(fmt.Sprintf("multipart=%v/peers=%d/bindPush=%v", multipart, peers, bindPush), func(t *testing.T) {
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					push := NewPush(ctx).(*pushSocket)
					defer push.Close()
					if bindPush {
						if err := push.Listen("tcp://127.0.0.1:0"); err != nil {
							t.Fatal(err)
						}
					}
					pulls := make([]Socket, peers)
					for i := range pulls {
						pull := NewPull(ctx)
						defer pull.Close()
						pulls[i] = pull
						if bindPush {
							if err := pull.Dial("tcp://" + push.Addr().String()); err != nil {
								t.Fatal(err)
							}
						} else {
							if err := pull.Listen("tcp://127.0.0.1:0"); err != nil {
								t.Fatal(err)
							}
							if err := push.Dial("tcp://" + pull.Addr().String()); err != nil {
								t.Fatal(err)
							}
						}
					}
					// Dial's handshake can finish just before the listener registers
					// the new peer. Wait until every worker is available for routing.
					for {
						push.sck.mu.RLock()
						connected := len(push.sck.conns)
						push.sck.mu.RUnlock()
						if connected == peers {
							break
						}
						select {
						case <-ctx.Done():
							t.Fatal(ctx.Err())
						case <-time.After(time.Millisecond):
						}
					}
					send := push.Send
					if multipart {
						send = push.SendMulti
					}
					messages := make([]Msg, 3*peers)
					for i := range messages {
						messages[i] = NewMsgFrom([]byte(fmt.Sprintf("task-%d", i)), []byte("body"))
						if err := send(messages[i]); err != nil {
							t.Fatal(err)
						}
					}
					for peer, pull := range pulls {
						for i := peer; i < len(messages); i += peers {
							got, err := pull.Recv()
							if err != nil {
								t.Fatal(err)
							}
							if !reflect.DeepEqual(got.Frames, messages[i].Frames) {
								t.Fatalf("peer %d: frames = %q, want task %d: %q", peer, got.Frames, i, messages[i].Frames)
							}
						}
					}
				})
			}
		}
	}
}
