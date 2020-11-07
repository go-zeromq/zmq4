// Copyright 2020 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/go-zeromq/zmq4"
)

var (
	wg     sync.WaitGroup
	outMsg zmq4.Msg
	inMsg  zmq4.Msg
)

func requester(t *testing.T, ep string) {

	req := zmq4.NewReq(context.Background())
	defer req.Close()
	defer wg.Done()

	err := req.Dial(ep)
	if err != nil {
		t.Fatalf("could not dial: %v", err)
	}

	// Test message w/ 3 frames
	outMsg = zmq4.NewMsgFromString([]string{"ZERO", "Hello!", "World!"})
	err = req.Send(outMsg)
	if err != nil {
		t.Fatalf("failed to send: %v", err)
	}

	inMsg, err = req.Recv()
	if err != nil {
		t.Fatalf("failed to recv: %v", err)
	}
}

func responder(t *testing.T, ep string) {

	rep := zmq4.NewRep(context.Background())
	defer rep.Close()
	defer wg.Done()

	err := rep.Listen(ep)
	if err != nil {
		t.Fatalf("could not dial: %v", err)
	}

	//  Wait for next request from client
	msg, err := rep.Recv()
	if err != nil {
		t.Fatalf("could not recv request: %v", err)
	}

	//  Send reply back to client
	err = rep.Send(msg)
	if err != nil {
		t.Fatalf("could not send reply: %v", err)
	}
	time.Sleep(2 * time.Millisecond)
}

func TestIssue99(t *testing.T) {
	wg.Add(2)
	ep, err := EndPoint("tcp")
	if err != nil {
		t.Fatalf("could not find endpoint: %v", err)
	}

	go requester(t, ep)
	go responder(t, ep)
	wg.Wait()
	if len(outMsg.Frames) != len(inMsg.Frames) {
		t.Error("message length mismatch")
	}
}
