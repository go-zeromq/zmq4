// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"
)

func TestNullSecurity(t *testing.T) {
	sec := nullSecurity{}
	if got, want := sec.Type(), NullSecurity; got != want {
		t.Fatalf("got=%v, want=%v", got, want)
	}
}

func TestNullHandshakeReqRep(t *testing.T) {
	var (
		reqQuit = NewMsgString("QUIT")
		repQuit = NewMsgString("bye")
	)

	sec := nullSecurity{}
	ctx, timeout := context.WithTimeout(context.Background(), 10*time.Second)
	defer timeout()

	ep := "ipc://ipc-req-rep-null-sec"
	cleanUp(ep)

	req := NewReq(ctx, WithSecurity(sec), WithLogger(Devnull))
	defer req.Close()

	rep := NewRep(ctx, WithSecurity(sec), WithLogger(Devnull))
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
