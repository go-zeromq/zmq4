// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

// Request-reply client.
//
// Connects REQ socket to tcp://localhost:5559
// Sends "Hello" to server, expects "World" back
package main

import (
	"context"
	"log"

	"github.com/go-zeromq/zmq4"
)

func main() {
	log.SetPrefix("rrclient: ")

	req := zmq4.NewReq(context.Background())
	defer req.Close()

	err := req.Dial("tcp://localhost:5559")
	if err != nil {
		log.Fatalf("could not dial: %v", err)
	}

	for i := 0; i < 10; i++ {
		err := req.Send(zmq4.NewMsgString("Hello"))
		if err != nil {
			log.Fatalf("could not send greeting: %v", err)
		}

		msg, err := req.Recv()
		if err != nil {
			log.Fatalf("could not recv greeting: %v", err)
		}
		log.Printf("received reply %d [%s]\n", i, msg.Frames[0])
	}
}
