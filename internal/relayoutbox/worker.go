// Package relayoutbox runs bounded delivery attempts for an already durable store.
package relayoutbox

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"sync"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db/sqlite"
	"github.com/ebrgr/TuneTrail/internal/logger"
	"github.com/ebrgr/TuneTrail/internal/relay"
)

// Store exposes only transactional claims and sanitized delivery state.
type Store interface {
	ClaimRelayBatch(context.Context, time.Time) (sqlite.RelayBatch, error)
	RelayTarget(context.Context, sqlite.RelayBatch) (string, string, relay.Permissions, error)
	RelayPayload(context.Context, sqlite.RelayBatch, sqlite.RelayItem) ([]byte, error)
	FailRelaySource(context.Context, sqlite.RelayBatch, sqlite.RelayItem, error) error
	RenewRelayBatch(context.Context, sqlite.RelayBatch, time.Time, ...sqlite.RelayItem) (bool, error)
	FinishRelayBatch(context.Context, sqlite.RelayBatch, []sqlite.RelayItem, relay.Result, time.Time, bool) (bool, error)
	RelayProfiles(context.Context) ([]int32, error)
	ExpireRelayWork(context.Context, int32, time.Time) (int64, error)
}

// Run starts bounded concurrent workers. Stop claims before canceling attempts.
func Run(ctx, claims context.Context, store Store) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		var workers sync.WaitGroup
		for range 4 {
			workers.Go(func() {
				tick := time.NewTicker(250 * time.Millisecond)
				defer tick.Stop()
				poll(ctx, claims, store, time.Now, tick.C)
			})
		}
		workers.Go(func() {
			tick := time.NewTicker(time.Minute)
			defer tick.Stop()
			for {
				select {
				case <-claims.Done():
					return
				case <-ctx.Done():
					return
				case now := <-tick.C:
					profiles, err := store.RelayProfiles(claims)
					if err != nil {
						continue
					}
					for _, profile := range profiles {
						if claims.Err() != nil {
							return
						}
						_, _ = store.ExpireRelayWork(claims, profile, now) // Retry on the next maintenance tick.
					}
				}
			}
		})
		workers.Wait()
	}()
	return done
}

// poll waits after idle or unproductive claims, including temporary SQLite contention.
func poll(ctx, claims context.Context, store Store, now func() time.Time, ticks <-chan time.Time) {
	for claims.Err() == nil && ctx.Err() == nil {
		batch, err := store.ClaimRelayBatch(claims, now())
		if err == nil && len(batch.Items) > 0 && process(ctx, store, batch, now) {
			continue
		}
		select {
		case <-claims.Done():
			return
		case <-ctx.Done():
			return
		case <-ticks:
		}
	}
}

var policyLog struct {
	sync.Mutex
	next time.Time
}

// logPolicyBlock omits network and request data and limits warnings across workers.
func logPolicyBlock(ctx context.Context, profile int32, now time.Time) {
	policyLog.Lock()
	defer policyLog.Unlock()
	if now.Before(policyLog.next) {
		return
	}
	policyLog.next = now.Add(time.Minute)
	logger.FromContext(ctx).Warn().Str("class", string(relay.PolicyBlocked)).Int32("profile_id", profile).Msg("Relay delivery paused")
}

// process persists each sub-lot before starting another POST and always releases the claim.
func process(ctx context.Context, store Store, batch sqlite.RelayBatch, now func() time.Time) (progress bool) {
	defer func() {
		// A failed final write leaves the lease available for crash recovery.
		_, _ = store.FinishRelayBatch(ctx, batch, nil, relay.Result{}, now(), true)
	}()
	if ctx.Err() != nil {
		return
	}
	base, token, permissions, err := store.RelayTarget(ctx, batch)
	if err != nil {
		if ctx.Err() == nil && errors.Is(err, sqlite.ErrRelayConfigInvalid) {
			_, _ = store.FinishRelayBatch(ctx, batch, nil, relay.Result{Class: relay.ConfigInvalid}, now(), true) // Lease recovery covers a failed write.
		}
		return false
	}
	sender := relay.NewDeliverer(permissions, now)
	queue := [][]sqlite.RelayItem{batch.Items}
	for posts := 0; len(queue) > 0 && posts < batch.PostBudget && ctx.Err() == nil; {
		part := queue[0]
		queue = queue[1:]
		items := make([]sqlite.RelayItem, 0, len(part))
		payload := make([][]byte, 0, len(part))
		for _, item := range part {
			raw, err := store.RelayPayload(ctx, batch, item)
			if errors.Is(err, sql.ErrNoRows) || errors.Is(err, sqlite.ErrRelaySourceInvalid) {
				if err := store.FailRelaySource(ctx, batch, item, err); err != nil {
					return
				}
				progress = true
				continue
			}
			if err != nil {
				return
			}
			items = append(items, item)
			payload = append(payload, raw)
		}
		if len(items) == 0 {
			continue
		}
		kind := "single"
		if len(items) > 1 {
			kind = "import"
		}
		// Preserve the versioned bytes for each item across envelope compositions.
		body := append([]byte(`{"listen_type":"`+kind+`","payload":[`), bytes.Join(payload, []byte(","))...)
		body = append(body, ']', '}')
		renewed, err := store.RenewRelayBatch(ctx, batch, now(), items...)
		if err != nil || !renewed {
			return
		}
		posts++
		result := sender.Deliver(ctx, base+"/submit-listens", token, body)
		if ctx.Err() != nil {
			return
		}
		if result.Class == relay.PolicyBlocked {
			logPolicyBlock(ctx, batch.Profile, now())
		}
		settled, err := store.FinishRelayBatch(ctx, batch, items, result, now(), false)
		if err != nil || !settled {
			return
		}
		progress = true
		if result.RemainingZero {
			return
		}
		switch result.Class {
		case relay.Success:
		case relay.BadRequest:
			if len(items) > 1 {
				middle := len(items) / 2
				queue = append([][]sqlite.RelayItem{items[:middle], items[middle:]}, queue...)
			}
		default:
			return
		}
	}
	return progress
}
