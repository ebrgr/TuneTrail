package sqlite

import (
	"context"
	"database/sql"
)

// mergeMetadataArtist keeps conflicts and transfers the origin link only when it still supports the merged artist.
func mergeMetadataArtist(ctx context.Context, tx *sql.Tx, from, to int32) error {
	available, err := hasMetadataValues(ctx, tx)
	if err != nil || !available {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE occurrence_credits SET artist_id=? WHERE artist_id=?`, to, from); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO metadata_values(profile_id,artist_id,event_id,kind,value,instance,source,observed_at)
		SELECT m.profile_id,?,m.event_id,m.kind,m.value,m.instance,m.source,m.observed_at
		FROM metadata_values m JOIN listens l ON l.event_id=m.event_id AND l.profile_id=m.profile_id
		WHERE m.artist_id=? AND (
			EXISTS (SELECT 1 FROM artist_tracks WHERE track_id=l.track_id AND artist_id=?)
			OR EXISTS (SELECT 1 FROM occurrence_credits WHERE occurrence_id=l.track_id AND artist_id=?))`, to, from, to, to)
	return err
}

// mergeMetadataRelease preserves the original listen when it moved to the destination release.
func mergeMetadataRelease(ctx context.Context, tx *sql.Tx, from, to int32) error {
	available, err := hasMetadataValues(ctx, tx)
	if err != nil || !available {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO metadata_values(profile_id,release_id,event_id,kind,value,instance,source,observed_at)
		SELECT m.profile_id,?,m.event_id,m.kind,m.value,m.instance,m.source,m.observed_at
		FROM metadata_values m
		JOIN listens l ON l.event_id=m.event_id AND l.profile_id=m.profile_id
		JOIN occurrences o ON o.id=l.track_id
		WHERE m.release_id=? AND o.release_id=?`, to, from, to)
	return err
}

// Historical migration fixtures also exercise catalog merges on the old schema.
func hasMetadataValues(ctx context.Context, tx *sql.Tx) (bool, error) {
	var available bool
	err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM sqlite_schema WHERE type='table' AND name='metadata_values')`).Scan(&available)
	return available, err
}

// removeUncreditedMetadata drops declarations whose origin no longer credits the artist.
func removeUncreditedMetadata(ctx context.Context, tx *sql.Tx, artist, occurrence int32) error {
	available, err := hasMetadataValues(ctx, tx)
	if err != nil || !available {
		return err
	}
	_, err = tx.ExecContext(ctx, `DELETE FROM metadata_values
		WHERE artist_id=? AND event_id IN (SELECT event_id FROM listens WHERE track_id=?)
		AND NOT EXISTS (SELECT 1 FROM occurrence_credits WHERE occurrence_id=? AND artist_id=?)`,
		artist, occurrence, occurrence, artist)
	return err
}
