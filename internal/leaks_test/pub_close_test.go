// Copyright 2026 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package leaks_test

import (
	"context"
	"testing"

	"github.com/go-zeromq/zmq4"
	"go.uber.org/goleak"
)

func TestPubCloseReleasesWriters(t *testing.T) {
	for _, tc := range []struct {
		name      string
		newSocket func(context.Context, ...zmq4.Option) zmq4.Socket
	}{
		{"PUB", zmq4.NewPub},
		{"XPUB", zmq4.NewXPub},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer goleak.VerifyNone(t)

			sub := zmq4.NewSub(context.Background())
			defer sub.Close()
			if err := sub.Listen("tcp://127.0.0.1:0"); err != nil {
				t.Fatal(err)
			}
			endpoint := "tcp://" + sub.Addr().String()

			// The subscriber stays open while publishers are repeatedly replaced.
			// Each successful Dial starts a writer waiting for its first message.
			for i := 0; i < 5; i++ {
				pub := tc.newSocket(context.Background())
				if err := pub.Dial(endpoint); err != nil {
					pub.Close()
					t.Fatal(err)
				}
				if err := pub.Close(); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
