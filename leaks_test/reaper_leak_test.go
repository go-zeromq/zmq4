// Copyright 2024 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package leaks_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/go-zeromq/zmq4"
	"go.uber.org/goleak"
)

// TestReaper does multiple rapid Dial/Close to check that connection reaper goroutines are not leaking.
// In is in own package as goleak detects also threads from values created during init().
func TestReaperLeak1(t *testing.T) {
	defer goleak.VerifyNone(t)

	mu := &sync.Mutex{}
	errs := []error{}

	ctx, cancel := context.WithCancel(context.Background())
	rep := zmq4.NewRep(ctx)
	ep := "ipc://@test.rep.socket"
	err := rep.Listen(ep)
	if err != nil {
		t.Fatal(err)
	}

	maxClients := 100
	maxMsgs := 100
	wgClients := &sync.WaitGroup{}
	wgServer := &sync.WaitGroup{}
	client := func() {
		defer wgClients.Done()
		for n := 0; n < maxMsgs; n++ {
			func() {
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
				defer cancel()
				req := zmq4.NewReq(ctx)
				err := req.Dial(ep)
				if err != nil {
					mu.Lock()
					defer mu.Unlock()
					errs = append(errs, err)
					return
				}

				err = req.Close()
				if err != nil {
					mu.Lock()
					defer mu.Unlock()
					errs = append(errs, err)
				}
			}()
		}
	}
	server := func() {
		defer wgServer.Done()
		pong := zmq4.NewMsgString("pong")
		for {
			msg, err := rep.Recv()
			if errors.Is(err, context.Canceled) {
				break
			}
			if err != nil {
				break
			}
			if string(msg.Frames[0]) != "ping" {
				mu.Lock()
				defer mu.Unlock()
				errs = append(errs, errors.New("unexpected message"))
				return
			}
			err = rep.Send(pong)
			if err != nil {
				mu.Lock()
				defer mu.Unlock()
				errs = append(errs, err)
			}
		}
	}

	wgServer.Add(1)
	go server()
	wgClients.Add(maxClients)
	for n := 0; n < maxClients; n++ {
		go client()
	}
	wgClients.Wait()
	cancel()
	wgServer.Wait()
	rep.Close()
	for _, err := range errs {
		t.Fatal(err)
	}
}
