// Copyright 2022 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

// Hello server
package main

import (
	"context"
	"fmt"
	zmq "github.com/go-zeromq/zmq4"
	"log"
	"time"
)

func main() {
	if err := helloserver(); err != nil {
		log.Fatalf("helloserver: %w", err)
	}
}

func helloserver() error {
	ctx := context.Background()
	socket := zmq.NewRep(ctx)
	defer socket.Close()
	if err := socket.Listen("tcp://*:5555"); err != nil {
		return fmt.Errorf("listening: %w", err)
	}

	for {
		msg, err := socket.Recv()
		if err != nil {
			return fmt.Errorf("receiving: %w", err)
		}
		println("Received ", msg.String())

		time.Sleep(time.Second)

		reply := fmt.Sprintf("World")
		if err := socket.Send(zmq.NewMsgString(reply)); err != nil {
			return fmt.Errorf("sending reply: %w", err)
		}
	}
}

