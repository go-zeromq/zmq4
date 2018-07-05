// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

// PubSub envelope publisher
package main

import (
	"context"
	"log"
	"time"

	"github.com/go-zeromq/zmq4"
)

func main() {
	log.SetPrefix("psenvpub: ")

	// prepare the publisher
	pub := zmq4.NewPub(context.Background())
	defer pub.Close()

	err := pub.Listen("tcp://*:5563")
	if err != nil {
		log.Fatalf("could not listen: %v", err)
	}

	msgA := zmq4.NewMsgFrom(
		[]byte("A"),
		[]byte("We don't want to see this"),
	)
	msgB := zmq4.NewMsgFrom(
		[]byte("B"),
		[]byte("We would like to see this"),
	)
	for {
		//  Write two messages, each with an envelope and content
		err = pub.Send(msgA)
		if err != nil {
			log.Fatal(err)
		}
		err = pub.Send(msgB)
		if err != nil {
			log.Fatal(err)
		}
		time.Sleep(time.Second)
	}
}
