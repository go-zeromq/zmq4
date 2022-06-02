// Copyright 2022 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build ignore
// +build ignore

// Hello client
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	zmq "github.com/go-zeromq/zmq4"
)

func main() {
	if err := hwclient(); err != nil {
		log.Fatalf("hwclient: %v", err)
	}
}

func hwclient() error {
	ctx := context.Background()
	socket := zmq.NewReq(ctx, zmq.WithDialerRetry(time.Second))
	defer socket.Close()

	fmt.Printf("Connecting to hello world server...")
	if err := socket.Dial("tcp://localhost:5555"); err != nil {
		return fmt.Errorf("dialing: %w", err)
	}

	for i := 0; i < 10; i++ {
		// Send hello.
		m := zmq.NewMsgString("hello")
		fmt.Println("sending ", m)
		if err := socket.Send(m); err != nil {
			return fmt.Errorf("sending: %w", err)
		}

		// Wait for reply.
		r, err := socket.Recv()
		if err != nil {
			return fmt.Errorf("receiving: %w", err)
		}
		fmt.Println("received ", r.String())
	}
	return nil
}
