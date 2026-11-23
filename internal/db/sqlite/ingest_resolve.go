package sqlite

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/ebrgr/TuneTrail/internal/mbz"
	"github.com/google/uuid"
)

const (
	activeRuleVersion       = "active-v1"
	ingestLease             = 120 * time.Second
	ingestRetryWriteTimeout = 5 * time.Second
	ingestAttemptTimeout    = ingestLease - 20*time.Second
)

type claimedReceipt struct {
	ignoredMBIDConflicts                                       int
	ignoredAliasConflicts                                      int
	id, lease, client, source, variant, format, deliveryPolicy string
	profile                                                    int32
	version, work, unix, attempts                              int64
	snapshot                                                   ingest.Snapshot
}

// ResolveNextIngest claims one active receipt independently of the admission flags.
func (s *Sqlite) ResolveNextIngest(ctx context.Context, mbzc mbz.MusicBrainzCaller) (bool, error) {
	// Keep the resolution deadline shorter than the claim lease.
	return s.resolveNextIngest(ctx, mbzc, ingestAttemptTimeout)
}

// resolveNextIngest keeps the resolution deadline separate from retry persistence.
func (s *Sqlite) resolveNextIngest(ctx context.Context, mbzc mbz.MusicBrainzCaller, timeout time.Duration) (bool, error) {
	receipt, snapshot, instant, err := s.claimNextIngest(ctx)
	if err != nil || receipt.id == "" {
		return false, err
	}
	// A parked receipt counts as progress but has no lease to resolve.
	if receipt.lease == "" {
		return true, nil
	}
	attemptCtx, cancel := context.WithTimeout(ctx, timeout)
	err = s.resolveIngestClaim(attemptCtx, receipt, snapshot, instant, mbzc)
	cancel()
	if err == nil {
		return true, nil
	}
	// A canceled worker leaves the lease for recovery without consuming a retry.
	if ctx.Err() != nil {
		return true, err
	}
	retryCtx, cancelRetry := context.WithTimeout(ctx, ingestRetryWriteTimeout)
	defer cancelRetry()
	return true, s.retryIngest(retryCtx, receipt, err)
}

// resolveIngestClaim uses the attempt context through the decision commit.
func (s *Sqlite) resolveIngestClaim(ctx context.Context, receipt claimedReceipt, snapshot, instant string, mbzc mbz.MusicBrainzCaller) error {
	at, err := time.Parse(receiptTime, instant)
	if err != nil {
		return s.rejectIngest(ctx, receipt, "invalid_listened_at")
	}
	receipt.unix = at.Unix()
	// Decode as an object so a JSON null is rejected as well as syntax and type errors.
	var decoded *ingest.Snapshot
	if err := json.Unmarshal([]byte(snapshot), &decoded); err != nil || decoded == nil {
		return s.rejectIngest(ctx, receipt, "invalid_snapshot")
	}
	receipt.snapshot = *decoded
	receipt.client = receipt.snapshot.Client
	if receipt.variant != "occurrence" {
		return s.rejectIngest(ctx, receipt, "unsupported_resolver_variant")
	}
	// Normalize comparisons without changing the persisted observations.
	if strings.TrimSpace(receipt.snapshot.Release) == "" {
		receipt.snapshot.Release = ""
	}
	if track, err := uuid.Parse(receipt.snapshot.TrackMBID); err == nil && track != uuid.Nil {
		receipt.snapshot.TrackMBID = track.String()
	}
	if receipt.source == ingest.ShadowSourceFile {
		// File declarations can bind catalog IDs without promoting them to verified proof.
		release := uuidFromSnapshot(receipt.snapshot.ReleaseMBID, receipt.snapshot.MappedReleaseMBID)
		proof := occurrenceProof{fileDeclared: true}
		if release != uuid.Nil {
			proof.release = release.String()
		}
		return s.finishOccurrence(ctx, receipt, proof)
	}
	// Finish remote verification well before the 120-second claim expires.
	proofCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	proof := verifyOccurrence(proofCtx, receipt.snapshot, mbzc, &s.occurrenceProofCache)
	err = proofCtx.Err()
	cancel()
	if err != nil {
		return err
	}
	return s.finishOccurrence(ctx, receipt, proof)
}

// claimNextIngest reads a due receipt and atomically reserves its lease.
func (s *Sqlite) claimNextIngest(ctx context.Context) (claimedReceipt, string, string, error) {
	clock := s.ingestNow()
	now := clock.Unix()
	var receipt claimedReceipt
	var snapshot, instant string
	const nextReceipt = `SELECT receipt_id FROM ingest_events INDEXED BY idx_ingest_resolver_queue
		WHERE mode='active' AND closed_at IS NULL
		AND parked_at IS NULL AND state IN ('received','retryable') AND coalesce(next_attempt_at,0)<=? AND coalesce(lease_until,0)<=?
		ORDER BY received_at,receipt_id LIMIT 1`
	// Preserve FIFO among due receipts without scanning parked rows or taking an idle writer lock.
	var next string
	err := s.db.QueryRowContext(ctx, nextReceipt, now, now).Scan(&next)
	if errors.Is(err, sql.ErrNoRows) {
		return claimedReceipt{}, "", "", nil
	}
	if err != nil {
		return claimedReceipt{}, "", "", err
	}
	// Decide under the update's writer lock. A due receipt at the cap parks without issuing another claim.
	err = s.db.QueryRowContext(ctx, `WITH candidate AS MATERIALIZED (
		SELECT receipt_id, attempt_count-claim_attempt_base>=? AS exhausted FROM ingest_events
		WHERE receipt_id=? AND mode='active' AND closed_at IS NULL AND parked_at IS NULL
		AND state IN ('received','retryable') AND coalesce(next_attempt_at,0)<=? AND coalesce(lease_until,0)<=?
	)
	UPDATE ingest_events SET
		lease_owner=CASE WHEN exhausted THEN NULL ELSE ? END,
		lease_until=CASE WHEN exhausted THEN NULL ELSE ? END,
		work_version=work_version+1,
		attempt_count=attempt_count+NOT exhausted,
		retry_attempt_base=retry_attempt_base+
			CASE WHEN NOT exhausted AND lease_until IS NOT NULL AND attempt_count>retry_attempt_base THEN 1 ELSE 0 END,
		state=CASE WHEN exhausted THEN 'retryable' ELSE state END,
		next_attempt_at=CASE WHEN exhausted THEN NULL ELSE next_attempt_at END,
		parked_at=CASE WHEN exhausted THEN ? END,
		park_reason=CASE WHEN exhausted THEN ? END
	FROM candidate WHERE ingest_events.receipt_id=candidate.receipt_id
	RETURNING receipt_id,profile_id,resolution_version,work_version,listened_at,original_snapshot,source_type,
		resolver_variant,coalesce(file_format,''),attempt_count-retry_attempt_base,delivery_policy,coalesce(lease_owner,'')`,
		maxIngestClaims, next, now, now, uuid.NewString(), now+int64(ingestLease/time.Second), now, ingestLeaseExhausted).
		Scan(&receipt.id, &receipt.profile, &receipt.version, &receipt.work, &instant, &snapshot, &receipt.source,
			&receipt.variant, &receipt.format, &receipt.attempts, &receipt.deliveryPolicy, &receipt.lease)
	if errors.Is(err, sql.ErrNoRows) {
		return claimedReceipt{}, "", "", nil
	}
	if err != nil {
		return claimedReceipt{}, "", "", err
	}
	if receipt.lease == "" {
		s.warnParkedIngest(ctx, ingestAlertKey{profile: receipt.profile, reason: ingestLeaseExhausted}, clock)
	}
	return receipt, snapshot, instant, nil
}

// uuidFromSnapshot uses the first valid identifier, with declared values first.
func uuidFromSnapshot(values ...string) uuid.UUID {
	for _, raw := range values {
		if id, err := uuid.Parse(raw); err == nil {
			return id
		}
	}
	return uuid.Nil
}

// withIngestWriter takes the SQLite writer lock before any transactional reads.
func (s *Sqlite) withIngestWriter(ctx context.Context, write func(*sql.Conn) error) error {
	tx, err := s.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer tx.Close()
	if _, err := tx.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			if _, err := tx.ExecContext(context.WithoutCancel(ctx), "ROLLBACK"); err != nil && !isAlreadyRolledBack(err) {
				_ = tx.Raw(func(any) error { return driver.ErrBadConn })
			}
		}
	}()
	if err := write(tx); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "COMMIT"); err != nil {
		return &ingest.ErrUnavailable{Cause: err}
	}
	committed = true
	return nil
}
