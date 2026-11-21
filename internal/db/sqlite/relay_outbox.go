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
	_, err := tx.ExecContext(ctx, `INSERT INTO outbox(id,profile_id,destination_id,event_id,operation,delivery_key,state,created_at,next_attempt_at,result,format_version,relay_generation)
		SELECT ?,d.profile_id,?,?,'listen',?,CASE WHEN d.open_count<100000 THEN 'pending' ELSE 'failed' END,?,
			CASE WHEN d.open_count<100000 THEN ? END,
			CASE WHEN d.open_count<100000 THEN '{}' ELSE '{"reason":"queue_capacity"}' END,1,d.generation
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
		if err := maintainRelayQueue(ctx, tx, batch.Profile, batch.Generation, at); err != nil {
			return err
		}
		rows, err := tx.QueryContext(ctx, `UPDATE outbox SET lease_owner=?,lease_until=?,work_version=work_version+1
			WHERE id IN (SELECT id FROM outbox WHERE profile_id=? AND destination_id='relay:'||? AND relay_generation=? AND `+relayDue+`
			 AND next_attempt_at<=? AND coalesce(lease_until,0)<=? AND created_at>? ORDER BY next_attempt_at,id LIMIT ?)
			RETURNING id,work_version,next_attempt_at`, batch.Owner, until, batch.Profile, batch.Profile, batch.Generation, at, at, now.Add(-relayMaxAge).Unix(), relayBatchSize)
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
	AND (EXISTS (SELECT 1 FROM outbox WHERE profile_id=d.profile_id AND destination_id='relay:'||d.profile_id AND relay_generation=d.generation AND ` + relayDue + ` AND next_attempt_at<=?)
	OR (d.open_count<100000 AND EXISTS (SELECT 1 FROM outbox WHERE profile_id=d.profile_id AND destination_id='relay:'||d.profile_id AND relay_generation=d.generation AND ` + relayCapacity + `)))
	ORDER BY d.last_claim_at,d.profile_id LIMIT 1`

// ExpireRelayWork normalizes stale generations and expires bounded work even while paused.
func (s *Sqlite) ExpireRelayWork(ctx context.Context, profile int32, now time.Time) (int64, error) {
	var total int64
	err := s.withIngestWriter(ctx, func(tx *sql.Conn) error {
		var generation int64
		err := tx.QueryRowContext(ctx, "SELECT generation FROM relay_destinations WHERE profile_id=?", profile).Scan(&generation)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if err := invalidateStaleRelayItems(ctx, tx, profile, generation); err != nil {
			return err
		}
		cutoff := now.Add(-relayMaxAge).Unix()
		// Retention is evaluated at UTC day precision, matching the archived data.
		if _, err := tx.ExecContext(ctx, "DELETE FROM relay_sent_archive WHERE profile_id=? AND sent_day<=?", profile, cutoff/86400); err != nil {
			return err
		}
		for _, scope := range []string{relayDue, relayCapacity} {
			result, err := tx.ExecContext(ctx, expireRelayWork+scope+" LIMIT 100)", profile, profile, generation, cutoff, now.Unix())
			if err != nil {
				return err
			}
			n, err := result.RowsAffected()
			if err != nil {
				return err
			}
			total += n
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return total, nil
}

const expireRelayWork = `UPDATE outbox SET state='failed',result='{"reason":"expired"}',next_attempt_at=NULL,
	lease_owner=NULL,lease_until=NULL,work_version=work_version+1
	WHERE id IN (SELECT id FROM outbox WHERE profile_id=? AND destination_id='relay:'||? AND relay_generation=?
	AND created_at<=? AND coalesce(lease_until,0)<=? AND `

const staleRelayUpdate = `UPDATE outbox INDEXED BY sqlite_autoindex_outbox_1 SET state='failed',result='{"reason":"destination_changed"}',
	next_attempt_at=NULL,lease_owner=NULL,lease_until=NULL,work_version=work_version+1 WHERE `

// invalidateStaleRelayItems materializes old-generation failures in bounded groups.
func invalidateStaleRelayItems(ctx context.Context, tx *sql.Conn, profile int32, generation int64) error {
	for _, scope := range []string{relayDue, relayCapacity} {
		_, err := tx.ExecContext(ctx, staleRelayUpdate+`id IN (SELECT id FROM outbox WHERE profile_id=? AND destination_id='relay:'||?
			AND relay_generation<? AND `+scope+` LIMIT 100)`, profile, profile, generation)
		if err != nil {
			return err
		}
	}
	return nil
}

// maintainRelayQueue expires bounded work and readmits the oldest capacity failures.
func maintainRelayQueue(ctx context.Context, tx *sql.Conn, profile int32, generation, now int64) error {
	if err := invalidateStaleRelayItems(ctx, tx, profile, generation); err != nil {
		return err
	}
	cutoff := now - int64(relayMaxAge/time.Second)
	// Check only the due window here. Periodic expiry also covers paused destinations.
	_, err := tx.ExecContext(ctx, `UPDATE outbox SET state='failed',result='{"reason":"expired"}',
		next_attempt_at=NULL,lease_owner=NULL,lease_until=NULL WHERE created_at<=?
		AND id IN (SELECT id FROM outbox WHERE profile_id=? AND destination_id='relay:'||? AND relay_generation=? AND `+relayDue+`
		AND next_attempt_at<=? AND coalesce(lease_until,0)<=? ORDER BY next_attempt_at,id LIMIT 100)`, cutoff, profile, profile, generation, now, now)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, expireRelayWork+relayCapacity+" LIMIT 100)", profile, profile, generation, cutoff, now)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE outbox SET state='pending',result='{}',next_attempt_at=?
		WHERE id IN (SELECT id FROM outbox WHERE profile_id=? AND destination_id='relay:'||? AND relay_generation=? AND `+relayCapacity+`
		AND created_at>? ORDER BY created_at,id LIMIT (SELECT min(100,100000-open_count) FROM relay_destinations WHERE profile_id=?))`, now, profile, profile, generation, cutoff, profile)
	return err
}

// RetryRelayItem explicitly readmits a failed item against the current destination.
// Isolated 400/413 failures terminate immediately. The eight-attempt limit
// bounds manual retries and any future recoverable item failures.
func (s *Sqlite) RetryRelayItem(ctx context.Context, profile int32, id string, now time.Time) (bool, error) {
	result, err := s.db.ExecContext(ctx, `UPDATE outbox SET state='pending',result='{}',next_attempt_at=?,relay_generation=(SELECT generation FROM relay_destinations WHERE profile_id=outbox.profile_id),
		lease_owner=NULL,lease_until=NULL,work_version=work_version+1
		WHERE profile_id=? AND id=? AND destination_id='relay:'||? AND format_version>0
		  AND (state='failed' OR (state IN ('pending','retryable') AND relay_generation<>(SELECT generation FROM relay_destinations WHERE profile_id=outbox.profile_id)))
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

const relayBatchScope = ` WHERE id IN (SELECT value FROM json_each(?)) AND profile_id=? AND destination_id='relay:'||?
	AND relay_generation=? AND lease_owner=? AND ` + relayDue
const renewRelayItems = `UPDATE outbox INDEXED BY sqlite_autoindex_outbox_1 SET lease_until=?` + relayBatchScope
const releaseRelayItems = `UPDATE outbox INDEXED BY sqlite_autoindex_outbox_1 SET state='retryable',next_attempt_at=?,lease_owner=NULL,lease_until=NULL` + relayBatchScope

// relayBatchIDs encodes the finite primary-key set for bounded writes under the lease.
func relayBatchIDs(batch RelayBatch) (string, error) {
	ids := make([]string, len(batch.Items))
	for i, item := range batch.Items {
		ids[i] = item.ID
	}
	body, err := json.Marshal(ids)
	return string(body), err
}

// invalidateRelayBatch marks only claimed rows when the destination generation has changed.
func invalidateRelayBatch(ctx context.Context, tx *sql.Conn, batch RelayBatch, ids string) error {
	_, err := tx.ExecContext(ctx, staleRelayUpdate+
		`id IN (SELECT value FROM json_each(?)) AND profile_id=? AND destination_id='relay:'||?
		AND relay_generation=? AND lease_owner=? AND `+relayDue+
		` AND EXISTS(SELECT 1 FROM relay_destinations WHERE profile_id=? AND generation>?)`,
		ids, batch.Profile, batch.Profile, batch.Generation, batch.Owner, batch.Profile, batch.Generation)
	return err
}

// RenewRelayBatch extends ownership before a POST. Changed credentials require a new claim.
func (s *Sqlite) RenewRelayBatch(ctx context.Context, batch RelayBatch, now time.Time) (bool, error) {
	ids, err := relayBatchIDs(batch)
	if err != nil {
		return false, err
	}
	var renewed bool
	err = s.withIngestWriter(ctx, func(tx *sql.Conn) error {
		until := now.Add(relayLease).Unix()
		res, err := tx.ExecContext(ctx, `UPDATE relay_destinations SET lease_until=? WHERE profile_id=? AND generation=? AND config_version=?
			AND work_version=? AND lease_owner=?`, until, batch.Profile, batch.Generation, batch.ConfigVersion, batch.Version, batch.Owner)
		if err != nil {
			return err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if n == 0 {
			return invalidateRelayBatch(ctx, tx, batch, ids)
		}
		// Items already finalized no longer carry the claim owner.
		res, err = tx.ExecContext(ctx, renewRelayItems, until, ids, batch.Profile, batch.Profile, batch.Generation, batch.Owner)
		if err != nil {
			return err
		}
		n, err = res.RowsAffected()
		renewed = n > 0
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
	ids, err := relayBatchIDs(batch)
	if err != nil {
		return false, err
	}
	finished := false
	err = s.withIngestWriter(ctx, func(tx *sql.Conn) error {
		var version, failures int64
		err := tx.QueryRowContext(ctx, `SELECT config_version,backoff_count FROM relay_destinations
			WHERE profile_id=? AND generation=? AND work_version=? AND lease_owner=?`,
			batch.Profile, batch.Generation, batch.Version, batch.Owner).Scan(&version, &failures)
		if errors.Is(err, sql.ErrNoRows) {
			// URL changes preserve the physical lease until the old attempt returns.
			if err := invalidateRelayBatch(ctx, tx, batch, ids); err != nil {
				return err
			}
			return releaseRelayDestination(ctx, tx, batch)
		}
		if err != nil {
			return err
		}
		finished = true
		if !canceled && result.Class != "" {
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
		_, err = tx.ExecContext(ctx, releaseRelayItems, now.Unix(), ids, batch.Profile, batch.Profile, batch.Generation, batch.Owner)
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
			WHERE profile_id=? AND id=? AND work_version=? AND lease_owner=? AND relay_generation=? AND `+relayDue,
			string(data), at, batch.Profile, item.ID, item.Version, batch.Owner, batch.Generation)
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
		WHERE profile_id=? AND id=? AND work_version=? AND lease_owner=? AND relay_generation=? AND `+relayDue,
		state, closed, string(data), increment, at, batch.Profile, item.ID, item.Version, batch.Owner, batch.Generation)
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
	case relay.Success:
		if result.RemainingZero {
			_, err := tx.ExecContext(ctx, "UPDATE relay_destinations SET pause_class='rate_limited',paused_until=?,backoff_count=0 WHERE profile_id=?", now.Add(result.ServerWait).Unix(), batch.Profile)
			return true, err
		}
		_, err := tx.ExecContext(ctx, "UPDATE relay_destinations SET pause_class=NULL,paused_until=NULL,backoff_count=0 WHERE profile_id=?", batch.Profile)
		return false, err
	case relay.BadRequest:
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
