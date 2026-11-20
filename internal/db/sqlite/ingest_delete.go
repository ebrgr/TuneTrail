package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/ebrgr/TuneTrail/internal/db"
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
	if _, err := tx.ExecContext(ctx, `WITH selected AS (`+selection+`)
		DELETE FROM ingest_events WHERE profile_id=? AND mode='active' AND resolver_variant='occurrence'
		AND event_id IS NULL AND closed_at IS NULL AND occurrence_id IN (SELECT track_id FROM selected)`, parameters...); err != nil {
		return err
	}
	rows, err := tx.QueryContext(ctx, `WITH selected AS (SELECT DISTINCT track_id FROM (`+selection+`))
		SELECT s.track_id, 'title', a.alias FROM selected s JOIN track_aliases a ON a.track_id=s.track_id
		UNION ALL SELECT s.track_id, 'artist', a.alias FROM selected s JOIN artist_tracks c ON c.track_id=s.track_id
			JOIN artist_aliases a ON a.artist_id=c.artist_id
		UNION ALL SELECT track_id, '', '' FROM selected`, args...)
	if err != nil {
		return err
	}
	type aliases struct{ titles, artists []string }
	tracks := make(map[int32]*aliases)
	for rows.Next() {
		var track int32
		var kind, alias string
		if err := rows.Scan(&track, &kind, &alias); err != nil {
			rows.Close()
			return err
		}
		music := tracks[track]
		if music == nil {
			music = &aliases{}
			tracks[track] = music
		}
		switch kind {
		case "artist":
			music.artists = append(music.artists, dedup.NormalizeName(alias))
		case "title":
			music.titles = append(music.titles, dedup.NormalizeName(alias))
		}
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if err := rows.Err(); err != nil {
		return err
	}
	// Pair title and artist aliases from the same selected track.
	type credit struct{ artist, title string }
	credits := make(map[credit]struct{})
	for _, track := range tracks {
		for _, title := range track.titles {
			for _, artist := range track.artists {
				credits[credit{artist, title}] = struct{}{}
			}
		}
	}
	rows, err = tx.QueryContext(ctx, `SELECT receipt_id,coalesce(json_extract(original_snapshot,'$.catalog_track_id'),0),
		json_extract(original_snapshot,'$.artist'),json_extract(original_snapshot,'$.title')
		FROM ingest_events WHERE profile_id=? AND mode='active' AND resolver_variant='v9' AND closed_at IS NULL AND state IN ('received','retryable')`, profile)
	if err != nil {
		return err
	}
	var ids []string
	for rows.Next() {
		var id string
		var track int32
		var artist, title sql.NullString
		if err := rows.Scan(&id, &track, &artist, &title); err != nil {
			rows.Close()
			return err
		}
		_, matched := credits[credit{dedup.NormalizeName(artist.String), dedup.NormalizeName(title.String)}]
		if (track != 0 && tracks[track] != nil) || (track == 0 && artist.Valid && title.Valid && matched) {
			ids = append(ids, id)
		}
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	selected, err := json.Marshal(ids)
	if err != nil {
		return err
	}
	// Delete the receipt, its idempotency key and its children atomically.
	_, err = tx.ExecContext(ctx, `DELETE FROM ingest_events
		WHERE profile_id=? AND mode='active' AND receipt_id IN (SELECT value FROM json_each(?))`, profile, string(selected))
	return err
}

// DiscardIngestPending removes a profile's explicitly selected pending receipt without a tombstone.
func (s *Sqlite) DiscardIngestPending(ctx context.Context, profile int32, receipt string) error {
	if profile <= 0 || receipt == "" {
		return db.ErrInvalidInput
	}
	return s.withIngestWriter(ctx, func(tx *sql.Conn) error {
		result, err := tx.ExecContext(ctx, `DELETE FROM ingest_events WHERE profile_id=? AND receipt_id=? AND mode='active'
			AND event_id IS NULL AND closed_at IS NULL AND state IN ('pending_identity','pending_duplicate','received','retryable')`, profile, receipt)
		if err != nil {
			return err
		}
		count, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if count == 0 {
			return db.ErrNotFound
		}
		return nil
	})
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
