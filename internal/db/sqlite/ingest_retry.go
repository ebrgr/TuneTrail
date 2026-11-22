package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/ebrgr/TuneTrail/internal/logger"
)

const (
	maxIngestAttempts    = 5
	maxIngestClaims      = 5 * maxIngestAttempts
	ingestLeaseExhausted = "lease_exhausted"
	ingestRetryExhausted = "retry_exhausted"
)

type ingestAlertKey struct {
	profile int32
	reason  string
}

// ingestNow gives the scheduler and lease checks a common clock.
func (s *Sqlite) ingestNow() time.Time {
	if s.ingestClock != nil {
		return s.ingestClock()
	}
	return time.Now()
}

// retryIngest reschedules or parks this claim in the same update that releases its lease.
func (s *Sqlite) retryIngest(ctx context.Context, r claimedReceipt, cause error) error {
	now := s.ingestNow()
	var next, parked, reason any
	if r.attempts >= maxIngestAttempts {
		parked, reason = now.Unix(), ingestRetryExhausted
	} else {
		next = now.Add(time.Minute << (r.attempts - 1)).Unix()
	}
	result, err := s.db.ExecContext(ctx, `UPDATE ingest_events SET state='retryable',next_attempt_at=?,parked_at=?,park_reason=?,
		lease_owner=NULL,lease_until=NULL WHERE receipt_id=? AND profile_id=? AND mode='active' AND closed_at IS NULL
		AND resolution_version=? AND work_version=? AND lease_owner=? AND lease_until>?`,
		next, parked, reason, r.id, r.profile, r.version, r.work, r.lease, now.Unix())
	if err != nil {
		return errors.Join(cause, err)
	}
	changed, err := result.RowsAffected()
	if err == nil && changed == 1 && parked != nil {
		s.warnParkedIngest(ctx, ingestAlertKey{profile: r.profile, reason: ingestRetryExhausted}, now)
	}
	return errors.Join(cause, err)
}

// warnParkedIngest coalesces warnings per profile and reason within a one-minute window.
func (s *Sqlite) warnParkedIngest(ctx context.Context, key ingestAlertKey, now time.Time) {
	s.ingestAlertMu.Lock()
	for old, until := range s.ingestAlerts {
		if !now.Before(until) {
			delete(s.ingestAlerts, old)
		}
	}
	if _, exists := s.ingestAlerts[key]; exists {
		s.ingestAlertMu.Unlock()
		return
	}
	if s.ingestAlerts == nil {
		s.ingestAlerts = make(map[ingestAlertKey]time.Time)
	}
	s.ingestAlerts[key] = now.Add(time.Minute)
	s.ingestAlertMu.Unlock()
	logger.FromContext(ctx).Warn().Int32("profile_id", key.profile).Str("reason", key.reason).Msg("Ingest receipt parked")
}

// rejectIngest closes deterministic input errors and records the decision atomically.
func (s *Sqlite) rejectIngest(ctx context.Context, r claimedReceipt, reason string) error {
	err := s.withIngestWriter(ctx, func(tx *sql.Conn) error {
		now := s.ingestNow().Unix()
		result, err := tx.ExecContext(ctx, `UPDATE ingest_events SET state='invalid',identity_status='invalid',
			duplicate_decision=NULL,reason_codes=json_array(?),rule_version=?,resolved_at=?,closed_at=?,
			resolution_version=resolution_version+1,next_attempt_at=NULL,lease_owner=NULL,lease_until=NULL
			WHERE receipt_id=? AND profile_id=? AND mode='active' AND closed_at IS NULL
			AND resolution_version=? AND work_version=? AND lease_owner=? AND lease_until>?`,
			reason, activeRuleVersion, now, now, r.id, r.profile, r.version, r.work, r.lease, now)
		if err != nil {
			return err
		}
		changed, err := result.RowsAffected()
		if err != nil || changed == 0 {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO ingest_revisions VALUES(?,?,?,'automatic','invalid',json_array(?),'{}',?,?)`,
			r.profile, r.id, r.version+1, reason, activeRuleVersion, now)
		return err
	})
	if err != nil {
		// A failed decision write is a storage failure and follows the ordinary retry policy.
		return s.retryIngest(ctx, r, err)
	}
	return nil
}

// requeueParkedIngest resets both budgets and records the latest operator atomically.
const requeueParkedIngest = `UPDATE ingest_events SET next_attempt_at=?,parked_at=NULL,park_reason=NULL,
	retry_attempt_base=attempt_count,claim_attempt_base=attempt_count,work_version=work_version+1,
	reprocessed_at=?,reprocessed_by=? WHERE profile_id=? AND parked_at IS NOT NULL`

// ReprocessIngest grants a fresh budget to one parked receipt in the authorized profile.
func (s *Sqlite) ReprocessIngest(ctx context.Context, profile, actor int32, receipt string) error {
	// The parked predicate makes a replay or a receipt from another profile a no-op.
	now := s.ingestNow().Unix()
	_, err := s.db.ExecContext(ctx, requeueParkedIngest+` AND receipt_id=?`, now, now, actor, profile, receipt)
	return err
}

// ReprocessAllIngest grants a fresh budget to every parked receipt in the authorized profile.
func (s *Sqlite) ReprocessAllIngest(ctx context.Context, profile, actor int32) error {
	const batchSize = 500
	return s.withIngestWriter(ctx, func(tx *sql.Conn) error {
		now := s.ingestNow().Unix()
		// Bound each statement while keeping the operation atomic with respect to claims and replays.
		for {
			result, err := tx.ExecContext(ctx, `WITH batch AS MATERIALIZED (
				SELECT receipt_id FROM ingest_events WHERE profile_id=? AND parked_at IS NOT NULL
				ORDER BY parked_at DESC,receipt_id DESC LIMIT ?
			) `+requeueParkedIngest+` AND receipt_id IN (SELECT receipt_id FROM batch)`,
				profile, batchSize, now, now, actor, profile)
			if err != nil {
				return err
			}
			changed, err := result.RowsAffected()
			if err != nil || changed < batchSize {
				return err
			}
		}
	})
}
