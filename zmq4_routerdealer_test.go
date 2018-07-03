// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4_test

import (
	"bytes"
	"context"
	"reflect"
	"sync"
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
			endpoint: func() string { return must(EndPoint("tcp")) },
			router: func(ctx context.Context) zmq4.Socket {
				return zmq4.NewRouter(ctx, zmq4.WithID(zmq4.SocketIdentity("router")))
			},
			dealer0: func(ctx context.Context) zmq4.Socket {
				return zmq4.NewDealer(ctx, zmq4.WithID(zmq4.SocketIdentity("dealer-0")))
			},
			dealer1: func(ctx context.Context) zmq4.Socket {
				return zmq4.NewDealer(ctx, zmq4.WithID(zmq4.SocketIdentity("dealer-1")))
			},
			dealer2: func(ctx context.Context) zmq4.Socket {
				return zmq4.NewDealer(ctx, zmq4.WithID(zmq4.SocketIdentity("dealer-2")))
			},
		},
		{
			name:     "ipc-router-dealer",
			skip:     true,
			endpoint: func() string { return must(EndPoint("ipc")) },
			router: func(ctx context.Context) zmq4.Socket {
				return zmq4.NewRouter(ctx, zmq4.WithID(zmq4.SocketIdentity("router")))
			},
			dealer0: func(ctx context.Context) zmq4.Socket {
				return zmq4.NewDealer(ctx, zmq4.WithID(zmq4.SocketIdentity("dealer-0")))
			},
			dealer1: func(ctx context.Context) zmq4.Socket {
				return zmq4.NewDealer(ctx, zmq4.WithID(zmq4.SocketIdentity("dealer-1")))
			},
			dealer2: func(ctx context.Context) zmq4.Socket {
				return zmq4.NewDealer(ctx, zmq4.WithID(zmq4.SocketIdentity("dealer-2")))
			},
		},
		{
			name:     "inproc-router-dealer",
			skip:     true,
			endpoint: func() string { return must(EndPoint("inproc")) },
			router: func(ctx context.Context) zmq4.Socket {
				return zmq4.NewRouter(ctx, zmq4.WithID(zmq4.SocketIdentity("router")))
			},
			dealer0: func(ctx context.Context) zmq4.Socket {
				return zmq4.NewDealer(ctx, zmq4.WithID(zmq4.SocketIdentity("dealer-0")))
			},
			dealer1: func(ctx context.Context) zmq4.Socket {
				return zmq4.NewDealer(ctx, zmq4.WithID(zmq4.SocketIdentity("dealer-1")))
			},
			dealer2: func(ctx context.Context) zmq4.Socket {
				return zmq4.NewDealer(ctx, zmq4.WithID(zmq4.SocketIdentity("dealer-2")))
			},
		},
	}
)

type testCaseRouterDealer struct {
	name     string
	skip     bool
	endpoint func() string
	router   func(context.Context) zmq4.Socket
	dealer0  func(context.Context) zmq4.Socket
	dealer1  func(context.Context) zmq4.Socket
	dealer2  func(context.Context) zmq4.Socket
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
			if tc.skip {
				t.Skipf(tc.name)
			}
			t.Parallel()
			ep := tc.endpoint()
			cleanUp(ep)

			ctx, timeout := context.WithTimeout(context.Background(), 10*time.Second)
			defer timeout()

			router := tc.router(ctx)
			defer router.Close()

			dealer0 := tc.dealer0(ctx)
			defer dealer0.Close()
			dealer1 := tc.dealer1(ctx)
			defer dealer1.Close()
			dealer2 := tc.dealer2(ctx)
			defer dealer2.Close()

			dealers := []zmq4.Socket{dealer0, dealer1, dealer2}
			fired := make([]bool, len(dealers))

			var wgd sync.WaitGroup
			wgd.Add(len(dealers))
			var wgr sync.WaitGroup
			wgr.Add(1)

			grp, ctx := errgroup.WithContext(ctx)
			grp.Go(func() error {

				err := router.Listen(ep)
				if err != nil {
					return errors.Wrapf(err, "could not listen")
				}

				wgd.Wait()
				wgr.Done()

				seen := make(map[string]int)
				fired := 0
				const N = 3
				for i := 0; i < len(dealers)*N+1 && fired < N; i++ {
					msg, err := router.Recv()
					if err != nil {
						return errors.Wrapf(err, "could not recv message")
					}

					if len(msg.Frames) == 0 {
						return errors.Errorf("router received empty message (test=%q, iter=%d, seen=%v)", tc.name, i, seen)
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
					err = router.Send(msg)
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

						wgd.Done()
						wgd.Wait()
						wgr.Wait()

						n := 0
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
							if len(msg.Frames) < 2 {
								return errors.Errorf("dealer-%d received invalid msg %v (test=%q, iter=%d)", idealer, msg, tc.name, n)
							}
							work := msg.Frames[1]
							if bytes.Equal(work, Fired) {
								fired[idealer] = true
								break loop
							}

							// do some random work
							time.Sleep(50 * time.Millisecond)
							n++
						}

						return err
					})
				}(idealer, dealers[idealer])
			}

			if err := grp.Wait(); err != nil {
				t.Errorf("workers: %v", fired)
				t.Fatal(err)
			}

			if !reflect.DeepEqual(fired, []bool{true, true, true}) {
				t.Fatalf("some workers did not get fired: %v", fired)
			}
		})
	}
}
