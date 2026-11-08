package imagework

import (
	"context"
	"errors"
	"github.com/ebrgr/TuneTrail/internal/imagepipeline"
	"sync/atomic"
	"testing"
	"time"
)

// awaitWaiters synchronizes tests without relying on scheduling delays.
func awaitWaiters(t *testing.T, g *Group, key string, n int) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		g.mu.Lock()
		c := g.calls[key]
		ok := c != nil && c.waiters == n
		g.mu.Unlock()
		if ok {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("participants did not join")
}

// TestParticipantsCancelIndependently keeps shared work alive after its first caller leaves.
func TestParticipantsCancelIndependently(t *testing.T) {
	g := New(2, time.Second*5)
	started := make(chan struct{})
	release := make(chan struct{})
	var calls atomic.Int32
	fn := func(ctx context.Context) error {
		calls.Add(1)
		close(started)
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	first, cancel := context.WithCancel(context.Background())
	defer cancel()
	a := make(chan error, 1)
	b := make(chan error, 1)
	go func() { a <- g.Do(first, "same", fn) }()
	<-started
	go func() { b <- g.Do(context.Background(), "same", fn) }()
	awaitWaiters(t, g, "same", 2)
	cancel()
	if err := <-a; !errors.Is(err, context.Canceled) {
		t.Fatalf("first: %v", err)
	}
	close(release)
	if err := <-b; err != nil {
		t.Fatalf("second: %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("work called %d times", calls.Load())
	}
}

// TestLastCancellationKeepsSlotUntilWorkReturns models a codec that ignores cancellation.
func TestLastCancellationKeepsSlotUntilWorkReturns(t *testing.T) {
	g := New(1, time.Second*5)
	started := make(chan struct{})
	release := make(chan struct{})
	finished := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		result <- g.Do(ctx, "a", func(ctx context.Context) error {
			close(started)
			<-ctx.Done()
			<-release
			close(finished)
			return ctx.Err()
		})
	}()
	<-started
	cancel()
	if err := <-result; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err := g.Do(context.Background(), "b", func(context.Context) error { return nil }); !errors.Is(err, imagepipeline.ErrBusy) {
		t.Fatalf("unbounded work: %v", err)
	}
	if err := g.Do(context.Background(), "a", func(context.Context) error { return nil }); !errors.Is(err, imagepipeline.ErrBusy) {
		t.Fatalf("joined canceled work: %v", err)
	}
	close(release)
	<-finished
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		err := g.Do(context.Background(), "b", func(context.Context) error { return nil })
		if err == nil {
			return
		}
		if !errors.Is(err, imagepipeline.ErrBusy) {
			t.Fatal(err)
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("work slot did not return")
}

// TestFollowerCancellationStopsWaiting without stopping the remaining participant.
func TestFollowerCancellationStopsWaiting(t *testing.T) {
	g := New(1, time.Second*5)
	start := make(chan struct{})
	release := make(chan struct{})
	a := make(chan error, 1)
	go func() {
		a <- g.Do(context.Background(), "a", func(context.Context) error { close(start); <-release; return nil })
	}()
	<-start
	ctx, cancel := context.WithCancel(context.Background())
	b := make(chan error, 1)
	go func() { b <- g.Do(ctx, "a", func(context.Context) error { t.Error("duplicate work"); return nil }) }()
	awaitWaiters(t, g, "a", 2)
	cancel()
	if err := <-b; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	close(release)
	if err := <-a; err != nil {
		t.Fatal(err)
	}
}

// TestDeadlineDoesNotAbandonCapacity bounds callers while uninterruptible work finishes.
func TestDeadlineDoesNotAbandonCapacity(t *testing.T) {
	g := New(1, 30*time.Millisecond)
	release := make(chan struct{})
	result := make(chan error, 1)
	go func() {
		result <- g.Do(context.Background(), "a", func(context.Context) error { <-release; return nil })
	}()
	select {
	case err := <-result:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("deadline: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("deadline did not release caller")
	}
	if err := g.Do(context.Background(), "b", func(context.Context) error { return nil }); !errors.Is(err, imagepipeline.ErrBusy) {
		t.Fatalf("untracked work: %v", err)
	}
	close(release)
}
