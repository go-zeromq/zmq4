// Copyright 2023 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package errgroup is bit more advanced than golang.org/x/sync/errgroup.
// Major difference is that when error group is created with WithContext
// the parent context would implicitly cancel all functions called by Go method.
package errgroup

import (
	"context"

	"golang.org/x/sync/errgroup"
)

// The Group is superior errgroup.Group which aborts whole group
// execution when parent context is cancelled
type Group struct {
	grp *errgroup.Group
	ctx context.Context
}

// WithContext creates Group and store inside parent context
// so the Go method would respect parent context cancellation
func WithContext(ctx context.Context) (*Group, context.Context) {
	grp, child_ctx := errgroup.WithContext(ctx)
	return &Group{grp: grp, ctx: ctx}, child_ctx
}

// Go runs the provided f function in a dedicated goroutine and waits for its
// completion or for the parent context cancellation.
func (g *Group) Go(f func() error) {
	g.getErrGroup().Go(g.wrap(f))
}

// Wait blocks until all function calls from the Go method have returned, then
// returns the first non-nil error (if any) from them.
// If the error group was created via WithContext then the Wait returns error
// of cancelled parent context prior any functions calls complete.
func (g *Group) Wait() error {
	return g.getErrGroup().Wait()
}

// SetLimit limits the number of active goroutines in this group to at most n.
// A negative value indicates no limit.
//
// Any subsequent call to the Go method will block until it can add an active
// goroutine without exceeding the configured limit.
//
// The limit must not be modified while any goroutines in the group are active.
func (g *Group) SetLimit(n int) {
	g.getErrGroup().SetLimit(n)
}

// TryGo calls the given function in a new goroutine only if the number of
// active goroutines in the group is currently below the configured limit.
//
// The return value reports whether the goroutine was started.
func (g *Group) TryGo(f func() error) bool {
	return g.getErrGroup().TryGo(g.wrap(f))
}

func (g *Group) wrap(f func() error) func() error {
	if g.ctx == nil {
		return f
	}

	return func() error {
		// If parent context is canceled,
		// just return its error and do not call func f
		select {
		case <-g.ctx.Done():
			return g.ctx.Err()
		default:
		}

		// Create return channel and call func f
		// Buffered channel is used as the following select
		// may be exiting by context cancellation
		// and in such case the write to channel can be block
		// and cause the go routine leak
		ch := make(chan error, 1)
		go func() {
			ch <- f()
		}()

		// Wait func f complete or
		// parent context to be cancelled,
		select {
		case err := <-ch:
			return err
		case <-g.ctx.Done():
			return g.ctx.Err()
		}
	}
}

// The getErrGroup returns actual x/sync/errgroup.Group.
// If the group is not allocated it would implicitly allocate it.
// Thats allows the internal/errgroup.Group be fully
// compatible to x/sync/errgroup.Group
func (g *Group) getErrGroup() *errgroup.Group {
	if g.grp == nil {
		g.grp = &errgroup.Group{}
	}
	return g.grp
}
