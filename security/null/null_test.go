// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package null_test

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-zeromq/zmq4"
	"github.com/go-zeromq/zmq4/security/null"
	"golang.org/x/sync/errgroup"
)

func TestSecurity(t *testing.T) {
	sec := null.Security()
	if got, want := sec.Type(), zmq4.NullSecurity; got != want {
		t.Fatalf("got=%v, want=%v", got, want)
	}
}

func TestHandshakeReqRep(t *testing.T) {
	var (
		reqQuit = zmq4.NewMsgString("QUIT")
		repQuit = zmq4.NewMsgString("bye")
	)

	sec := null.Security()
	ctx, timeout := context.WithTimeout(context.Background(), 10*time.Second)
	defer timeout()

	ep := "ipc://ipc-req-rep-null-sec"
	cleanUp(ep)

	req := zmq4.NewReq(ctx, zmq4.WithSecurity(sec))
	defer req.Close()

	rep := zmq4.NewRep(ctx, zmq4.WithSecurity(sec))
	defer rep.Close()

	grp, _ := errgroup.WithContext(ctx)
	grp.Go(func() error {
		err := rep.Listen(ep)
		if err != nil {
			return fmt.Errorf("could not listen: %w", err)
		}

		msg, err := rep.Recv()
		if err != nil {
			return fmt.Errorf("could not recv REQ message: %w", err)
		}

		if !reflect.DeepEqual(msg, reqQuit) {
			return fmt.Errorf("got = %v, want = %v", msg, repQuit)
		}

		err = rep.Send(repQuit)
		if err != nil {
			return fmt.Errorf("could not send REP message: %w", err)
		}

		return nil
	})

	grp.Go(func() error {
		err := req.Dial(ep)
		if err != nil {
			return fmt.Errorf("could not dial: %w", err)
		}

		err = req.Send(reqQuit)
		if err != nil {
			return fmt.Errorf("could not send REQ message: %w", err)
		}
		return nil
	})

	if err := grp.Wait(); err != nil {
		t.Fatalf("error: %+v", err)
	}
}

func cleanUp(ep string) {
	if strings.HasPrefix(ep, "ipc://") {
		os.Remove(ep[len("ipc://"):])
	}
}
