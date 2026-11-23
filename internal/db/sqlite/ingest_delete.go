package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/ebrgr/TuneTrail/internal/dedup"
	"github.com/ebrgr/TuneTrail/internal/ingest"
)

// deleteListenReceipts compares shadow snapshots with the deleted track's aliases and credits in the same profile and UTC second. Receipts own their variants.
func deleteListenReceipts(ctx context.Context, tx *sql.Tx, profile, track int32, at time.Time) error {
	rows, err := tx.QueryContext(ctx, `SELECT 'title',alias FROM track_aliases WHERE track_id=?
		UNION ALL SELECT 'artist',a.alias FROM artist_tracks c JOIN artist_aliases a ON a.artist_id=c.artist_id WHERE c.track_id=?`, track, track)
	if err != nil {
		return err
	}
	music := map[string]map[string]bool{"title": {}, "artist": {}}
	for rows.Next() {
		var kind, name string
		if err := rows.Scan(&kind, &name); err != nil {
			rows.Close()
			return err
		}
		music[kind][dedup.NormalizeName(name)] = true
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if err := rows.Err(); err != nil {
		return err
	}

	start := at.UTC().Truncate(time.Second)
	rows, err = tx.QueryContext(ctx, `WITH receipts AS (
		SELECT receipt_id,original_snapshot FROM ingest_events WHERE profile_id=? AND mode='shadow' AND listened_at>=? AND listened_at<?
	) SELECT receipt_id,json_extract(original_snapshot,'$.artist'),json_extract(original_snapshot,'$.title') FROM receipts
	UNION ALL SELECT o.receipt_id,json_extract(o.original_snapshot,'$.artist'),json_extract(o.original_snapshot,'$.title')
	FROM receipts r JOIN ingest_observations o ON o.receipt_id=r.receipt_id AND o.profile_id=?`,
		profile, start.Format(receiptTime), start.Add(time.Second).Format(receiptTime), profile)
	if err != nil {
		return err
	}
	ids := make(map[string]struct{})
	for rows.Next() {
		var id string
		var artist, title sql.NullString
		if err := rows.Scan(&id, &artist, &title); err != nil {
			rows.Close()
			return err
		}
		if artist.Valid && title.Valid && music["artist"][dedup.NormalizeName(artist.String)] && music["title"][dedup.NormalizeName(title.String)] {
			ids[id] = struct{}{}
		}
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for id := range ids {
		if _, err := tx.ExecContext(ctx, `DELETE FROM ingest_events WHERE profile_id=? AND receipt_id=? AND mode='shadow'`, profile, id); err != nil {
			return err
		}
	}
	return nil
}

// deleteSelectedActiveReceipts matches unresolved work against the same track
// selection as the history delete, while the selected listens are still present.
func deleteSelectedActiveReceipts(ctx context.Context, tx *sql.Tx, profile int32, selection string, args []any) error {
	// Occurrence receipts need an explicit resolved occurrence, never a title resemblance.
	parameters := append(append([]any(nil), args...), profile)
	_, err := tx.ExecContext(ctx, `WITH selected AS (`+selection+`)
		DELETE FROM ingest_events WHERE profile_id=? AND mode='active' AND resolver_variant='occurrence'
		AND event_id IS NULL AND closed_at IS NULL AND occurrence_id IN (SELECT track_id FROM selected)`, parameters...)
	return err
}

// receiptSourcePrefix keeps receipt creation and integration cleanup on the same identity format.
func receiptSourcePrefix(source string) string {
	return source + ":"
}

// deleteShadowIntegrations removes only adapter-owned identities, globally when profile is zero.
func deleteShadowIntegrations(ctx context.Context, tx *sql.Tx, profile int32) error {
	query := `DELETE FROM integrations WHERE id IN (? || profile_id, ? || profile_id) AND NOT EXISTS (SELECT 1 FROM ingest_events e WHERE e.profile_id=integrations.profile_id AND e.source_instance_id=integrations.id)`
	args := []any{receiptSourcePrefix(ingest.ShadowSourceListenBrainz), receiptSourcePrefix(ingest.ShadowSourceFile)}
	if profile != 0 {
		query += ` AND profile_id=?`
		args = append(args, profile)
	}
	_, err := tx.ExecContext(ctx, query, args...)
	return err
}
