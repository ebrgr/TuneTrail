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

const (
	ingestDiscardPageSize  = 25
	ingestDiscardBatchSize = 500
	// Match the default retention of operational ingest details.
	ingestDiscardRetention = 90 * 24 * time.Hour
)

// Parked receipts retain their underlying received or retryable state.
// Outbox intentions are keyed by (profile_id,event_id). Requiring a NULL
// event_id excludes every linked intention, including open relay deliveries.
const ingestDiscardScope = `event_id IS NULL
	AND state IN ('received','retryable','pending_identity','pending_duplicate','invalid')
	AND (lease_until IS NULL OR lease_until<=?)`

type ingestDiscardCandidate struct {
	Receipt, State, ParkReason string
	Resolution, Work           int64
	Received, LastDiscard      int64
	Parked                     sql.NullInt64
}

type ingestDiscardRow struct {
	candidate ingestDiscardCandidate
	display   db.IngestDiscardReceipt
}

// A trusted replay can reuse both the receipt ID and its versions, even within
// the same second. The previous discard ID distinguishes that new incarnation.
const ingestDiscardGeneration = `coalesce((SELECT id FROM ingest_discards d
	WHERE d.profile_id=ingest_events.profile_id AND d.receipt_id=ingest_events.receipt_id
	ORDER BY id DESC LIMIT 1),0)`

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
		coalesce(park_reason,''),received_at,`+ingestDiscardGeneration+`,source_type,coalesce(json_extract(reason_codes,'$[0]'),'')
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
		if err := rows.Scan(&c.Receipt, &c.State, &c.Resolution, &c.Work, &c.Parked, &c.ParkReason, &c.Received, &c.LastDiscard, &d.Source, &d.Reason); err != nil {
			return nil, err
		}
		d.Receipt, d.State, d.ParkReason, d.Received = c.Receipt, c.State, c.ParkReason, time.Unix(c.Received, 0).UTC()
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

// scanIngestDiscardSet visits the complete eligible set in stable index order.
func scanIngestDiscardSet(ctx context.Context, reader ingestActivationReader, profile int32, now int64, visit func(ingestDiscardCandidate) error) error {
	rows, err := reader.QueryContext(ctx, `SELECT receipt_id,state,resolution_version,work_version,parked_at,
		coalesce(park_reason,''),received_at,`+ingestDiscardGeneration+`
		FROM ingest_events WHERE profile_id=? AND `+ingestDiscardScope+`
		ORDER BY received_at,receipt_id`, profile, now)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var c ingestDiscardCandidate
		if err := rows.Scan(&c.Receipt, &c.State, &c.Resolution, &c.Work, &c.Parked, &c.ParkReason, &c.Received, &c.LastDiscard); err != nil {
			return err
		}
		if err := visit(c); err != nil {
			return err
		}
	}
	return rows.Err()
}

// ingestDiscardSetDigest hashes the ordered set without putting its IDs into the form.
func ingestDiscardSetDigest(ctx context.Context, reader ingestActivationReader, profile int32, now int64, capture *[]ingestDiscardCandidate) (db.IngestReviewBatch, error) {
	sum := sha256.New()
	encoder := json.NewEncoder(sum)
	preview := db.IngestReviewBatch{}
	if err := encoder.Encode(profile); err != nil {
		return preview, err
	}
	err := scanIngestDiscardSet(ctx, reader, profile, now, func(c ingestDiscardCandidate) error {
		if capture != nil {
			*capture = append(*capture, c)
		}
		preview.Count++
		return encoder.Encode(c)
	})
	if err == nil && preview.Count > 0 {
		preview.Digest = hex.EncodeToString(sum.Sum(nil))
	}
	return preview, err
}

// IngestDiscardPreview shows one page and binds the bulk action to the entire eligible set.
func (s *Sqlite) IngestDiscardPreview(ctx context.Context, profile int32, receipt string, page int) (db.IngestDiscardPreview, error) {
	preview := db.IngestDiscardPreview{Page: page}
	if profile <= 0 || page < 1 || page > (1<<31-1)/ingestDiscardPageSize {
		return preview, db.ErrInvalidInput
	}
	now := s.ingestNow().Unix()
	rows, err := ingestDiscardRows(ctx, s.db, profile, now, receipt, ingestDiscardPageSize+1, (page-1)*ingestDiscardPageSize)
	if err != nil {
		return preview, err
	}
	if receipt != "" && len(rows) == 0 {
		return preview, checkIngestDiscardTarget(ctx, s.db, profile, receipt)
	}
	preview.More = len(rows) > ingestDiscardPageSize
	if preview.More {
		preview.Next = page + 1
		rows = rows[:ingestDiscardPageSize]
	}
	for i, row := range rows {
		row.display.Digest, err = ingestDiscardDigest(profile, row.candidate.Receipt, rows[i:i+1])
		if err != nil {
			return preview, err
		}
		preview.Receipts = append(preview.Receipts, row.display)
	}
	if receipt == "" && page == 1 {
		preview.Batch, err = ingestDiscardSetDigest(ctx, s.db, profile, now, nil)
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

// DiscardIngestPending revalidates the confirmed action before deleting any receipt.
func (s *Sqlite) DiscardIngestPending(ctx context.Context, profile, actor int32, receipt, digest string) error {
	if profile <= 0 || actor <= 0 || len(digest) != sha256.Size*2 {
		return db.ErrInvalidInput
	}
	if receipt == "" {
		return s.discardIngestSet(ctx, profile, actor, digest)
	}
	return s.withIngestWriter(ctx, func(tx *sql.Conn) error {
		now := s.ingestNow().Unix()
		rows, err := ingestDiscardRows(ctx, tx, profile, now, receipt, 1, 0)
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return checkIngestDiscardTarget(ctx, tx, profile, receipt)
		}
		expected, err := ingestDiscardDigest(profile, receipt, rows)
		if err != nil {
			return err
		}
		if expected != digest {
			return db.ErrConflict
		}
		if err := discardIngestRow(ctx, tx, profile, actor, now, rows[0].candidate); err != nil {
			return err
		}
		return purgeIngestDiscards(ctx, tx, profile, now)
	})
}

// discardIngestSet keeps the original selection across transactions. Each batch
// compares the current queue with the unprocessed suffix before applying its CAS.
func (s *Sqlite) discardIngestSet(ctx context.Context, profile, actor int32, digest string) error {
	var remaining []ingestDiscardCandidate
	for first := true; first || len(remaining) > 0; first = false {
		var processed int
		err := s.withIngestWriter(ctx, func(tx *sql.Conn) error {
			now := s.ingestNow().Unix()
			if first {
				preview, err := ingestDiscardSetDigest(ctx, tx, profile, now, &remaining)
				if err != nil {
					return err
				}
				if preview.Digest != digest {
					return db.ErrConflict
				}
			} else {
				i := 0
				err := scanIngestDiscardSet(ctx, tx, profile, now, func(c ingestDiscardCandidate) error {
					if i >= len(remaining) || c != remaining[i] {
						return db.ErrConflict
					}
					i++
					return nil
				})
				if err != nil {
					return err
				}
				if i != len(remaining) {
					return db.ErrConflict
				}
			}
			processed = min(ingestDiscardBatchSize, len(remaining))
			for _, candidate := range remaining[:processed] {
				if err := discardIngestRow(ctx, tx, profile, actor, now, candidate); err != nil {
					return err
				}
			}
			if processed == len(remaining) {
				return purgeIngestDiscards(ctx, tx, profile, now)
			}
			return nil
		})
		if err != nil {
			return err
		}
		remaining = remaining[processed:]
	}
	return nil
}

// purgeIngestDiscards bounds maintenance work and preserves at least 90 days of history.
func purgeIngestDiscards(ctx context.Context, tx *sql.Conn, profile int32, now int64) error {
	_, err := tx.ExecContext(ctx, `DELETE FROM ingest_discards WHERE id IN (
		SELECT id FROM ingest_discards WHERE profile_id=? AND discarded_at<?
		ORDER BY discarded_at,id LIMIT ?
	)`, profile, now-int64(ingestDiscardRetention/time.Second), ingestDiscardBatchSize)
	return err
}

// discardIngestRow copies only audit metadata, then deletes using the same CAS predicate.
func discardIngestRow(ctx context.Context, tx *sql.Conn, profile, actor int32, now int64, c ingestDiscardCandidate) error {
	where := ` WHERE profile_id=? AND receipt_id=? AND resolution_version=? AND work_version=? AND state=? AND parked_at IS ? AND received_at=? AND ` + ingestDiscardScope
	args := []any{profile, c.Receipt, c.Resolution, c.Work, c.State, c.Parked, c.Received, now}
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
