// Copyright 2020 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"context"
	"sync"
	"testing"
	"time"
)

var (
	wg     sync.WaitGroup
	outMsg Msg
	inMsg  Msg
)

func requester(t *testing.T) {

	req := NewReq(context.Background())
	defer req.Close()
	defer wg.Done()

	err := req.Dial("tcp://localhost:5559")
	if err != nil {
		t.Fatalf("could not dial: %v", err)
	}

	// Test message w/ 3 frames
	outMsg = NewMsgFromString([]string{"ZERO", "Hello!", "World!"})
	err = req.Send(outMsg)
	if err != nil {
		t.Fatalf("failed to send: %v", err)
	}

	inMsg, err = req.Recv()
	if err != nil {
		t.Fatalf("failed to recv: %v", err)
	}
}

func responder(t *testing.T) {

	//  Socket to talk to clients
	rep := NewRep(context.Background())
	defer rep.Close()
	defer wg.Done()

	err := rep.Listen("tcp://*:5559")
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
	go requester(t)
	go responder(t)
	wg.Wait()
	if len(outMsg.Frames) != len(inMsg.Frames) {
		t.Error("message length mismatch")
	}
}
