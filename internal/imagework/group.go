// Package imagework shares bounded image operations between independently cancellable callers.
package imagework

import (
	"context"
	"sync"
	"time"

	"github.com/ebrgr/TuneTrail/internal/imagepipeline"
)

type call struct {
	done     chan struct{}
	workCtx  context.Context
	cancel   context.CancelFunc
	waiters  int
	finished bool
	err      error
}

// Group limits distinct in-flight operations, including codecs still finishing after cancellation.
type Group struct {
	mu      sync.Mutex
	calls   map[string]*call
	limit   int
	timeout time.Duration
}

// New returns a group with a bound on distinct work and its maximum cooperative duration.
func New(limit int, timeout time.Duration) *Group {
	if limit <= 0 || timeout <= 0 {
		panic("invalid image work limit")
	}
	return &Group{calls: make(map[string]*call), limit: limit, timeout: timeout}
}

// Do shares one operation while allowing each caller to cancel its own wait.
// When all callers leave, work is cancelled and remains tracked until it returns.
func (g *Group) Do(ctx context.Context, key string, fn func(context.Context) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	g.mu.Lock()
	c := g.calls[key]
	if c != nil && c.waiters == 0 {
		g.mu.Unlock()
		return imagepipeline.ErrBusy
	}
	if c == nil {
		if len(g.calls) >= g.limit {
			g.mu.Unlock()
			return imagepipeline.ErrBusy
		}
		workCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), g.timeout)
		c = &call{done: make(chan struct{}), cancel: cancel, workCtx: workCtx}
		g.calls[key] = c
		go func() {
			err := fn(workCtx)
			g.mu.Lock()
			c.err = err
			c.finished = true
			delete(g.calls, key)
			close(c.done)
			g.mu.Unlock()
			cancel()
		}()
	}
	c.waiters++
	g.mu.Unlock()
	var err error
	select {
	case <-ctx.Done():
		err = ctx.Err()
	case <-c.done:
		err = c.err
	case <-c.workCtx.Done():
		// A non-interruptible codec stays tracked, but callers need not wait past the deadline.
		g.mu.Lock()
		if c.finished {
			err = c.err
		} else {
			err = c.workCtx.Err()
		}
		g.mu.Unlock()
	}
	g.mu.Lock()
	c.waiters--
	if c.waiters == 0 && !c.finished {
		c.cancel()
	}
	g.mu.Unlock()
	return err
}
