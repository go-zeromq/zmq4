// Copyright 2023 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package zmq4

import (
	"context"
	"testing"
	"time"
)

func TestPushTimeout(t *testing.T) {
	ep := "ipc://@push_timeout_test"
	push := NewPush(context.Background(), WithTimeout(1*time.Second))
	defer push.Close()
	if err := push.Listen(ep); err != nil {
		t.FailNow()
	}

	pull := NewPull(context.Background())
	defer pull.Close()
	if err := pull.Dial(ep); err != nil {
		t.FailNow()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			// The ctx limits overall time of execution
			// If it gets canceled, that meain tests failed
			// as write to socket did not genereate timeout error
			t.Fatalf("test failed before being able to generate timeout error: %+v", ctx.Err())
		default:
		}

		err := push.Send(NewMsgString("test string"))
		if err == nil {
			continue
		}
		if err != context.DeadlineExceeded {
			t.Fatalf("expected a context.DeadlineExceeded error, got=%+v", err)
		}
		break
	}

}
