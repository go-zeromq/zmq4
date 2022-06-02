// Copyright 2022 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build ignore
// +build ignore

// Hello server
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	zmq "github.com/go-zeromq/zmq4"
)

func main() {
	if err := hwserver(); err != nil {
		log.Fatalf("hwserver: %w", err)
	}
}

func hwserver() error {
	ctx := context.Background()
	// Socket to talk to clients
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
		fmt.Println("Received ", msg)

		// Do some 'work'
		time.Sleep(time.Second)

		reply := fmt.Sprintf("World")
		if err := socket.Send(zmq.NewMsgString(reply)); err != nil {
			return fmt.Errorf("sending reply: %w", err)
		}
	}
}
