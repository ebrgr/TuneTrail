package sqlite

import (
	"context"
	"database/sql"
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
	snapshot, digest, legacyDigest string
}

// AdmitOccurrence stores the entire occurrence batch before returning its acknowledgement.
func (s *Sqlite) AdmitOccurrence(ctx context.Context, batch ingest.Batch) ([]ingest.Result, error) {
	source := receiptSourcePrefix(batch.Source) + fmt.Sprint(batch.ProfileID)
	batchID := uuid.NewString()
	prepared := make([]preparedReceipt, len(batch.Items))
	digests := make([]string, len(batch.Items))
	legacyDigests := make([]string, len(batch.Items))
	for n, item := range batch.Items {
		snapshot, digest := item.Encode()
		if batch.FileDigest != "" && item.Snapshot.MBIDSource == batch.FileFormat {
			// The admission clock is evidence, not a change to file contents.
			digestItem := item
			digestItem.Snapshot.MBIDDeclaredAt = ""
			_, digest = digestItem.Encode()
		}
		prepared[n] = preparedReceipt{Item: item, snapshot: string(snapshot), digest: digest, legacyDigest: item.LegacyDigest(batch.Source, batch.FileFormat)}
		digests[n] = digest
		legacyDigests[n] = prepared[n].legacyDigest
	}
	var batchDigest, legacyBatchDigest string
	if batch.ClientKey != "" {
		data, _ := json.Marshal(digests)
		batchDigest = ingest.Digest(data)
		legacy, _ := json.Marshal(legacyDigests)
		legacyBatchDigest = ingest.Digest(legacy)
	}
	var results []ingest.Result
	err := s.withIngestWriter(ctx, func(tx *sql.Conn) error {
		var policy string
		if err := tx.QueryRowContext(ctx, "SELECT policy FROM relay_delivery_gate WHERE id=1").Scan(&policy); err != nil {
			return &ingest.ErrUnavailable{Cause: err}
		}
		if len(prepared) == 0 {
			return nil
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO integrations (id, profile_id, source_type, created_at)
			VALUES (?, ?, ?, ?) ON CONFLICT(id) DO NOTHING`, source, batch.ProfileID, batch.Source, batch.ReceivedAt.Unix()); err != nil {
			return err
		}
		var previousBatchDigest, previousBatchVersion string
		if batch.ClientKey != "" {
			err := tx.QueryRowContext(ctx, `SELECT batch_digest,digest_version FROM ingest_events
				WHERE profile_id=? AND source_instance_id=? AND operation=? AND client_idempotency_key=?
				ORDER BY item_ordinal LIMIT 1`, batch.ProfileID, source, batch.Operation, batch.ClientKey).Scan(&previousBatchDigest, &previousBatchVersion)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return err
			}
		}
		comparableBatchDigest := batchDigest
		if previousBatchVersion == ingest.LegacyDigestVersion {
			comparableBatchDigest = legacyBatchDigest
		}
		batchConflict := previousBatchDigest != "" && previousBatchDigest != comparableBatchDigest
		for _, item := range prepared {
			result, err := receiveActiveItem(ctx, tx, batch, item, source, batchID, batchDigest,
				legacyBatchDigest, batchConflict, policy)
			if err != nil {
				return err
			}
			if result.DeliveryPolicy != "" && result.DeliveryPolicy != policy {
				return errors.New("receipt delivery policy did not match admission")
			}
			results = append(results, result)
		}
		return nil
	})
	if err != nil {
		return nil, ingestStorageError(err)
	}
	return results, nil
}

// ingestStorageError maps storage failures to a retryable HTTP acknowledgement.
func ingestStorageError(err error) error {
	var storage *sqlitedriver.Error
	if errors.As(err, &storage) {
		switch storage.Code() & 255 {
		case sqlitecodes.SQLITE_FULL, sqlitecodes.SQLITE_IOERR, sqlitecodes.SQLITE_READONLY, sqlitecodes.SQLITE_CANTOPEN,
			sqlitecodes.SQLITE_CORRUPT, sqlitecodes.SQLITE_NOTADB, sqlitecodes.SQLITE_BUSY, sqlitecodes.SQLITE_LOCKED:
			return &ingest.ErrUnavailable{Cause: err}
		}
	}
	return err
}

// isOccurrenceGuard recognizes controlled SQLite trigger errors without matching other SQL failures.
func isOccurrenceGuard(err error, message string) bool {
	return matchesSQLiteError(err, sqlitecodes.SQLITE_CONSTRAINT_TRIGGER, message)
}

// matchesSQLiteError recognizes the named constraint without classifying unrelated failures.
func matchesSQLiteError(err error, code int, message string) bool {
	var sqliteErr *sqlitedriver.Error
	return errors.As(err, &sqliteErr) && sqliteErr.Code() == code && strings.Contains(sqliteErr.Error(), message)
}

// isAlreadyRolledBack recognizes automatic SQLite rollback after a storage error.
func isAlreadyRolledBack(err error) bool {
	var sqliteErr *sqlitedriver.Error
	return errors.As(err, &sqliteErr) && sqliteErr.Code() == sqlitecodes.SQLITE_ERROR &&
		strings.Contains(sqliteErr.Error(), "no transaction is active")
}

const receiptColumns = `receipt_id, source_instance_id, coalesce(external_namespace,''), coalesce(external_event_id,''),
	snapshot_digest, coalesce(batch_digest,''), time_precision, listened_at, original_snapshot, state, digest_version`

// scanReceipt reads the stored receipt and its verdict.
func scanReceipt(row interface{ Scan(...any) error }) (r dedup.Receipt, err error) {
	var instant sql.NullString
	var snapshot string
	fields := []any{&r.ID, &r.Source, &r.Namespace, &r.ExternalID, &r.Digest, &r.BatchDigest, &r.Precision, &instant, &snapshot, &r.State, &r.DigestVersion, &r.Mode, &r.Decision}
	if err = row.Scan(fields...); err != nil {
		return r, err
	}
	if instant.Valid {
		r.Time, err = time.Parse(receiptTime, instant.String)
	}
	if err == nil {
		err = json.Unmarshal([]byte(snapshot), &r.Snapshot)
	}
	return r, err
}

var errReceiptKeyConflict = errors.New("receipt keys resolve to different items")

// findKey checks each applicable unique index under the same writer transaction.
func findKey(ctx context.Context, tx *sql.Conn, b ingest.Batch, i ingest.Item, source string) (*dedup.Receipt, error) {
	columns := receiptColumns + ",mode,coalesce(duplicate_decision,'')"
	base := `SELECT ` + columns + ` FROM ingest_events WHERE profile_id=? AND source_instance_id=? AND `
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
			// Prefer active ownership so a shadow retry never mutates another receipt.
			if r.Mode != found.Mode {
				if r.Mode == "active" {
					found = &r
				}
				return nil
			}
			return errReceiptKeyConflict
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
func replayIngest(r *dedup.Receipt) ingest.Result {
	return ingest.Result{ReceiptID: r.ID, Decision: r.Decision, Replay: true,
		Pending: r.State != "accepted" && r.State != "duplicate" && r.State != "invalid"}
}

// receiveActiveItem replays existing keys or inserts a new receipt.
func receiveActiveItem(ctx context.Context, tx *sql.Conn, b ingest.Batch, i preparedReceipt, source, batchID, batchDigest, legacyBatchDigest string, batchConflict bool, policy string) (ingest.Result, error) {
	previous, err := findKey(ctx, tx, b, i.Item, source)
	if errors.Is(err, errReceiptKeyConflict) {
		return ingest.Result{Decision: "review", Pending: true}, nil
	}
	if err != nil {
		return ingest.Result{}, err
	}
	// Schema 21 admits no unlinked shadow rows. Linked receipts retain ownership.
	if previous != nil && previous.Mode == "shadow" {
		return replayIngest(previous), nil
	}
	if previous != nil {
		result := replayIngest(previous)
		digest, comparableBatchDigest := i.digest, batchDigest
		if previous.DigestVersion == ingest.LegacyDigestVersion {
			digest, comparableBatchDigest = i.legacyDigest, legacyBatchDigest
		}
		if previous.Digest != digest || previous.BatchDigest != comparableBatchDigest {
			_, err := tx.ExecContext(ctx, `INSERT INTO ingest_observations(id,profile_id,receipt_id,schema_version,snapshot_digest,digest_version,original_snapshot,received_at)
				VALUES(?,?,?,1,?,?,?,?) ON CONFLICT(profile_id,receipt_id,digest_version,snapshot_digest) DO NOTHING`,
				uuid.NewString(), b.ProfileID, previous.ID, i.digest, ingest.DigestVersion, i.snapshot, b.ReceivedAt.Unix())
			if err != nil {
				return result, err
			}
		}
		// Read the completed file outcome without changing ordinal replay semantics.
		if b.FileDigest != "" {
			err := tx.QueryRowContext(ctx, `SELECT coalesce(json_extract(evidence,'$.ignored_alias_conflicts'),0), coalesce(json_extract(evidence,'$.ignored_mbid_conflicts'),0)
				FROM ingest_events WHERE profile_id=? AND receipt_id=?`, b.ProfileID, previous.ID).Scan(&result.IgnoredAliasConflicts, &result.IgnoredMBIDConflicts)
			if err != nil {
				return result, err
			}
		}
		// File replays keep their ordinal-based receipt semantics.
		if b.FileDigest == "" &&
			i.ExternalID != "" && previous.Digest == digest && (previous.State == "accepted" || previous.State == "duplicate") {
			return reviewTrustedReplay(ctx, tx, b, i.Item, previous.ID, batchID, policy, result)
		}
		return result, nil
	}
	id := uuid.NewString()
	at, precision, offset, semantics := i.Item.OccurrenceTime()
	if at.IsZero() {
		at, precision = b.ReceivedAt, "estimated"
	}
	reasons := "[]"
	if batchConflict {
		reasons = `["client_batch_conflict"]`
	}
	statement := `INSERT INTO ingest_events
		(receipt_id,profile_id,source_instance_id,schema_version,source_type,adapter_version,batch_id,item_ordinal,operation,received_at,
		external_namespace,external_event_id,external_id_trusted,client_idempotency_key,batch_digest,file_digest,file_format,format_version,
		listened_at,original_time,time_precision,time_semantics,original_offset,original_snapshot,snapshot_digest,digest_version,mode,state,reason_codes,rule_version,resolver_variant,delivery_policy,api_key_id)
		VALUES(?,?,?,1,?,'1',?,?,?,?,nullif(?,''),nullif(?,''),?,nullif(?,''),nullif(?,''),nullif(?,''),nullif(?,''),CASE WHEN ?!='' THEN '1' END,
		?,nullif(?,''),?,?,nullif(?,''),?,?,?,'active','received',?,?,'occurrence',?,CASE WHEN ?>0 THEN ? END)`
	args := []any{
		id, b.ProfileID, source, b.Source, batchID, i.Ordinal, b.Operation, b.ReceivedAt.Unix(), i.ExternalNamespace, i.ExternalID, i.ExternalID != "",
		b.ClientKey, batchDigest, b.FileDigest, b.FileFormat, b.FileDigest, at.UTC().Format(receiptTime), i.OriginalTime(), precision, semantics, offset,
		i.snapshot, i.digest, ingest.DigestVersion, reasons, activeRuleVersion, policy, b.APIKeyID, b.APIKeyID}
	var confirmed string
	err = tx.QueryRowContext(ctx, statement+" RETURNING delivery_policy", args...).Scan(&confirmed)
	return ingest.Result{ReceiptID: id, Pending: true, DeliveryPolicy: confirmed}, err
}

// ReadFileReceipts returns checkpoints in absolute ordinal order, independent of staging.
// Pass -1 to begin and the last returned ordinal to continue.
func (s *Sqlite) ReadFileReceipts(ctx context.Context, profile int32, format, digest string, after int) ([]ingest.FileReceipt, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT item_ordinal,receipt_id,original_snapshot,listened_at,time_precision,state,
		coalesce(json_extract(evidence,'$.ignored_alias_conflicts'),0), coalesce(json_extract(evidence,'$.ignored_mbid_conflicts'),0)
		FROM ingest_events WHERE profile_id=? AND source_instance_id=? AND file_format=? AND format_version='1'
		  AND file_digest=? AND item_ordinal>? ORDER BY item_ordinal LIMIT 500`,
		profile, receiptSourcePrefix(ingest.ShadowSourceFile)+fmt.Sprint(profile), format, digest, after)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []ingest.FileReceipt
	for rows.Next() {
		var record ingest.FileReceipt
		var snapshot, instant, state string
		if err := rows.Scan(&record.Ordinal, &record.ReceiptID, &snapshot, &instant, &record.Precision, &state, &record.IgnoredAliasConflicts, &record.IgnoredMBIDConflicts); err != nil {
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
