// Copyright 2026 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/go-zeromq/zmq4"
	"golang.org/x/sync/errgroup"
)

// A burst larger than any one write a subscriber is sent in reaches it whole
// and in order, with the messages of a topic it did not subscribe to, sent in
// between, filtered out.
func TestPubSubBurstArrivesInOrder(t *testing.T) {
	const msgCount = 20000

	pub := zmq4.NewPub(bkg)
	sub := zmq4.NewSub(bkg)
	defer pub.Close()
	defer sub.Close()

	// High enough that nothing is dropped, so every message must arrive.
	if err := pub.SetOption(zmq4.OptionHWM, 2*msgCount); err != nil {
		t.Fatalf("could not set HWM: %+v", err)
	}

	ep := must(EndPoint("tcp"))
	cleanUp(ep)

	ctx, timeout := context.WithTimeout(context.Background(), 20*time.Second)
	defer timeout()

	grp, _ := errgroup.WithContext(ctx)
	pss := newPubSubSync(1)

	grp.Go(func() error {
		if err := pub.Listen(ep); err != nil {
			return fmt.Errorf("could not listen on end point: %+v", err)
		}
		pss.WaitForSubscriptions()
		time.Sleep(time.Second) // let the subscription reach the publisher

		for i := 0; i < msgCount; i++ {
			body := []byte(strconv.Itoa(i))
			if err := pub.Send(zmq4.NewMsgFrom([]byte("other"), body)); err != nil {
				return fmt.Errorf("could not send message %d: %w", i, err)
			}
			if err := pub.Send(zmq4.NewMsgFrom([]byte("msg"), body)); err != nil {
				return fmt.Errorf("could not send message %d: %w", i, err)
			}
		}
		return nil
	})

	grp.Go(func() error {
		if err := sub.Dial(ep); err != nil {
			return fmt.Errorf("could not dial end point: %+v", err)
		}
		pss.DialComplete()
		pss.WaitForDialers()

		if err := sub.SetOption(zmq4.OptionSubscribe, "msg"); err != nil {
			return fmt.Errorf("could not subscribe: %w", err)
		}
		pss.SubscriptionComplete()
		pss.WaitForSubscriptions()

		for i := 0; i < msgCount; i++ {
			msg, err := sub.Recv()
			if err != nil {
				return fmt.Errorf("could not recv message %d: %w", i, err)
			}
			if got := string(msg.Frames[0]); got != "msg" {
				return fmt.Errorf("message %d is on topic %q", i, got)
			}
			if got, want := string(msg.Frames[1]), strconv.Itoa(i); got != want {
				return fmt.Errorf("message %d is %q, want %q", i, got, want)
			}
		}
		return nil
	})

	if err := grp.Wait(); err != nil {
		t.Fatalf("error: %+v", err)
	}
}
