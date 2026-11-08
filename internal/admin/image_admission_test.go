package admin

import (
	"context"
	"errors"
	"github.com/ebrgr/TuneTrail/internal/imagepipeline"
	"testing"
	"time"
)

// TestImageAdmissionBoundsAndCancel checks nested leases, four waiting slots and overload.
func TestImageAdmissionBoundsAndCancel(t *testing.T) {
	ctx, release, err := AcquireImageAdmission(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	nested, nestedRelease, err := AcquireImageAdmission(ctx)
	if err != nil || nested != ctx {
		t.Fatalf("nested: %v", err)
	}
	nestedRelease()
	type result struct {
		release func()
		err     error
	}
	out := make(chan result, 4)
	var cancels []context.CancelFunc
	for range 4 {
		ctx, cancel := context.WithCancel(context.Background())
		cancels = append(cancels, cancel)
		go func() { _, release, err := AcquireImageAdmission(ctx); out <- result{release, err} }()
	}
	deadline := time.Now().Add(3 * time.Second)
	for len(imageSlots) != 5 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if len(imageSlots) != 5 {
		t.Fatal("waiters not admitted")
	}
	if _, _, err := AcquireImageAdmission(context.Background()); !errors.Is(err, imagepipeline.ErrBusy) {
		t.Fatalf("overload: %v", err)
	}
	for _, cancel := range cancels {
		cancel()
	}
	for range 4 {
		r := <-out
		if !errors.Is(r.err, context.Canceled) {
			t.Fatalf("waiter: %v", r.err)
		}
	}
	release()
	_, again, err := AcquireImageAdmission(ctx)
	if err != nil {
		t.Fatalf("stale lease: %v", err)
	}
	again()
}

// TestImageLocksCancelWhileWaiting checks cancellation independently of mutex holders.
func TestImageLocksCancelWhileWaiting(t *testing.T) {
	profileMu.Lock()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := profileMu.LockContext(ctx)
	profileMu.Unlock()
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
