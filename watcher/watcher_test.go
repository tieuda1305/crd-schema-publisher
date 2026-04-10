package watcher

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestDebounce_CoalescesRapidEvents(t *testing.T) {
	var count atomic.Int32
	trigger := make(chan struct{}, 10)
	done := make(chan struct{})

	go debounceLoop(trigger, 100*time.Millisecond, func() error {
		count.Add(1)
		return nil
	}, done)

	// Send 5 events in rapid succession
	for i := 0; i < 5; i++ {
		trigger <- struct{}{}
		time.Sleep(10 * time.Millisecond)
	}

	// Wait for debounce to fire
	time.Sleep(300 * time.Millisecond)
	close(done)

	if c := count.Load(); c != 1 {
		t.Fatalf("expected 1 publish cycle, got %d", c)
	}
}

func TestDebounce_SkipsWhenPublishInProgress(t *testing.T) {
	var count atomic.Int32
	trigger := make(chan struct{}, 10)
	done := make(chan struct{})

	go debounceLoop(trigger, 50*time.Millisecond, func() error {
		count.Add(1)
		// Simulate slow publish
		time.Sleep(300 * time.Millisecond)
		return nil
	}, done)

	// First event triggers publish
	trigger <- struct{}{}
	time.Sleep(100 * time.Millisecond) // debounce fires, publish starts (takes 300ms)

	// Second event during publish — debounce fires but publish in progress, skip
	trigger <- struct{}{}
	time.Sleep(100 * time.Millisecond) // debounce fires while first publish still running

	// Wait for everything to settle
	time.Sleep(500 * time.Millisecond)
	close(done)

	// Only 1 publish should have run (second was skipped)
	if c := count.Load(); c != 1 {
		t.Fatalf("expected 1 publish cycle (second skipped), got %d", c)
	}
}

func TestDebounce_DrainsInFlightPublishOnShutdown(t *testing.T) {
	var count atomic.Int32
	trigger := make(chan struct{}, 10)
	done := make(chan struct{})

	var loopDone sync.WaitGroup
	loopDone.Add(1)

	go func() {
		defer loopDone.Done()
		debounceLoop(trigger, 50*time.Millisecond, func() error {
			count.Add(1)
			// Simulate slow publish
			time.Sleep(500 * time.Millisecond)
			return nil
		}, done)
	}()

	// Trigger a publish
	trigger <- struct{}{}
	time.Sleep(100 * time.Millisecond) // debounce fires, publish starts

	// Shut down while publish is in progress
	close(done)

	// debounceLoop should block until publish completes, then return
	loopDone.Wait()

	// The in-flight publish must have completed (not been killed)
	if c := count.Load(); c != 1 {
		t.Fatalf("expected 1 publish cycle to complete during drain, got %d", c)
	}
}
