package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

var errIngestTrackMissing = errors.New("ingest track no longer exists")

// finishIngest rechecks the claim and legacy identity under the same writer reservation as the decision.
func (s *Sqlite) finishIngest(ctx context.Context, r claimedReceipt, track int32) error {
	return s.withIngestWriter(ctx, func(tx *sql.Conn) error {
		var recording sql.NullInt64
		var review bool
		var precision, admissionReason string
		err := tx.QueryRowContext(ctx, `SELECT o.recording_id,coalesce(f.review,0),e.time_precision,coalesce(json_extract(e.reason_codes,'$[0]'),'')
			FROM ingest_events e LEFT JOIN occurrences o ON o.id=? AND o.legacy_track_id=o.id
			LEFT JOIN profile_ingest_flags f ON f.profile_id=e.profile_id
			WHERE e.receipt_id=? AND e.profile_id=? AND e.mode='active' AND e.closed_at IS NULL
			AND e.resolution_version=? AND e.work_version=? AND e.lease_owner=? AND e.lease_until>?`,
			track, r.id, r.profile, r.version, r.work, r.lease, s.ingestNow().Unix()).Scan(&recording, &review, &precision, &admissionReason)
		if err != nil {
			return err
		}
		if !recording.Valid {
			return errIngestTrackMissing
		}
		// Multiple explicitly separate listens can share a track and second.
		var existing, candidate string
		err = tx.QueryRowContext(ctx, `SELECT event_id FROM listens WHERE profile_id=? AND track_id=? AND listened_at=? ORDER BY (origin='legacy') DESC,event_id LIMIT 1`,
			r.profile, track, r.unix).Scan(&existing)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		candidate = existing
		if candidate == "" {
			err = tx.QueryRowContext(ctx, `SELECT l.event_id FROM listens l JOIN occurrences o ON o.id=l.track_id
				WHERE l.profile_id=? AND l.listened_at BETWEEN ? AND ?
				AND (l.track_id=? OR o.recording_id=?)
				AND NOT EXISTS (
					SELECT 1 FROM ingest_events prior JOIN ingest_events incoming ON incoming.receipt_id=?
					WHERE prior.profile_id=l.profile_id AND prior.event_id=l.event_id AND prior.mode='active'
					AND prior.state='accepted' AND prior.source_instance_id=incoming.source_instance_id
					AND (
						(prior.external_id_trusted=1 AND incoming.external_id_trusted=1
						AND prior.external_namespace=incoming.external_namespace AND prior.external_event_id<>incoming.external_event_id)
						OR (prior.file_digest=incoming.file_digest AND prior.file_format=incoming.file_format
						AND prior.format_version=incoming.format_version AND prior.item_ordinal<>incoming.item_ordinal)
					)
				) ORDER BY l.listened_at,l.event_id LIMIT 1`,
				r.profile, r.unix-2, r.unix+2, track, recording.Int64, r.id).Scan(&candidate)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return err
			}
		}
		return finalizeActiveReceipt(ctx, tx, r, track, recording.Int64, precision, review, existing, candidate, admissionReason)
	})
}

// finalizeActiveReceipt writes the decision, revision, listen and eligible relay intent atomically.
func finalizeActiveReceipt(ctx context.Context, tx *sql.Conn, r claimedReceipt, track int32, recording int64, precision string, review bool, existing, candidate, admissionReason string) error {
	now := time.Now().Unix()
	state, decision, reason := "accepted", "different", "no_candidate"
	var event, closed any
	switch {
	case candidate != "" && review:
		state, decision, reason = "pending_duplicate", "review", "ambiguous_time_candidate"
		if existing != "" {
			reason = "legacy_identity_collision"
		}
	case existing != "":
		state, decision, reason = "duplicate", "same", "review_deferred"
		event, closed = existing, now
	default:
		if candidate != "" || admissionReason == "review_deferred" {
			reason = "review_deferred"
		}
		event, closed = uuid.NewString(), now
		if _, err := tx.ExecContext(ctx, `INSERT INTO listens(profile_id,track_id,listened_at,client,event_id) VALUES(?,?,?,?,?)`,
			r.profile, track, r.unix, r.client, event); err != nil {
			return err
		}
	}
	reasons, _ := json.Marshal([]string{reason})
	evidence, _ := json.Marshal(map[string]string{"candidate_event_id": candidate, "precision": precision})
	result, err := tx.ExecContext(ctx, `UPDATE ingest_events SET state=?,identity_status='resolved',duplicate_decision=?,reason_codes=?,evidence=?,
		rule_version=?,resolution_version=resolution_version+1,resolved_at=?,event_id=?,occurrence_id=?,recording_id=?,
		closed_at=?,next_attempt_at=NULL,lease_owner=NULL,lease_until=NULL
		WHERE receipt_id=? AND profile_id=? AND mode='active' AND resolution_version=? AND work_version=? AND lease_owner=?`,
		state, decision, string(reasons), string(evidence), activeRuleVersion, now, event, track, recording, closed,
		r.id, r.profile, r.version, r.work, r.lease)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return errors.New("ingest claim changed")
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO ingest_revisions VALUES(?,?,?,'automatic',?,?,?,?,?)`,
		r.profile, r.id, r.version+1, decision, string(reasons), string(evidence), activeRuleVersion, now); err != nil {
		return err
	}
	if state != "accepted" {
		return nil
	}
	return insertRelayIntent(ctx, tx, r, event.(string), now)
}
