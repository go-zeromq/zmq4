// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

// Router/Dealer example.
package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/go-zeromq/zmq4"
)

const (
	NWORKERS = 10
	endpoint = "tcp://localhost:5671"
)

var (
	Fired      = []byte("Fired!")
	WorkHarder = []byte("Work Harder!")

	ready = zmq4.NewMsgFrom([]byte(""), []byte("ready"))
)

func main() {
	rand.Seed(1234)
	bkg := context.Background()
	router := zmq4.NewCRouter(bkg, zmq4.CWithID(zmq4.SocketIdentity("router")))

	err := router.Listen("tcp://*:5671")
	if err != nil {
		log.Fatalf("could not listen %q: %v", endpoint, err)
	}
	defer router.Close()

	var wg sync.WaitGroup
	wg.Add(NWORKERS)
	for i := 0; i < NWORKERS; i++ {
		go worker(i, &wg)
	}

	nfired := 0
	for {
		msg, err := router.Recv()
		if err != nil {
			log.Fatalf("router failed to recv message: %v", err)
		}

		id := msg.Frames[0]
		fire := rand.Float64() * 100
		switch {
		case fire < 30:
			msg = zmq4.NewMsgFrom(id, []byte(""), Fired)
			nfired++
		default:
			msg = zmq4.NewMsgFrom(id, []byte(""), WorkHarder)
		}
		err = router.Send(msg)
		if err != nil {
			log.Fatalf("router failed to send message to %q: %v", id, err)
		}
		if nfired == NWORKERS {
			break
		}
	}
	wg.Wait()
	log.Printf("fired everybody.")
}

func worker(i int, wg *sync.WaitGroup) {
	id := zmq4.SocketIdentity(fmt.Sprintf("dealer-%d", i))
	dealer := zmq4.NewDealer(context.Background(), zmq4.WithID(id))
	defer dealer.Close()
	defer wg.Done()

	err := dealer.Dial(endpoint)
	if err != nil {
		log.Fatalf("dealer %d failed to dial: %v", i, err)
	}

	total := 0
dloop:
	for {
		// ready to work
		err = dealer.Send(ready)
		if err != nil {
			log.Fatalf("dealer %d failed to send ready message: %v", i, err)
		}

		// get workload from broker
		msg, err := dealer.Recv()
		if err != nil {
			log.Fatalf("dealer %d failed to recv message: %v", i, err)
		}
		work := msg.Frames[1]
		if bytes.Equal(work, Fired) {
			break dloop
		}

		// do some random work
		time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)
		total++
	}

	log.Printf("dealer %d completed %d tasks", i, total)
}
