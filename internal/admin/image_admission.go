package admin

import (
	"context"
	"sync"

	"github.com/ebrgr/TuneTrail/internal/imagepipeline"
)

type admissionKey struct{}
type admissionLease struct {
	active bool
	mu     sync.Mutex
}

// Image requests enter before HTTP multipart parsing or service-side reads.
var imageSlots = make(chan struct{}, 5)
var imageActive = make(chan struct{}, 1)

// AcquireImageAdmission bounds active administrative image requests and their waiters.
// A derived context reuses the current lease when the HTTP adapter invokes a service.
func AcquireImageAdmission(ctx context.Context) (context.Context, func(), error) {
	if err := ctx.Err(); err != nil {
		return ctx, nil, err
	}
	if lease, ok := ctx.Value(admissionKey{}).(*admissionLease); ok {
		lease.mu.Lock()
		active := lease.active
		lease.mu.Unlock()
		if active {
			return ctx, func() {}, nil
		}
	}
	select {
	case imageSlots <- struct{}{}:
	default:
		return ctx, nil, imagepipeline.ErrBusy
	}
	select {
	case imageActive <- struct{}{}:
		if err := ctx.Err(); err != nil {
			<-imageActive
			<-imageSlots
			return ctx, nil, err
		}
		lease := &admissionLease{active: true}
		var once sync.Once
		release := func() {
			once.Do(func() { lease.mu.Lock(); lease.active = false; lease.mu.Unlock(); <-imageActive; <-imageSlots })
		}
		return context.WithValue(ctx, admissionKey{}, lease), release, nil
	case <-ctx.Done():
		<-imageSlots
		return ctx, nil, ctx.Err()
	}
}
