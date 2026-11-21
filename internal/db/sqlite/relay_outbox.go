package sqlite

import (
	"cmp"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"math/bits"
	"math/rand/v2"
	"slices"
	"strconv"
	"time"

	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/ebrgr/TuneTrail/internal/relay"
	"github.com/ebrgr/TuneTrail/internal/retry"
	"github.com/google/uuid"
)

const (
	relayBatchSize = 100
	relayMaxAge    = 30 * 24 * time.Hour
	relayLease     = 2 * time.Minute
)

const relayDue = `format_version>0 AND state IN ('pending','retryable')`
const relayCapacity = `format_version>0 AND state='failed' AND json_extract(result,'$.reason')='queue_capacity'`

// insertRelayIntent runs after acceptance, under the same writer reservation.
func insertRelayIntent(ctx context.Context, tx *sql.Conn, r claimedReceipt, event string, now int64) error {
	if r.source != ingest.ShadowSourceListenBrainz || (r.deliveryPolicy != "durable" && r.variant == "occurrence") {
		return nil
	}
	destination := "relay:" + strconv.FormatInt(int64(r.profile), 10)
	if _, err := tx.ExecContext(ctx, `INSERT INTO integrations(id,profile_id,source_type,created_at)
		SELECT ?,profile_id,'listenbrainz-relay',? FROM profile_relays WHERE profile_id=? AND url<>'' AND token<>''
		ON CONFLICT(id) DO NOTHING`, destination, now, r.profile); err != nil {
		return err
	}
	if r.deliveryPolicy != "durable" {
		_, err := tx.ExecContext(ctx, `INSERT INTO outbox(id,profile_id,destination_id,event_id,operation,delivery_key,state,created_at,closed_at,result)
			SELECT ?,profile_id,?,?,'listen',?,'discarded',?,?,'{"reason":"pre_082_best_effort"}'
			FROM profile_relays WHERE profile_id=? AND url<>'' AND token<>''`,
			uuid.NewString(), destination, event, event, now, now, r.profile)
		return err
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO outbox(id,profile_id,destination_id,event_id,operation,delivery_key,state,created_at,next_attempt_at,result,format_version)
		SELECT ?,d.profile_id,?,?,'listen',?,CASE WHEN d.open_count<100000 THEN 'pending' ELSE 'failed' END,?,
			CASE WHEN d.open_count<100000 THEN ? END,
			CASE WHEN d.open_count<100000 THEN '{}' ELSE '{"reason":"queue_capacity"}' END,1
		FROM relay_destinations d JOIN profile_relays p USING(profile_id)
		WHERE d.profile_id=? AND p.url<>'' AND p.token<>''`,
		uuid.NewString(), destination, event, event, now, now, r.profile)
	return err
}

// RelayBatch fences a destination and its items without carrying credentials.
type RelayBatch struct {
	Profile                            int32
	Generation, ConfigVersion, Version int64
	Owner                              string
	Items                              []RelayItem
	PostBudget                         int
}

// RelayItem identifies a versioned outbox claim.
type RelayItem struct {
	ID      string
	Version int64
	due     int64
}

// ClaimRelayBatch reserves one destination and at most 100 of its due items.
func (s *Sqlite) ClaimRelayBatch(ctx context.Context, now time.Time) (RelayBatch, error) {
	var batch RelayBatch
	at := now.Unix()
	// The read-only probe keeps idle polls out of the writer queue.
	var candidate int32
	err := s.db.QueryRowContext(ctx, relayCandidate, at, at, at).Scan(&candidate)
	if errors.Is(err, sql.ErrNoRows) {
		return batch, nil
	}
	if err != nil {
		return batch, err
	}
	err = s.withIngestWriter(ctx, func(tx *sql.Conn) error {
		if err := tx.QueryRowContext(ctx, relayCandidate, at, at, at).Scan(&batch.Profile); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			return err
		}
		batch.Owner = uuid.NewString()
		until := now.Add(relayLease).Unix()
		err := tx.QueryRowContext(ctx, `UPDATE relay_destinations SET lease_owner=?,lease_until=?,work_version=work_version+1,last_claim_at=max(?,(SELECT max(last_claim_at)+1 FROM relay_destinations))
			WHERE profile_id=? AND coalesce(lease_until,0)<=?
			RETURNING generation,config_version,work_version`, batch.Owner, until, now.UnixNano(), batch.Profile, at).
			Scan(&batch.Generation, &batch.ConfigVersion, &batch.Version)
		if err != nil {
			return err
		}
		if err := maintainRelayQueue(ctx, tx, batch.Profile, at); err != nil {
			return err
		}
		rows, err := tx.QueryContext(ctx, `UPDATE outbox SET lease_owner=?,lease_until=?,work_version=work_version+1
			WHERE id IN (SELECT id FROM outbox WHERE profile_id=? AND destination_id='relay:'||? AND `+relayDue+`
			 AND next_attempt_at<=? AND coalesce(lease_until,0)<=? AND created_at>? ORDER BY next_attempt_at,id LIMIT ?)
			RETURNING id,work_version,next_attempt_at`, batch.Owner, until, batch.Profile, batch.Profile, at, at, now.Add(-relayMaxAge).Unix(), relayBatchSize)
		if err != nil {
			return err
		}
		for rows.Next() {
			var item RelayItem
			if err := rows.Scan(&item.ID, &item.Version, &item.due); err != nil {
				rows.Close()
				return err
			}
			batch.Items = append(batch.Items, item)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		if len(batch.Items) == 0 {
			return releaseRelayDestination(ctx, tx, batch)
		}
		slices.SortFunc(batch.Items, func(a, b RelayItem) int {
			if c := cmp.Compare(a.due, b.due); c != 0 {
				return c
			}
			return cmp.Compare(a.ID, b.ID)
		})
		batch.PostBudget = 2*bits.Len(uint(len(batch.Items)-1)) + 2
		return nil
	})
	if err != nil {
		return RelayBatch{}, err
	}
	return batch, nil
}

const relayCandidate = `SELECT d.profile_id FROM relay_destinations d JOIN profile_relays p USING(profile_id)
	WHERE p.url<>'' AND p.token<>'' AND coalesce(d.lease_until,0)<=?
	AND (d.pause_class IS NULL OR d.paused_until<=?)
	AND (EXISTS (SELECT 1 FROM outbox WHERE profile_id=d.profile_id AND destination_id='relay:'||d.profile_id AND ` + relayDue + ` AND next_attempt_at<=?)
	OR (d.open_count<100000 AND EXISTS (SELECT 1 FROM outbox WHERE profile_id=d.profile_id AND destination_id='relay:'||d.profile_id AND ` + relayCapacity + `)))
	ORDER BY d.last_claim_at,d.profile_id LIMIT 1`

// ExpireRelayWork marks a bounded number of aged items, including paused destinations.
func (s *Sqlite) ExpireRelayWork(ctx context.Context, profile int32, now time.Time) (int64, error) {
	result, err := s.db.ExecContext(ctx, expireRelayWork, profile, profile, now.Add(-relayMaxAge).Unix())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

const expireRelayWork = `UPDATE outbox SET state='failed',result='{"reason":"expired"}',next_attempt_at=NULL,lease_owner=NULL,lease_until=NULL
	WHERE id IN (SELECT id FROM outbox WHERE profile_id=? AND destination_id='relay:'||? AND created_at<=?
	AND lease_owner IS NULL AND ((` + relayDue + `) OR (` + relayCapacity + `)) LIMIT 100)`

// maintainRelayQueue expires bounded work and readmits the oldest capacity failures.
func maintainRelayQueue(ctx context.Context, tx *sql.Conn, profile int32, now int64) error {
	cutoff := now - int64(relayMaxAge/time.Second)
	// Inspect a bounded due window here. The separate expiry operation also covers paused destinations.
	_, err := tx.ExecContext(ctx, `UPDATE outbox SET state='failed',result='{"reason":"expired"}',
		next_attempt_at=NULL,lease_owner=NULL,lease_until=NULL WHERE created_at<=?
		AND id IN (SELECT id FROM outbox WHERE profile_id=? AND destination_id='relay:'||? AND `+relayDue+`
		AND next_attempt_at<=? AND coalesce(lease_until,0)<=? ORDER BY next_attempt_at,id LIMIT 100)`, cutoff, profile, profile, now, now)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE outbox SET state='pending',result='{}',next_attempt_at=?
		WHERE id IN (SELECT id FROM outbox WHERE profile_id=? AND destination_id='relay:'||? AND `+relayCapacity+`
		AND created_at>? ORDER BY created_at,id LIMIT (SELECT min(100,100000-open_count) FROM relay_destinations WHERE profile_id=?))`, now, profile, profile, cutoff, profile)
	return err
}

// RetryRelayItem explicitly readmits a failed item against the current destination.
func (s *Sqlite) RetryRelayItem(ctx context.Context, profile int32, id string, now time.Time) (bool, error) {
	result, err := s.db.ExecContext(ctx, `UPDATE outbox SET state='pending',result='{}',next_attempt_at=?
		WHERE profile_id=? AND id=? AND destination_id='relay:'||? AND state='failed' AND format_version>0
		  AND created_at>? AND attempt_count<8
		  AND EXISTS (SELECT 1 FROM relay_destinations d JOIN profile_relays p USING(profile_id)
		      WHERE d.profile_id=outbox.profile_id AND d.open_count<100000 AND p.url<>'' AND p.token<>'')`,
		now.Unix(), profile, id, profile, now.Add(-relayMaxAge).Unix())
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n == 1, err
}

// RenewRelayBatch extends ownership before a POST. Changed credentials require a new claim.
func (s *Sqlite) RenewRelayBatch(ctx context.Context, batch RelayBatch, now time.Time) (bool, error) {
	var renewed bool
	err := s.withIngestWriter(ctx, func(tx *sql.Conn) error {
		until := now.Add(relayLease).Unix()
		res, err := tx.ExecContext(ctx, `UPDATE relay_destinations SET lease_until=? WHERE profile_id=? AND generation=? AND config_version=?
			AND work_version=? AND lease_owner=?`, until, batch.Profile, batch.Generation, batch.ConfigVersion, batch.Version, batch.Owner)
		if err != nil {
			return err
		}
		n, err := res.RowsAffected()
		if err != nil || n == 0 {
			return err
		}
		renewed = true
		// Items already finalized no longer carry this claim owner.
		_, err = tx.ExecContext(ctx, `UPDATE outbox SET lease_until=? WHERE profile_id=? AND lease_owner=?`, until, batch.Profile, batch.Owner)
		return err
	})
	return renewed && err == nil, err
}

// FinishRelayBatch records results for a sub-batch or releases the claim on interruption.
func (s *Sqlite) FinishRelayBatch(ctx context.Context, batch RelayBatch, items []RelayItem, result relay.Result, now time.Time, release bool) (bool, error) {
	canceled := ctx.Err() != nil
	if canceled {
		// The caller's cancellation must not prevent releasing durable ownership.
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		defer cancel()
		release = true
		items = nil
	}
	finished := false
	err := s.withIngestWriter(ctx, func(tx *sql.Conn) error {
		var version, failures int64
		err := tx.QueryRowContext(ctx, `SELECT config_version,backoff_count FROM relay_destinations
			WHERE profile_id=? AND generation=? AND work_version=? AND lease_owner=?`,
			batch.Profile, batch.Generation, batch.Version, batch.Owner).Scan(&version, &failures)
		if errors.Is(err, sql.ErrNoRows) {
			// URL changes invalidate items but preserve the physical lease until the old attempt returns.
			return releaseRelayDestination(ctx, tx, batch)
		}
		if err != nil {
			return err
		}
		finished = true
		if !canceled {
			terminal := result.Class == relay.Success || (result.Class == relay.BadRequest && len(items) == 1 && (result.Status == 400 || result.Status == 413))
			for _, item := range items {
				if err := finishRelayItem(ctx, tx, batch, item, result, terminal, now.Unix()); err != nil {
					return err
				}
			}
			if version == batch.ConfigVersion {
				paused, err := pauseRelayDestination(ctx, tx, batch, result, now, failures)
				if err != nil {
					return err
				}
				release = release || paused
			}
		}
		if !release {
			return nil
		}
		_, err = tx.ExecContext(ctx, `UPDATE outbox SET state='retryable',next_attempt_at=?,lease_owner=NULL,lease_until=NULL
			WHERE profile_id=? AND lease_owner=? AND `+relayDue, now.Unix(), batch.Profile, batch.Owner)
		if err != nil {
			return err
		}
		return releaseRelayDestination(ctx, tx, batch)
	})
	return finished && err == nil, err
}

// finishRelayItem applies the claim fence even if the lease has expired.
func finishRelayItem(ctx context.Context, tx *sql.Conn, batch RelayBatch, item RelayItem, result relay.Result, terminal bool, at int64) error {
	reason := ""
	if terminal && result.Class == relay.BadRequest {
		reason = "item_rejected"
	}
	data, err := json.Marshal(struct {
		Class  relay.Class `json:"class"`
		Status int         `json:"status"`
		Reason string      `json:"reason,omitempty"`
	}{Class: result.Class, Status: result.Status, Reason: reason})
	if err != nil {
		return err
	}
	if !terminal {
		_, err := tx.ExecContext(ctx, `UPDATE outbox SET result=?,last_attempt_at=?
			WHERE profile_id=? AND id=? AND work_version=? AND lease_owner=? AND `+relayDue,
			string(data), at, batch.Profile, item.ID, item.Version, batch.Owner)
		return err
	}
	state := "failed"
	var closed any
	increment := 1
	if result.Class == relay.Success {
		state, closed, increment = "sent", at, 0
	}
	_, err = tx.ExecContext(ctx, `UPDATE outbox SET state=?,closed_at=?,result=?,attempt_count=min(8,attempt_count+?),last_attempt_at=?,
		next_attempt_at=NULL,lease_owner=NULL,lease_until=NULL
		WHERE profile_id=? AND id=? AND work_version=? AND lease_owner=? AND `+relayDue,
		state, closed, string(data), increment, at, batch.Profile, item.ID, item.Version, batch.Owner)
	return err
}

// releaseRelayDestination uses the physical claim fence without comparing its expiry.
func releaseRelayDestination(ctx context.Context, tx *sql.Conn, batch RelayBatch) error {
	_, err := tx.ExecContext(ctx, `UPDATE relay_destinations SET lease_owner=NULL,lease_until=NULL
		WHERE profile_id=? AND work_version=? AND lease_owner=?`, batch.Profile, batch.Version, batch.Owner)
	return err
}

// pauseRelayDestination applies server waits and per-destination backoff.
func pauseRelayDestination(ctx context.Context, tx *sql.Conn, batch RelayBatch, result relay.Result, now time.Time, failures int64) (bool, error) {
	class := result.Class
	switch class {
	case relay.Success, relay.BadRequest:
		if !result.RemainingZero {
			_, err := tx.ExecContext(ctx, "UPDATE relay_destinations SET backoff_count=0 WHERE profile_id=?", batch.Profile)
			return false, err
		}
		class = relay.RateLimited
	case relay.Auth, relay.Route, relay.Redirect, relay.PolicyBlocked:
		_, err := tx.ExecContext(ctx, "UPDATE relay_destinations SET paused_until=NULL,pause_class=? WHERE profile_id=?", class, batch.Profile)
		return true, err
	}
	base := 30 * time.Second
	if class == relay.TLS {
		base = 15 * time.Minute
	}
	delay := retry.FullJitter(base, time.Hour, failures, rand.Int64N)
	until := now.Add(max(delay, result.ServerWait)).Unix()
	_, err := tx.ExecContext(ctx, `UPDATE relay_destinations SET paused_until=?,pause_class=?,backoff_count=backoff_count+1 WHERE profile_id=?`, until, class, batch.Profile)
	return true, err
}
