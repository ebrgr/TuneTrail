package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
)

const ingestReviewBatchSize = 25

const reviewSingleCandidate = `CASE
	WHEN json_type(e.evidence,'$.candidates')='array' THEN
		CASE WHEN json_array_length(e.evidence,'$.candidates')=1 AND json_extract(e.evidence,'$.truncated') IS NOT 1
			AND substr(json_extract(e.evidence,'$.candidates[0]'),1,6)='event:'
			THEN substr(json_extract(e.evidence,'$.candidates[0]'),7) END
	WHEN json_type(e.evidence,'$.candidate_event_id')='text' THEN json_extract(e.evidence,'$.candidate_event_id')
	END`

// ingestReviewBatch selects the oldest eligible receipts and binds the action to those exact versions.
func ingestReviewBatch(ctx context.Context, tx ingestQueryer, profile int32, value string) ([]db.IngestReviewDecision, string, error) {
	query := `SELECT e.receipt_id,e.resolution_version,l.event_id
		FROM ingest_events e CROSS JOIN listens l ON l.profile_id=e.profile_id AND l.event_id=(` + reviewSingleCandidate + `)
		JOIN occurrences o ON o.id=l.track_id WHERE e.profile_id=? AND ` + reviewQueueScope + `
		ORDER BY e.received_at,e.receipt_id`
	if value == "different" {
		query = `SELECT e.receipt_id,e.resolution_version,e.listened_at
			FROM ingest_events e JOIN occurrences o ON o.id=e.occurrence_id
			WHERE e.profile_id=? AND ` + reviewQueueScope + ` AND e.listened_at IS NOT NULL
			ORDER BY e.received_at,e.receipt_id`
	}
	rows, err := tx.QueryContext(ctx, query, profile)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	var reviews []db.IngestReviewDecision
	for len(reviews) < ingestReviewBatchSize && rows.Next() {
		review := db.IngestReviewDecision{Value: value}
		var link string
		if err := rows.Scan(&review.Receipt, &review.Version, &link); err != nil {
			return nil, "", err
		}
		if value == "same" {
			review.Candidate = link
		} else if _, err := time.Parse(receiptTime, link); err != nil {
			continue
		}
		reviews = append(reviews, review)
	}
	if err := rows.Err(); err != nil || len(reviews) == 0 {
		return reviews, "", err
	}
	encoded, err := json.Marshal(reviews)
	if err != nil {
		return nil, "", err
	}
	digest := sha256.Sum256(encoded)
	return reviews, hex.EncodeToString(digest[:]), nil
}

// ResolveIngestReviewBatch applies a previewed batch once, under one writer reservation.
func (s *Sqlite) ResolveIngestReviewBatch(ctx context.Context, profile, actor int32, value, digest string) error {
	if actor <= 0 || len(digest) != sha256.Size*2 || (value != "same" && value != "different") {
		return db.ErrInvalidInput
	}
	return s.withIngestWriter(ctx, func(tx *sql.Conn) error {
		reviews, expected, err := ingestReviewBatch(ctx, tx, profile, value)
		if err != nil {
			return err
		}
		if expected != digest {
			return db.ErrConflict
		}
		for _, review := range reviews {
			if err := s.resolveIngestReview(ctx, tx, profile, actor, review); err != nil {
				return err
			}
		}
		return nil
	})
}
