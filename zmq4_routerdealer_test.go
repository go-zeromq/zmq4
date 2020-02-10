// Copyright 2018 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4_test

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/go-zeromq/zmq4"
	"golang.org/x/sync/errgroup"
	"golang.org/x/xerrors"
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
			t.Parallel()
			ep := tc.endpoint()
			cleanUp(ep)

			if tc.skip {
				t.Skipf(tc.name)
			}

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
			fired := make([]int, len(dealers))

			var wgd sync.WaitGroup
			wgd.Add(len(dealers))
			var wgr sync.WaitGroup
			wgr.Add(1)

			var seenMu sync.RWMutex
			seen := make(map[string]int)
			grp, ctx := errgroup.WithContext(ctx)
			grp.Go(func() error {

				err := router.Listen(ep)
				if err != nil {
					return xerrors.Errorf("could not listen: %w", err)
				}

				if addr := router.Addr(); addr == nil {
					return xerrors.Errorf("listener with nil Addr")
				}

				wgd.Wait()
				wgr.Done()

				fired := 0
				const N = 3
				for i := 0; i < len(dealers)*N+1 && fired < N; i++ {
					msg, err := router.Recv()
					if err != nil {
						return xerrors.Errorf("could not recv message: %w", err)
					}

					if len(msg.Frames) == 0 {
						seenMu.RLock()
						str := fmt.Sprintf("%v", seen)
						seenMu.RUnlock()
						return xerrors.Errorf("router received empty message (test=%q, iter=%d, seen=%v)", tc.name, i, str)
					}
					id := string(msg.Frames[0])
					seenMu.Lock()
					seen[id]++
					n := seen[id]
					seenMu.Unlock()
					switch {
					case n >= N:
						msg = zmq4.NewMsgFrom([]byte(id), []byte(""), Fired)
						fired++
					default:
						msg = zmq4.NewMsgFrom([]byte(id), []byte(""), WorkHarder)
					}
					err = router.Send(msg)
					if err != nil {
						return xerrors.Errorf("could not send %v: %w", msg, err)
					}
				}
				if fired != N {
					return xerrors.Errorf("did not fire everybody (fired=%d, want=%d)", fired, N)
				}
				return nil
			})
			for idealer := range dealers {
				func(idealer int, dealer zmq4.Socket) {
					grp.Go(func() error {

						err := dealer.Dial(ep)
						if err != nil {
							return xerrors.Errorf("could not dial: %w", err)
						}

						if addr := dealer.Addr(); addr != nil {
							return xerrors.Errorf("dialer with non-nil Addr")
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
								return xerrors.Errorf("could not send %v: %w", ready, err)
							}

							// get workload from broker
							msg, err := dealer.Recv()
							if err != nil {
								return xerrors.Errorf("could not recv msg: %w", err)
							}
							if len(msg.Frames) < 2 {
								seenMu.RLock()
								str := fmt.Sprintf("%v", seen)
								seenMu.RUnlock()
								return xerrors.Errorf("dealer-%d received invalid msg %v (test=%q, iter=%d, seen=%v)", idealer, msg, tc.name, n, str)
							}
							work := msg.Frames[1]
							fired[idealer]++
							if bytes.Equal(work, Fired) {
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
				t.Fatalf("error: %+v", err)
			}

			if !reflect.DeepEqual(fired, []int{3, 3, 3}) {
				t.Fatalf("some workers did not get fired: %v", fired)
			}
		})
	}
}

func TestRouterWithNoDealer(t *testing.T) {
	router := zmq4.NewRouter(context.Background())
	err := router.Listen("tcp://*:*")
	if err != nil {
		t.Fatalf("could not listen: %+v", err)
	}

	err = router.Close()
	if err != nil {
		t.Fatalf("could not close router: %+v", err)
	}
}

func TestRouterDealerClose(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "router"},
		{name: "dealer"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			socks := map[string]zmq4.Socket{
				"router": zmq4.NewRouter(ctx),
				"dealer": zmq4.NewDealer(ctx),
			}
			router := socks["router"]
			dealer := socks["dealer"]

			err := router.Listen("tcp://*:*")
			if err != nil {
				t.Fatalf("router could not listen: %+v", err)
			}
			_, port, _ := net.SplitHostPort(router.Addr().String())
			err = dealer.Dial("tcp://*:" + port)
			if err != nil {
				t.Fatalf("dealer could not dial: %+v", err)
			}
			start := make(chan bool)
			var wg sync.WaitGroup
			wg.Add(1)
			go func(sock zmq4.Socket, start <-chan bool) {
				defer wg.Done()
				<-start
				_, err := sock.Recv()
				if err == nil {
					t.Error("expected error: context canceled")
				}
			}(socks[tt.name], start)

			err = socks[tt.name].Close()
			if err != nil {
				t.Fatalf("could not close %s: %+v", tt.name, err)
			}
			start <- true
			wg.Wait()
		})
	}
}
