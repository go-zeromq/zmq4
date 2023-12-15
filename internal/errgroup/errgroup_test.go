// Copyright 2023 The go-zeromq Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package errgroup

import (
	"context"
	"fmt"
	"testing"

	"golang.org/x/sync/errgroup"
)

// TestRegularErrGroupDoesNotRespectParentContext checks regular errgroup behavior
// where errgroup.WithContext does not respect the parent context
func TestRegularErrGroupDoesNotRespectParentContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	eg, _ := errgroup.WithContext(ctx)

	what := fmt.Errorf("func generated error")
	ch := make(chan error)
	eg.Go(func() error { return <-ch })

	cancel()   // abort parent context
	ch <- what // signal the func in regular errgroup to fail
	err := eg.Wait()

	// The error shall be one returned by the function
	// as regular errgroup.WithContext does not respect parent context
	if err != what {
		t.Errorf("invalid error. got=%+v, want=%+v", err, what)
	}
}

// TestErrGroupWithContextCanCallFunctions checks the errgroup operations
// are fine working and errgroup called function can return error
func TestErrGroupWithContextCanCallFunctions(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	eg, _ := WithContext(ctx)

	what := fmt.Errorf("func generated error")
	ch := make(chan error)
	eg.Go(func() error { return <-ch })

	ch <- what       // signal the func in errgroup to fail
	err := eg.Wait() // wait errgroup complete and read error

	// The error shall be one returned by the function
	if err != what {
		t.Errorf("invalid error. got=%+v, want=%+v", err, what)
	}
}

// TestErrGroupWithContextDoesRespectParentContext checks the errgroup operations
// are cancellable by parent context
func TestErrGroupWithContextDoesRespectParentContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	eg, _ := WithContext(ctx)

	s1 := make(chan struct{})
	s2 := make(chan struct{})
	eg.Go(func() error {
		s1 <- struct{}{}
		<-s2
		return fmt.Errorf("func generated error")
	})

	// We have no set limit to errgroup so
	// shall be able to start function via TryGo
	if ok := eg.TryGo(func() error { return nil }); !ok {
		t.Errorf("Expected TryGo to be able start function!!!")
	}

	<-s1     // wait for function to start
	cancel() // abort parent context

	eg.Go(func() error {
		t.Errorf("The parent context was already cancelled and this function shall not be called!!!")
		return nil
	})

	s2 <- struct{}{} // signal the func in regular errgroup to fail
	err := eg.Wait() // wait errgroup complete and read error

	// The error shall be one returned by the function
	// as regular errgroup.WithContext does not respect parent context
	if err != context.Canceled {
		t.Errorf("expected a context.Canceled error, got=%+v", err)
	}
}

// TestErrGroupFallback tests fallback logic to be compatible with x/sync/errgroup
func TestErrGroupFallback(t *testing.T) {
	eg := Group{}
	eg.SetLimit(2)

	ch1 := make(chan error)
	eg.Go(func() error { return <-ch1 })

	ch2 := make(chan error)
	ok := eg.TryGo(func() error { return <-ch2 })
	if !ok {
		t.Errorf("Expected errgroup.TryGo to success!!!")
	}

	// The limit set to 2, so 3rd function shall not be possible to call
	ok = eg.TryGo(func() error {
		t.Errorf("This function is unexpected to be called!!!")
		return nil
	})
	if ok {
		t.Errorf("Expected errgroup.TryGo to fail!!!")
	}

	ch1 <- nil
	ch2 <- nil
	err := eg.Wait()

	if err != nil {
		t.Errorf("expected a nil error, got=%+v", err)
	}
}
