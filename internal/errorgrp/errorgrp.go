// Package errorgrp is bit more advanced than errgroup
// Major difference is that when error group is created with WithContext2
// the parent context would implicitly cancel all functions called by Go method.
//
// The name is selected so you can mix regular errgroup and errorgrp in same file.
package errorgrp

import (
	"context"

	"golang.org/x/sync/errgroup"
)

// The Group2 is superior errgroup.Group which aborts whole group
// execution when parent context is cancelled
type Group2 struct {
	grp *errgroup.Group
	ctx context.Context
}

// WithContext2 creates Group2 and store inside parent context
// so the Go method would respect parent context cancellation
func WithContext2(ctx context.Context) (*Group2, context.Context) {
	grp, child_ctx := errgroup.WithContext(ctx)
	return &Group2{grp: grp, ctx: ctx}, child_ctx
}

// Go function would wait for parent context to be cancelled,
// or func f to be complete complete
func (g *Group2) Go(f func() error) {
	g.grp.Go(func() error {
		// If parent context is canceled,
		// just return its error and do not call func f
		select {
		case <-g.ctx.Done():
			return g.ctx.Err()
		default:
		}

		// Create return channel
		// and call func f
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
	})
}

// Wait is direct call to errgroup.Wait
func (g *Group2) Wait() error {
	return g.grp.Wait()
}
