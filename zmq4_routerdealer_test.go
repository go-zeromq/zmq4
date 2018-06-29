// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4_test

import (
	"bytes"
	"context"
	"math/rand"
	"reflect"
	"testing"
	"time"

	"github.com/go-zeromq/zmq4"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

var (
	routerdealers = []testCaseRouterDealer{
		{
			name:     "tcp-router-dealer",
			endpoint: must(EndPoint("tcp")),
			router:   zmq4.NewRouter(bkg, zmq4.WithID(zmq4.SocketIdentity("router"))),
			dealer0:  zmq4.NewDealer(bkg, zmq4.WithID(zmq4.SocketIdentity("dealer-0"))),
			dealer1:  zmq4.NewDealer(bkg, zmq4.WithID(zmq4.SocketIdentity("dealer-1"))),
			dealer2:  zmq4.NewDealer(bkg, zmq4.WithID(zmq4.SocketIdentity("dealer-2"))),
		},
		{
			name:     "ipc-router-dealer",
			endpoint: "ipc://ipc-router-dealer",
			router:   zmq4.NewRouter(bkg, zmq4.WithID(zmq4.SocketIdentity("router"))),
			dealer0:  zmq4.NewDealer(bkg, zmq4.WithID(zmq4.SocketIdentity("dealer-0"))),
			dealer1:  zmq4.NewDealer(bkg, zmq4.WithID(zmq4.SocketIdentity("dealer-1"))),
			dealer2:  zmq4.NewDealer(bkg, zmq4.WithID(zmq4.SocketIdentity("dealer-2"))),
		},
		{
			name:     "inproc-router-dealer",
			endpoint: "inproc://inproc-router-dealer",
			router:   zmq4.NewRouter(bkg, zmq4.WithID(zmq4.SocketIdentity("router"))),
			dealer0:  zmq4.NewDealer(bkg, zmq4.WithID(zmq4.SocketIdentity("dealer-0"))),
			dealer1:  zmq4.NewDealer(bkg, zmq4.WithID(zmq4.SocketIdentity("dealer-1"))),
			dealer2:  zmq4.NewDealer(bkg, zmq4.WithID(zmq4.SocketIdentity("dealer-2"))),
		},
	}
)

type testCaseRouterDealer struct {
	name     string
	skip     bool
	endpoint string
	router   zmq4.Socket
	dealer0  zmq4.Socket
	dealer1  zmq4.Socket
	dealer2  zmq4.Socket
}

func TestRouterDealer(t *testing.T) {
	var (
		Fired      = []byte("Fired!")
		WorkHarder = []byte("Work Harder!")

		ready = zmq4.NewMsgFrom([]byte(""), []byte("ready"))
	)

	for i := range routerdealers {
		tc := routerdealers[i]
		t.Run(tc.name, func(t *testing.T) {
			defer tc.router.Close()
			defer tc.dealer0.Close()
			defer tc.dealer1.Close()
			defer tc.dealer2.Close()

			if tc.skip {
				t.Skipf(tc.name)
			}
			t.Parallel()
			ep := tc.endpoint

			ctx, timeout := context.WithTimeout(context.Background(), 10*time.Second)
			defer timeout()

			dealers := []zmq4.Socket{tc.dealer0, tc.dealer1, tc.dealer2}
			fired := make([]bool, len(dealers))

			grp, ctx := errgroup.WithContext(ctx)
			grp.Go(func() error {

				err := tc.router.Listen(ep)
				if err != nil {
					return errors.Wrapf(err, "could not listen")
				}

				seen := make(map[string]int)
				fired := 0
				const N = 3
				for i := 0; i < len(dealers)*N+1 && fired < N; i++ {
					msg, err := tc.router.Recv()
					if err != nil {
						return errors.Wrapf(err, "could not recv message")
					}

					id := string(msg.Frames[0])
					seen[id]++
					switch n := seen[id]; {
					case n >= N:
						msg = zmq4.NewMsgFrom([]byte(id), []byte(""), Fired)
						fired++
					default:
						msg = zmq4.NewMsgFrom([]byte(id), []byte(""), WorkHarder)
					}
					err = tc.router.Send(msg)
					if err != nil {
						return errors.Wrapf(err, "could not send %v", msg)
					}
				}
				if fired != N {
					return errors.Wrapf(err, "did not fire everybody (fired=%d, want=%d)", fired, N)
				}
				return nil
			})
			for idealer := range dealers {
				func(idealer int, dealer zmq4.Socket) {
					grp.Go(func() error {

						err := dealer.Dial(ep)
						if err != nil {
							return errors.Wrapf(err, "could not dial")
						}

					loop:
						for {
							// tell the broker we are ready for work
							err = dealer.Send(ready)
							if err != nil {
								return errors.Wrapf(err, "could not send %v", ready)
							}

							// get workload from broker
							msg, err := dealer.Recv()
							if err != nil {
								return errors.Wrapf(err, "could not recv msg")
							}
							work := msg.Frames[1]
							if bytes.Equal(work, Fired) {
								fired[idealer] = true
								break loop
							}

							// do some random work
							time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)
						}

						return err
					})
				}(idealer, dealers[idealer])
			}

			if err := grp.Wait(); err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(fired, []bool{true, true, true}) {
				t.Fatalf("some workers did not get fired: %v", fired)
			}
		})
	}
}
