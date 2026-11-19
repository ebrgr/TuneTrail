package sqlite

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ebrgr/TuneTrail/internal/dedup"
	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/google/uuid"
	sqlitedriver "modernc.org/sqlite"
	sqlitecodes "modernc.org/sqlite/lib"
)

const receiptTime = "2006-01-02T15:04:05.000000000Z"

type preparedReceipt struct {
	ingest.Item
	snapshot, digest string
}

// ReceiveBatch reserves the SQLite writer before inspecting integrations, keys or candidates.
func (s *Sqlite) ReceiveBatch(ctx context.Context, batch ingest.Batch) (results []ingest.Result, err error) {
	defer func() {
		var storage *sqlitedriver.Error
		var unavailable *ingest.ErrUnavailable
		if !errors.As(err, &unavailable) && errors.As(err, &storage) {
			switch storage.Code() & 255 {
			case sqlitecodes.SQLITE_FULL, sqlitecodes.SQLITE_IOERR, sqlitecodes.SQLITE_READONLY, sqlitecodes.SQLITE_CANTOPEN, sqlitecodes.SQLITE_CORRUPT, sqlitecodes.SQLITE_NOTADB:
				err = &ingest.ErrUnavailable{Cause: err}
			}
		}
	}()
	if len(batch.Items) == 0 {
		return nil, nil
	}
	source := fmt.Sprintf("%s:%d", batch.Source, batch.ProfileID)
	batchID := uuid.NewString()
	prepared := make([]preparedReceipt, len(batch.Items))
	digests := make([]string, len(batch.Items))
	for n, item := range batch.Items {
		snapshot, digest := item.Encode()
		prepared[n] = preparedReceipt{Item: item, snapshot: string(snapshot), digest: digest}
		digests[n] = digest
	}
	var batchDigest string
	if batch.ClientKey != "" {
		data, _ := json.Marshal(digests)
		batchDigest = ingest.Digest(data)
	}
	tx, err := s.db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Close()
	committed := false
	defer func() {
		if committed {
			return
		}
		// A failed commit or a canceled request can leave a transaction open.
		// SQLite may already have rolled back on a storage error.
		if _, err := tx.ExecContext(context.WithoutCancel(ctx), "ROLLBACK"); err != nil && !isAlreadyRolledBack(err) {
			_ = tx.Raw(func(any) error { return driver.ErrBadConn })
		}
	}()
	if _, err = tx.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return nil, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO integrations (id, profile_id, source_type, created_at)
		VALUES (?, ?, ?, ?) ON CONFLICT(id) DO NOTHING`, source, batch.ProfileID, batch.Source, batch.ReceivedAt.Unix())
	if err != nil {
		return nil, err
	}
	var previousBatchDigest string
	if batch.ClientKey != "" {
		err := tx.QueryRowContext(ctx, `SELECT batch_digest FROM ingest_events
			WHERE profile_id=? AND source_instance_id=? AND operation=? AND client_idempotency_key=?
			ORDER BY item_ordinal LIMIT 1`, batch.ProfileID, source, batch.Operation, batch.ClientKey).Scan(&previousBatchDigest)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}
	for _, item := range prepared {
		result, err := receiveItem(ctx, tx, batch, item, source, batchID, batchDigest, previousBatchDigest)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	if _, err := tx.ExecContext(ctx, "COMMIT"); err != nil {
		return nil, &ingest.ErrUnavailable{Cause: err}
	}
	committed = true
	return results, nil
}

// isAlreadyRolledBack recognizes automatic SQLite rollback after a storage error.
func isAlreadyRolledBack(err error) bool {
	var sqliteErr *sqlitedriver.Error
	return errors.As(err, &sqliteErr) && sqliteErr.Code() == sqlitecodes.SQLITE_ERROR &&
		strings.Contains(sqliteErr.Error(), "no transaction is active")
}

const receiptColumns = `receipt_id, source_instance_id, coalesce(external_namespace,''), coalesce(external_event_id,''),
	snapshot_digest, coalesce(batch_digest,''), time_precision, listened_at, original_snapshot, state`

// scanReceipt reads the narrow projection used by shadow rules.
func scanReceipt(row interface{ Scan(...any) error }) (r dedup.Receipt, err error) {
	var instant, snapshot string
	if err = row.Scan(&r.ID, &r.Source, &r.Namespace, &r.ExternalID, &r.Digest, &r.BatchDigest, &r.Precision, &instant, &snapshot, &r.State); err != nil {
		return r, err
	}
	r.Time, err = time.Parse(receiptTime, instant)
	if err == nil {
		err = json.Unmarshal([]byte(snapshot), &r.Snapshot)
	}
	return r, err
}

// findKey checks each applicable unique index under the same writer transaction.
func findKey(ctx context.Context, tx *sql.Conn, b ingest.Batch, i ingest.Item, source string) (*dedup.Receipt, error) {
	base := `SELECT ` + receiptColumns + ` FROM ingest_events WHERE profile_id=? AND source_instance_id=? AND `
	var found *dedup.Receipt
	find := func(predicate string, args ...any) error {
		r, err := scanReceipt(tx.QueryRowContext(ctx, base+predicate, append([]any{b.ProfileID, source}, args...)...))
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if found != nil && found.ID != r.ID {
			return errors.New("receipt keys resolve to different items")
		}
		found = &r
		return nil
	}
	if i.ExternalID != "" {
		if err := find("external_id_trusted=1 AND external_namespace=? AND external_event_id=?", i.ExternalNamespace, i.ExternalID); err != nil {
			return nil, err
		}
	}
	if b.ClientKey != "" {
		if err := find("operation=? AND client_idempotency_key=? AND item_ordinal=?", b.Operation, b.ClientKey, i.Ordinal); err != nil {
			return nil, err
		}
	}
	if b.FileDigest != "" {
		if err := find("file_digest=? AND file_format=? AND format_version='1' AND item_ordinal=?", b.FileDigest, b.FileFormat, i.Ordinal); err != nil {
			return nil, err
		}
	}
	return found, nil
}

// receiveItem records the decision without setting any of the future identity columns.
func receiveItem(ctx context.Context, tx *sql.Conn, b ingest.Batch, i preparedReceipt, source, batchID, batchDigest, previousBatchDigest string) (ingest.Result, error) {
	r := dedup.Receipt{Source: source, Namespace: i.ExternalNamespace, ExternalID: i.ExternalID, Digest: i.digest,
		BatchDigest: batchDigest, Snapshot: i.Snapshot, Time: i.ListenedAt, Precision: "second"}
	if r.Time.IsZero() {
		r.Time, r.Precision = b.ReceivedAt, "estimated"
	}
	previous, err := findKey(ctx, tx, b, i.Item, source)
	if err != nil {
		return ingest.Result{}, err
	}
	decision := dedup.Compare(dedup.Shadow, r, previous, previous != nil)
	if previous == nil && previousBatchDigest != "" && previousBatchDigest != batchDigest {
		// An extra ordinal under a reused batch key is still part of the conflict.
		decision.Value, decision.Reason = "review", "client_batch_conflict"
	} else if previous == nil {
		decision, err = compareNearby(ctx, tx, b.ProfileID, r, decision)
		if err != nil {
			return ingest.Result{}, err
		}
	}
	reasons, _ := json.Marshal([]string{decision.Reason})
	evidence, _ := json.Marshal(decision.Evidence)
	state := "received"
	if decision.Value == "review" {
		state = "pending_duplicate"
	}
	id := uuid.NewString()
	if previous != nil {
		id = previous.ID
		if decision.Value == "same" {
			// A keyed retry returns the existing receipt without rewriting its verdict.
			return ingest.Result{ReceiptID: id, Decision: decision.Value, Pending: previous.State == "pending_duplicate"}, nil
		}
		observation, err := tx.ExecContext(ctx, `INSERT INTO ingest_observations(id,profile_id,receipt_id,schema_version,snapshot_digest,digest_version,original_snapshot,received_at)
				VALUES(?,?,?,1,?,?,?,?) ON CONFLICT(profile_id,receipt_id,digest_version,snapshot_digest) DO NOTHING`,
			uuid.NewString(), b.ProfileID, id, i.digest, ingest.DigestVersion, i.snapshot, b.ReceivedAt.Unix())
		if err != nil {
			return ingest.Result{}, err
		}
		added, err := observation.RowsAffected()
		if err != nil {
			return ingest.Result{}, err
		}
		// Only a new conflicting observation can update the stored review evidence.
		if added > 0 {
			_, err = tx.ExecContext(ctx, `UPDATE ingest_events SET duplicate_decision=?,reason_codes=?,evidence=?,rule_version=?,state=?
				WHERE profile_id=? AND receipt_id=?`, decision.Value, string(reasons), string(evidence), decision.RuleVersion, state, b.ProfileID, id)
		}
		return ingest.Result{ReceiptID: id, Decision: decision.Value, Pending: true}, err
	} else {
		_, err = tx.ExecContext(ctx, `INSERT INTO ingest_events
			(receipt_id,profile_id,source_instance_id,schema_version,source_type,adapter_version,batch_id,item_ordinal,operation,received_at,
			external_namespace,external_event_id,external_id_trusted,client_idempotency_key,batch_digest,file_digest,file_format,format_version,
			listened_at,original_time,time_precision,time_semantics,original_snapshot,snapshot_digest,digest_version,state,duplicate_decision,reason_codes,evidence,rule_version)
			VALUES(?,?,?,1,?,'1',?,?,?,?,nullif(?,''),nullif(?,''),?,nullif(?,''),nullif(?,''),nullif(?,''),nullif(?,''),CASE WHEN ?!='' THEN '1' END,
			?,nullif(?,''),?,'unknown',?,?,?,?,?,?,?,?)`,
			id, b.ProfileID, source, b.Source, batchID, i.Ordinal, b.Operation, b.ReceivedAt.Unix(), i.ExternalNamespace, i.ExternalID, i.ExternalID != "",
			b.ClientKey, batchDigest, b.FileDigest, b.FileFormat, b.FileDigest, r.Time.UTC().Format(receiptTime), i.OriginalTime(), r.Precision, i.snapshot, i.digest, ingest.DigestVersion, state, decision.Value, string(reasons), string(evidence), decision.RuleVersion)
	}
	return ingest.Result{ReceiptID: id, Decision: decision.Value, Pending: state == "pending_duplicate"}, err
}

// compareNearby looks only at receipts for this profile within the rule's UTC window.
func compareNearby(ctx context.Context, tx *sql.Conn, profile int32, r dedup.Receipt, decision dedup.Decision) (dedup.Decision, error) {
	rows, err := tx.QueryContext(ctx, `SELECT `+receiptColumns+` FROM ingest_events
		WHERE profile_id=? AND listened_at BETWEEN ? AND ? AND state!='invalid'
		ORDER BY listened_at,receipt_id`, profile, r.Time.Add(-dedup.Shadow.Window).UTC().Format(receiptTime), r.Time.Add(dedup.Shadow.Window).UTC().Format(receiptTime))
	if err != nil {
		return decision, err
	}
	defer rows.Close()
	for rows.Next() {
		candidate, err := scanReceipt(rows)
		if err != nil {
			return decision, err
		}
		result := dedup.Compare(dedup.Shadow, r, &candidate, false)
		if result.Value == "review" {
			return result, nil
		}
	}
	return decision, rows.Err()
}

// ReadFileReceipts returns checkpoints in absolute ordinal order, independent of staging.
// Pass -1 to begin and the last returned ordinal to continue.
func (s *Sqlite) ReadFileReceipts(ctx context.Context, profile int32, format, digest string, after int) ([]ingest.FileReceipt, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT item_ordinal,receipt_id,original_snapshot,listened_at,time_precision,state
		FROM ingest_events WHERE profile_id=? AND source_instance_id=? AND file_format=? AND format_version='1'
		  AND file_digest=? AND item_ordinal>? ORDER BY item_ordinal LIMIT 500`,
		profile, fmt.Sprintf("file:%d", profile), format, digest, after)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []ingest.FileReceipt
	for rows.Next() {
		var record ingest.FileReceipt
		var snapshot, instant, state string
		if err := rows.Scan(&record.Ordinal, &record.ReceiptID, &snapshot, &instant, &record.Precision, &state); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(snapshot), &record.Snapshot); err != nil {
			return nil, err
		}
		record.ListenedAt, err = time.Parse(receiptTime, instant)
		if err != nil {
			return nil, err
		}
		record.Pending = state == "pending_duplicate"
		result = append(result, record)
	}
	return result, rows.Err()
}
