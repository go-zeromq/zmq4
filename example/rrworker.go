// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

// Request-reply worker.
//
// Connects REP socket to tcp://*:5559
// Expects "Hello" from client, replies with "World"
package main

import (
	"context"
	"log"
	"time"

	"github.com/go-zeromq/zmq4"
)

func main() {
	log.SetPrefix("rrworker: ")

	//  Socket to talk to clients
	rep := zmq4.NewRep(context.Background())
	defer rep.Close()

	err := rep.Listen("tcp://*:5559")
	if err != nil {
		log.Fatalf("could not dial: %v", err)
	}

	for {
		//  Wait for next request from client
		msg, err := rep.Recv()
		if err != nil {
			log.Fatalf("could not recv request: %v", err)
		}

		log.Printf("received request: [%s]\n", msg.Frames[0])

		//  Do some 'work'
		time.Sleep(time.Second)

		//  Send reply back to client
		err = rep.Send(zmq4.NewMsgString("World"))
		if err != nil {
			log.Fatalf("could not send reply: %v", err)
		}
	}
}
