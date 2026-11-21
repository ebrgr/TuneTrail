package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ebrgr/TuneTrail/internal/db"
)

// profileDeleteSelection uses binary primary-name equality and the entire unordered credit set.
// Artist deletion requires exactly one credit, so collaborations always survive.
func profileDeleteSelection(kind string) (string, string, error) {
	var table, relation, key string
	switch kind {
	case "track":
		table, relation, key = "tracks_with_title", "artist_tracks", "track_id"
	case "album":
		table, relation, key = "releases_with_title", albumArtistRelations, "release_id"
	case "artist":
		return `SELECT l.rowid,l.track_id,l.listened_at,l.client FROM listens l
   JOIN artist_tracks c ON c.track_id=l.track_id JOIN artists_with_name a ON a.id=c.artist_id
   WHERE l.profile_id=? AND a.name COLLATE BINARY=(SELECT name FROM artists_with_name WHERE id=?) COLLATE BINARY
   AND (SELECT count(*) FROM artist_tracks WHERE track_id=l.track_id)=1 ORDER BY l.rowid`,
			`SELECT name FROM artists_with_name WHERE id=?`, nil
	default:
		return "", "", db.ErrInvalidInput
	}
	item := "l.track_id"
	if kind == "album" {
		item = "(SELECT release_id FROM tracks WHERE id=l.track_id)"
	}
	// EXCEPT in both directions compares sets, avoiding concatenation and delimiter ambiguities.
	credits := func(id string) string {
		return `SELECT a.name COLLATE BINARY FROM ` + relation + ` c JOIN artists_with_name a ON a.id=c.artist_id WHERE c.` + key + `=` + id
	}
	return `WITH source AS (SELECT id,title FROM ` + table + ` WHERE id=?)
  SELECT l.rowid,l.track_id,l.listened_at,l.client FROM listens l
  JOIN ` + table + ` item ON item.id=` + item + ` CROSS JOIN source
  WHERE l.profile_id=? AND item.title COLLATE BINARY=source.title COLLATE BINARY
  AND NOT EXISTS (` + credits("item.id") + ` EXCEPT ` + credits("source.id") + `)
  AND NOT EXISTS (` + credits("source.id") + ` EXCEPT ` + credits("item.id") + `) ORDER BY l.rowid`,
		`SELECT title FROM ` + table + ` WHERE id=?`, nil
}

// profileDeleteIn owns the single selection used for the displayed count and write revalidation.
func profileDeleteIn(ctx context.Context, tx *sql.Tx, profile int32, kind string, id int32) (db.ProfileDeletePlan, string, []any, error) {
	p := db.ProfileDeletePlan{ProfileID: profile, Kind: kind, ItemID: id}
	if profile <= 0 || id <= 0 {
		return p, "", nil, db.ErrInvalidInput
	}
	selection, nameQuery, err := profileDeleteSelection(kind)
	if err != nil {
		return p, "", nil, err
	}
	if err = tx.QueryRowContext(ctx, nameQuery, id).Scan(&p.Name); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = db.ErrNotFound
		}
		return p, "", nil, err
	}
	args := []any{id, profile}
	if kind == "artist" {
		args = []any{profile, id}
	}
	hash := sha256.New()
	enc := json.NewEncoder(hash)
	if err = enc.Encode([]any{profile, kind, id, p.Name}); err != nil {
		return p, "", nil, err
	}
	// A change to the reviewed credit set also requires a fresh confirmation.
	if kind != "artist" {
		relation, key := "artist_tracks", "track_id"
		if kind == "album" {
			relation, key = albumArtistRelations, "release_id"
		}
		credits, creditErr := tx.QueryContext(ctx, `SELECT DISTINCT a.name FROM `+relation+` c JOIN artists_with_name a ON a.id=c.artist_id WHERE c.`+key+`=? ORDER BY a.name COLLATE BINARY`, id)
		if creditErr != nil {
			return p, "", nil, creditErr
		}
		for credits.Next() {
			var name string
			if err = credits.Scan(&name); err == nil {
				err = enc.Encode(name)
			}
			if err != nil {
				credits.Close()
				return p, "", nil, err
			}
		}
		creditErr = credits.Err()
		closeErr := credits.Close()
		if creditErr != nil {
			return p, "", nil, creditErr
		}
		if closeErr != nil {
			return p, "", nil, closeErr
		}
	}
	rows, err := tx.QueryContext(ctx, selection, args...)
	if err != nil {
		return p, "", nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var rowid, track, at int64
		var client string
		if err = rows.Scan(&rowid, &track, &at, &client); err != nil {
			return p, "", nil, err
		}
		if err = enc.Encode([]any{rowid, track, at, client}); err != nil {
			return p, "", nil, err
		}
		p.Count++
	}
	if err = rows.Err(); err != nil {
		return p, "", nil, err
	}
	p.Fingerprint = hex.EncodeToString(hash.Sum(nil))
	return p, selection, args, nil
}

// PreviewProfileDelete reads a consistent personal listening snapshot without changing the catalog.
func (s *Sqlite) PreviewProfileDelete(ctx context.Context, profile int32, kind string, id int32) (db.ProfileDeletePlan, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return db.ProfileDeletePlan{}, err
	}
	defer tx.Rollback() // Read-only cleanup after errors; commit below closes a successful transaction.
	p, _, _, err := profileDeleteIn(ctx, tx, profile, kind, id)
	if err != nil {
		return p, err
	}
	return p, tx.Commit()
}

// ApplyProfileDelete rejects stale confirmations and clears receipt provenance for the affected profile.
func (s *Sqlite) ApplyProfileDelete(ctx context.Context, expected db.ProfileDeletePlan) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback() // Any validation, trigger, cancellation or commit failure rolls back the whole delete.
	current, selection, args, err := profileDeleteIn(ctx, tx, expected.ProfileID, expected.Kind, expected.ItemID)
	if errors.Is(err, db.ErrNotFound) {
		err = db.ErrConflict
	}
	if err != nil {
		return 0, err
	}
	if current.Fingerprint != expected.Fingerprint || current.Count != expected.Count {
		return 0, db.ErrConflict
	}
	if current.Count > 0 {
		if err := deleteSelectedActiveReceipts(ctx, tx, expected.ProfileID, selection, args); err != nil {
			return 0, staleWriteConflict(err)
		}
	}
	// Event cascades discard open relay work and archive sent diagnostics in this transaction.
	result, err := tx.ExecContext(ctx, `DELETE FROM listens WHERE profile_id=? AND rowid IN (SELECT rowid FROM (`+selection+`))`, append([]any{expected.ProfileID}, args...)...)
	if err != nil {
		return 0, staleWriteConflict(fmt.Errorf("delete profile listens: %w", err))
	}
	count, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	if count != current.Count {
		return 0, db.ErrConflict
	}
	// Shadow receipts have no stable event link for a bulk catalog deletion.
	if count > 0 {
		if _, err := tx.ExecContext(ctx, `DELETE FROM ingest_events WHERE profile_id=? AND mode='shadow'`, expected.ProfileID); err != nil {
			return 0, staleWriteConflict(err)
		}
		if err := deleteShadowIntegrations(ctx, tx, expected.ProfileID); err != nil {
			return 0, staleWriteConflict(err)
		}
	}
	if err = tx.Commit(); err != nil {
		return 0, staleWriteConflict(err)
	}
	return count, nil
}
