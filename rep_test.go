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

func TestIssue99(t *testing.T) {
	var (
		wg     sync.WaitGroup
		outMsg zmq4.Msg
		inMsg  zmq4.Msg
	)

	ep, err := EndPoint("tcp")
	if err != nil {
		t.Errorf("could not find endpoint: %v", err)
	}

	requester := func() {

		req := zmq4.NewReq(context.Background())
		defer req.Close()
		defer wg.Done()

		err := req.Dial(ep)
		if err != nil {
			t.Errorf("could not dial: %v", err)
		}

		// Test message w/ 3 frames
		outMsg = zmq4.NewMsgFromString([]string{"ZERO", "Hello!", "World!"})
		err = req.Send(outMsg)
		if err != nil {
			t.Errorf("failed to send: %v", err)
		}

		inMsg, err = req.Recv()
		if err != nil {
			t.Errorf("failed to recv: %v", err)
		}
	}

	responder := func() {

		rep := zmq4.NewRep(context.Background())
		defer rep.Close()
		defer wg.Done()

		err := rep.Listen(ep)
		if err != nil {
			t.Errorf("could not dial: %+v", err)
			return
		}

		//  Wait for next request from client
		msg, err := rep.Recv()
		if err != nil {
			t.Errorf("could not recv request: %+v", err)
			return
		}

		//  Send reply back to client
		err = rep.Send(msg)
		if err != nil {
			t.Errorf("could not send reply: %+v", err)
			return
		}
		time.Sleep(2 * time.Millisecond)
	}

	wg.Add(2)

	go requester()
	go responder()

	wg.Wait()

	if want, got := len(outMsg.Frames), len(inMsg.Frames); want != got {
		t.Fatalf("message length mismatch: got=%d, want=%d", got, want)
	}
}
