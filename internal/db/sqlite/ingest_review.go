package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/google/uuid"
)

const reviewQueueScope = "mode='active' AND state='pending_duplicate' AND closed_at IS NULL"

const reviewPageSize = 25

const reviewCountSQL = `SELECT count(*) FROM ingest_events WHERE profile_id=? AND ` + reviewQueueScope

const reviewPageSQL = `SELECT e.receipt_id,coalesce(e.file_format,e.source_type),coalesce(o.presented_title,(SELECT alias FROM track_aliases WHERE track_id=o.legacy_track_id AND is_primary=1 LIMIT 1),''),
	coalesce(json_extract(e.reason_codes,'$[0]'),''),e.time_precision,coalesce(e.rule_version,''),
	e.received_at,e.listened_at,e.resolution_version,o.id IS NOT NULL
	FROM ingest_events e LEFT JOIN occurrences o ON o.id=e.occurrence_id
	WHERE e.profile_id=? AND ` + reviewQueueScope + `
	ORDER BY e.received_at DESC,e.receipt_id DESC LIMIT ? OFFSET ?`

// healthReviews reads the requested page and its candidates from one SQLite snapshot.
func (s *Sqlite) healthReviews(ctx context.Context, profile int32, page int, health *db.IngestHealth) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	flags, err := ingestActivationFlags(ctx, tx, profile)
	if err != nil {
		return err
	}
	health.ReviewAvailable, health.ReviewEnabled = flags.Active, flags.Review
	err = tx.QueryRowContext(ctx, reviewCountSQL, profile).Scan(&health.ReviewCount)
	if err != nil {
		return err
	}
	lastPage := max(1, (health.ReviewCount+reviewPageSize-1)/reviewPageSize)
	page = min(max(1, page), lastPage)
	health.ReviewPrevious = page - 1
	if page < lastPage {
		health.ReviewNext = page + 1
	}
	rows, err := tx.QueryContext(ctx, reviewPageSQL, profile, reviewPageSize, (page-1)*reviewPageSize)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var item db.IngestReview
		var received int64
		var listened sql.NullString
		if err := rows.Scan(&item.Receipt, &item.Source, &item.Title, &item.Reason, &item.Precision, &item.Rule,
			&received, &listened, &item.Version, &item.CanKeep); err != nil {
			return err
		}
		item.Received = time.Unix(received, 0).UTC()
		at, err := time.Parse(receiptTime, listened.String)
		if err == nil {
			item.ListenedAt = &at
		} else {
			item.CanKeep = false
		}
		health.Reviews = append(health.Reviews, item)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for i := range health.Reviews {
		item := &health.Reviews[i]
		item.Candidates, err = healthReviewCandidates(ctx, tx, profile, item.Receipt)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

// healthReviewCandidates projects catalog titles and timestamps without loading the stored evidence document.
func healthReviewCandidates(ctx context.Context, tx *sql.Tx, profile int32, receipt string) ([]db.IngestReviewCandidate, error) {
	rows, err := tx.QueryContext(ctx, `SELECT c.event_id,c.listened_at,
		coalesce(c.presented_title,(SELECT alias FROM track_aliases WHERE track_id=c.occurrence_id AND is_primary=1 LIMIT 1))
		FROM (`+reviewCandidateRows+`) c ORDER BY c.listened_at,c.event_id`, profile, receipt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var candidates []db.IngestReviewCandidate
	for rows.Next() {
		var item db.IngestReviewCandidate
		var stamp int64
		var title sql.NullString
		if err := rows.Scan(&item.Event, &stamp, &title); err != nil {
			return nil, err
		}
		item.Title, item.At = title.String, time.Unix(stamp, 0).UTC()
		candidates = append(candidates, item)
	}
	return candidates, rows.Err()
}

// reviewCandidateRows scopes evidence membership and listen ownership identically for display and writes.
// CROSS JOIN keeps the receipt and its small candidate array ahead of the listen lookup.
const reviewCandidateRows = `SELECT DISTINCT l.event_id,l.listened_at,o.id AS occurrence_id,o.recording_id,o.presented_title
	FROM ingest_events e CROSS JOIN json_each(CASE WHEN json_type(e.evidence,'$.candidates')='array'
		THEN json_extract(e.evidence,'$.candidates')
		ELSE json_array('event:'||nullif(json_extract(e.evidence,'$.candidate_event_id'),'')) END) candidate
	CROSS JOIN listens l ON l.profile_id=e.profile_id AND l.event_id=substr(candidate.value,7)
	JOIN occurrences o ON o.id=l.track_id
	WHERE e.profile_id=? AND e.receipt_id=? AND candidate.type='text' AND substr(candidate.value,1,6)='event:'`

// SetIngestReview changes future candidate handling without affecting activation or existing pendencies.
func (s *Sqlite) SetIngestReview(ctx context.Context, profile int32, enabled bool) error {
	return s.withIngestWriter(ctx, func(tx *sql.Conn) error {
		flags, err := ingestActivationFlags(ctx, tx, profile)
		if err != nil {
			return err
		}
		if !flags.Active {
			return db.ErrConflict
		}
		_, err = tx.ExecContext(ctx, `UPDATE profile_ingest_flags SET review=? WHERE profile_id=?`, enabled, profile)
		return err
	})
}

// ResolveIngestReview records a human decision under the ingestion writer reservation.
func (s *Sqlite) ResolveIngestReview(ctx context.Context, profile int32, review db.IngestReviewDecision) error {
	if review.Receipt == "" || review.Version < 0 ||
		!(review.Value == "same" && review.Candidate != "" || review.Value == "different" && review.Candidate == "") {
		return db.ErrInvalidInput
	}
	return s.withIngestWriter(ctx, func(tx *sql.Conn) error {
		r := claimedReceipt{id: review.Receipt, profile: profile}
		var state, mode, rule, evidence string
		var closed, occurrence, recording sql.NullInt64
		var instant sql.NullString
		err := tx.QueryRowContext(ctx, `SELECT state,mode,closed_at,resolution_version,work_version,
			occurrence_id,recording_id,listened_at,coalesce(CASE WHEN json_type(original_snapshot,'$.client')='text' THEN json_extract(original_snapshot,'$.client') END,''),
			source_type,coalesce(file_format,''),resolver_variant,delivery_policy,coalesce(nullif(rule_version,''),'manual-review-v1'),evidence
			FROM ingest_events WHERE profile_id=? AND receipt_id=?`, profile, r.id).Scan(
			&state, &mode, &closed, &r.version, &r.work, &occurrence, &recording, &instant, &r.client,
			&r.source, &r.format, &r.variant, &r.deliveryPolicy, &rule, &evidence)
		if errors.Is(err, sql.ErrNoRows) || err == nil && mode != "active" {
			return db.ErrNotFound
		}
		if err != nil {
			return err
		}
		if state != "pending_duplicate" || closed.Valid || r.version != review.Version {
			return db.ErrConflict
		}

		now := s.ingestNow().Unix()
		event, finalState, reason := review.Candidate, "duplicate", "user_confirmed_duplicate"
		if review.Value == "same" {
			err := tx.QueryRowContext(ctx, `SELECT occurrence_id,recording_id FROM (`+reviewCandidateRows+`) WHERE event_id=?`,
				profile, r.id, event).Scan(&occurrence, &recording)
			if errors.Is(err, sql.ErrNoRows) {
				return db.ErrNotFound
			}
			if err != nil {
				return err
			}
		} else {
			at, err := time.Parse(receiptTime, instant.String)
			if err != nil || !occurrence.Valid {
				return db.ErrConflict
			}
			// Re-read the recording because catalog reconciliation may have changed it during review.
			err = tx.QueryRowContext(ctx, `SELECT recording_id FROM occurrences WHERE id=?`, occurrence).Scan(&recording)
			if errors.Is(err, sql.ErrNoRows) {
				return db.ErrConflict
			}
			if err != nil {
				return err
			}
			event, finalState, reason = uuid.NewString(), "accepted", "distinct_occurrence_event"
			origin, legacy := occurrenceOrigin(r), r.variant == "v9"
			if legacy {
				origin, reason = "legacy", "user_kept_separate"
			}
			// Exempt only an explicit repetition of an existing legacy tuple from the legacy unique index.
			if _, err := tx.ExecContext(ctx, `INSERT INTO listens(profile_id,track_id,listened_at,client,event_id,origin,review_distinct)
				VALUES(?,?,?,?,?,?,? AND EXISTS(SELECT 1 FROM listens WHERE profile_id=? AND track_id=? AND listened_at=?
					AND origin='legacy' AND review_distinct=0))`,
				profile, occurrence, at.Unix(), r.client, event, origin, legacy, profile, occurrence, at.Unix()); err != nil {
				return err
			}
		}
		// Both fences and the pending state must still match before writing a revision or relay intent.
		err = tx.QueryRowContext(ctx, `UPDATE ingest_events SET state=?,identity_status='resolved',duplicate_decision=?,
			reason_codes=json_array(?),resolution_version=resolution_version+1,work_version=work_version+1,
			resolved_at=?,closed_at=?,event_id=?,occurrence_id=?,recording_id=?,next_attempt_at=NULL,lease_owner=NULL,lease_until=NULL
			WHERE profile_id=? AND receipt_id=? AND resolution_version=? AND work_version=? AND `+reviewQueueScope+`
			RETURNING resolution_version`, finalState, review.Value, reason, now, now, event, occurrence, recording,
			profile, r.id, review.Version, r.work).Scan(&r.version)
		if errors.Is(err, sql.ErrNoRows) {
			return db.ErrConflict
		}
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO ingest_revisions VALUES(?,?,?,'user',?,json_array(?),json_set(?,'$.review_event_id',?),?,?)`,
			profile, r.id, r.version, review.Value, reason, evidence, event, rule, now); err != nil {
			return err
		}
		if finalState == "accepted" {
			return insertRelayIntent(ctx, tx, r, event, now)
		}
		return nil
	})
}
