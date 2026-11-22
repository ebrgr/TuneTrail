package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
)

const ingestDiscardBatchSize = 25

// Parked receipts retain their underlying received or retryable state.
// Outbox intentions are keyed by (profile_id,event_id). Requiring a NULL
// event_id excludes every linked intention, including open relay deliveries.
const ingestDiscardScope = `event_id IS NULL
	AND state IN ('received','retryable','pending_identity','pending_duplicate','invalid')
	AND (lease_until IS NULL OR lease_until<=?)`

type ingestDiscardCandidate struct {
	Receipt, State, ParkReason string
	Resolution, Work           int64
	Parked                     sql.NullInt64
}

type ingestDiscardRow struct {
	candidate ingestDiscardCandidate
	display   db.IngestDiscardReceipt
}

// ingestDiscardRows reads only versions and safe display fields for one profile.
func ingestDiscardRows(ctx context.Context, tx ingestActivationReader, profile int32, now int64, receipt string, limit, offset int) ([]ingestDiscardRow, error) {
	filter := ""
	args := []any{profile, now}
	if receipt != "" {
		filter = " AND receipt_id=?"
		args = append(args, receipt)
	}
	args = append(args, limit, offset)
	rows, err := tx.QueryContext(ctx, `SELECT receipt_id,state,resolution_version,work_version,parked_at,
		coalesce(park_reason,''),source_type,coalesce(json_extract(reason_codes,'$[0]'),''),received_at
		FROM ingest_events WHERE profile_id=? AND `+ingestDiscardScope+filter+`
		ORDER BY received_at,receipt_id LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []ingestDiscardRow
	for rows.Next() {
		var row ingestDiscardRow
		c, d := &row.candidate, &row.display
		var received int64
		if err := rows.Scan(&c.Receipt, &c.State, &c.Resolution, &c.Work, &c.Parked, &c.ParkReason, &d.Source, &d.Reason, &received); err != nil {
			return nil, err
		}
		d.Receipt, d.State, d.ParkReason, d.Received = c.Receipt, c.State, c.ParkReason, time.Unix(received, 0).UTC()
		result = append(result, row)
	}
	return result, rows.Err()
}

// ingestDiscardDigest binds a confirmation to its profile, action and exact ordered versions.
func ingestDiscardDigest(profile int32, receipt string, rows []ingestDiscardRow) (string, error) {
	if len(rows) == 0 {
		return "", nil
	}
	input := struct {
		Profile int32
		Receipt string
		Rows    []ingestDiscardCandidate
	}{Profile: profile, Receipt: receipt, Rows: make([]ingestDiscardCandidate, 0, len(rows))}
	for _, row := range rows {
		input.Rows = append(input.Rows, row.candidate)
	}
	encoded, err := json.Marshal(input)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

// IngestDiscardPreview lists a page and binds the bulk action to the oldest eligible receipts.
func (s *Sqlite) IngestDiscardPreview(ctx context.Context, profile int32, receipt string, page int) (db.IngestDiscardPreview, error) {
	preview := db.IngestDiscardPreview{Page: page}
	if profile <= 0 || page < 1 || page > (1<<31-1)/ingestDiscardBatchSize {
		return preview, db.ErrInvalidInput
	}
	now := s.ingestNow().Unix()
	rows, err := ingestDiscardRows(ctx, s.db, profile, now, receipt, ingestDiscardBatchSize+1, (page-1)*ingestDiscardBatchSize)
	if err != nil {
		return preview, err
	}
	if receipt != "" && len(rows) == 0 {
		return preview, checkIngestDiscardTarget(ctx, s.db, profile, receipt)
	}
	preview.More = len(rows) > ingestDiscardBatchSize
	if preview.More {
		preview.Next = page + 1
		rows = rows[:ingestDiscardBatchSize]
	}
	for i, row := range rows {
		row.display.Digest, err = ingestDiscardDigest(profile, row.candidate.Receipt, rows[i:i+1])
		if err != nil {
			return preview, err
		}
		preview.Receipts = append(preview.Receipts, row.display)
	}
	if receipt == "" && page == 1 {
		preview.Batch.Count = len(rows)
		preview.Batch.Digest, err = ingestDiscardDigest(profile, "", rows)
	}
	return preview, err
}

// checkIngestDiscardTarget distinguishes linked listens and an idempotent replay from a stale target.
func checkIngestDiscardTarget(ctx context.Context, reader ingestActivationReader, profile int32, receipt string) error {
	var protected, discarded bool
	err := reader.QueryRowContext(ctx, `SELECT event_id IS NOT NULL OR state='accepted',0
		FROM ingest_events WHERE profile_id=? AND receipt_id=?
		UNION ALL SELECT 0,1 FROM ingest_discards WHERE profile_id=? AND receipt_id=? LIMIT 1`,
		profile, receipt, profile, receipt).Scan(&protected, &discarded)
	if errors.Is(err, sql.ErrNoRows) {
		return db.ErrNotFound
	}
	if err != nil {
		return err
	}
	if protected {
		return db.ErrIngestDiscardListen
	}
	if discarded {
		return nil
	}
	return db.ErrConflict
}

// DiscardIngestPending deletes a confirmed receipt or a bounded batch in one writer transaction.
func (s *Sqlite) DiscardIngestPending(ctx context.Context, profile, actor int32, receipt, digest string) error {
	if profile <= 0 || actor <= 0 || len(digest) != sha256.Size*2 {
		return db.ErrInvalidInput
	}
	return s.withIngestWriter(ctx, func(tx *sql.Conn) error {
		now := s.ingestNow().Unix()
		rows, err := ingestDiscardRows(ctx, tx, profile, now, receipt, ingestDiscardBatchSize, 0)
		if err != nil {
			return err
		}
		if receipt != "" && len(rows) == 0 {
			return checkIngestDiscardTarget(ctx, tx, profile, receipt)
		}
		expected, err := ingestDiscardDigest(profile, receipt, rows)
		if err != nil {
			return err
		}
		if expected != digest {
			return db.ErrConflict
		}
		for _, row := range rows {
			if err := discardIngestRow(ctx, tx, profile, actor, now, row.candidate); err != nil {
				return err
			}
		}
		return nil
	})
}

// discardIngestRow copies only audit metadata, then deletes using the same CAS predicate.
func discardIngestRow(ctx context.Context, tx *sql.Conn, profile, actor int32, now int64, c ingestDiscardCandidate) error {
	where := ` WHERE profile_id=? AND receipt_id=? AND resolution_version=? AND work_version=? AND state=? AND parked_at IS ? AND ` + ingestDiscardScope
	args := []any{profile, c.Receipt, c.Resolution, c.Work, c.State, c.Parked, now}
	if _, err := tx.ExecContext(ctx, `INSERT INTO ingest_discards
		(profile_id,receipt_id,actor_id,discarded_at,origin_state,park_reason,reason_codes)
		SELECT profile_id,receipt_id,?,?,state,park_reason,reason_codes FROM ingest_events`+where,
		append([]any{actor, now}, args...)...); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `DELETE FROM ingest_events`+where, args...)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err == nil && count != 1 {
		return db.ErrConflict
	}
	return err
}
