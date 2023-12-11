package errorgrp

import (
	"context"
	"fmt"
	"testing"

	"golang.org/x/sync/errgroup"
)

// TestErrGroupDoesNotRespectParentContext check regulare errgroup behavior
// where errgroup.WithContext does not respects the parent context
func TestErrGroupDoesNotRespectParentContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	eg, _ := errgroup.WithContext(ctx)

	er := fmt.Errorf("func generated  error")
	s := make(chan struct{}, 1)
	eg.Go(func() error {
		<-s
		return er
	})

	// Abort context
	cancel()
	// Signal the func in regular errgroup to fail
	s <- struct{}{}
	// Wait regular errgroup complete and read error
	err := eg.Wait()

	// The error shall be one returned by the function
	// as regular errgroup.WithContext does not respect parent context
	if err != er {
		t.Fail()
	}
}

func TestErrorGrpWithContext2DoesRespectsParentContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	eg, _ := WithContext2(ctx)

	er := fmt.Errorf("func generated  error")
	s := make(chan struct{}, 1)
	eg.Go(func() error {
		<-s
		return er
	})

	// Abort context
	cancel()
	// Signal the func in regular errgroup to fail
	s <- struct{}{}
	// Wait regular errgroup complete and read error
	err := eg.Wait()

	// The error shall be one returned by the function
	// as regular errgroup.WithContext does not respect parent context
	if err != context.Canceled {
		t.Fail()
	}
}
