// Copyright 2020 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zmq4

import (
	"context"
	"fmt"
	"log"
	"sync"

	"golang.org/x/sync/errgroup"
)

// Proxy connects a frontend socket to a backend socket.
type Proxy struct {
	ctx  context.Context // life-line of proxy
	grp  *errgroup.Group
	cmds chan proxyCmd
}

type proxyCmd byte

const (
	proxyStats proxyCmd = iota
	proxyPause
	proxyResume
	proxyKill
)

// NewProxy creates a new Proxy value.
// It proxies messages received on the frontend to the backend (and vice versa)
// If capture is not nil, messages proxied are also sent on that socket.
//
// Conceptually, data flows from frontend to backend. Depending on the
// socket types, replies may flow in the opposite direction.
// The direction is conceptual only; the proxy is fully symmetric and
// there is no technical difference between frontend and backend.
//
// Before creating a Proxy, users must set any socket options,
// and Listen or Dial both frontend and backend sockets.
func NewProxy(ctx context.Context, front, back, capture Socket) *Proxy {
	grp, ctx := errgroup.WithContext(ctx)
	proxy := Proxy{
		ctx:  ctx,
		grp:  grp,
		cmds: make(chan proxyCmd),
	}
	proxy.init(front, back, capture)
	return &proxy
}

func (p *Proxy) Pause()  { p.cmds <- proxyPause }
func (p *Proxy) Stats()  { p.cmds <- proxyStats }
func (p *Proxy) Resume() { p.cmds <- proxyResume }
func (p *Proxy) Kill()   { p.cmds <- proxyKill }

// Run runs the proxy loop.
func (p *Proxy) Run() error {
	return p.grp.Wait()
}

func (p *Proxy) init(front, back, capture Socket) {
	canRecv := func(sck Socket) bool {
		switch sck.Type() {
		case Push:
			return false
		default:
			return true
		}
	}

	canSend := func(sck Socket) bool {
		switch sck.Type() {
		case Pull:
			return false
		default:
			return true
		}
	}

	type Pipe struct {
		name string
		dst  Socket
		src  Socket
	}

	var (
		quit  = make(chan struct{})
		pipes = []Pipe{
			{
				name: "backend",
				dst:  back,
				src:  front,
			},
			{
				name: "frontend",
				dst:  front,
				src:  back,
			},
		}
	)

	// workers makes sure all goroutines are launched and scheduled.
	var workers sync.WaitGroup
	workers.Add(len(pipes) + 1)
	for i := range pipes {
		pipe := pipes[i]
		if pipe.src == nil || !canRecv(pipe.src) {
			workers.Done()
			continue
		}
		p.grp.Go(func() error {
			workers.Done()
			canSend := canSend(pipe.dst)
			for {
				msg, err := pipe.src.Recv()
				select {
				case <-p.ctx.Done():
					return p.ctx.Err()
				case <-quit:
					return nil
				default:
					if canSend {
						err = pipe.dst.Send(msg)
						if err != nil {
							log.Printf("could not forward to %s: %+v", pipe.name, err)
							continue
						}
					}
					if err == nil && capture != nil && len(msg.Frames) != 0 {
						_ = capture.Send(msg)
					}
				}
			}
		})
	}

	p.grp.Go(func() error {
		workers.Done()
		for {
			select {
			case <-p.ctx.Done():
				return p.ctx.Err()
			case cmd := <-p.cmds:
				switch cmd {
				case proxyPause, proxyResume, proxyStats:
					// TODO
				case proxyKill:
					close(quit)
					return nil
				default:
					// API error. panic.
					panic(fmt.Errorf("invalid control socket command: %v", cmd))
				}
			}
		}
	})

	// wait for all worker routines to be scheduled.
	workers.Wait()
}
