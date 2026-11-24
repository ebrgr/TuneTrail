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
		SELECT m.profile_id,?,
			(SELECT l.event_id FROM listens l WHERE l.event_id=m.event_id AND (
				EXISTS (SELECT 1 FROM artist_tracks WHERE track_id=l.track_id AND artist_id=?)
				OR EXISTS (SELECT 1 FROM occurrence_credits WHERE occurrence_id=l.track_id AND artist_id=?))),
			m.kind,m.value,m.instance,m.source,m.observed_at FROM metadata_values m WHERE m.artist_id=?`, to, to, to, from)
	return err
}

// mergeMetadataRelease preserves values whose profiles still have a listen on the destination release.
func mergeMetadataRelease(ctx context.Context, tx *sql.Tx, from, to int32) error {
	available, err := hasMetadataValues(ctx, tx)
	if err != nil || !available {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO metadata_values(profile_id,release_id,event_id,kind,value,instance,source,observed_at)
		SELECT m.profile_id,?,
			(SELECT l.event_id FROM listens l JOIN occurrences o ON o.id=l.track_id WHERE l.event_id=m.event_id AND o.release_id=?),
			m.kind,m.value,m.instance,m.source,m.observed_at FROM metadata_values m WHERE m.release_id=?
		AND EXISTS (SELECT 1 FROM occurrences o CROSS JOIN listens l
			WHERE o.release_id=? AND l.profile_id=m.profile_id AND l.track_id=o.id)`, to, to, from, to)
	return err
}

// Historical migration fixtures also exercise catalog merges on the old schema.
func hasMetadataValues(ctx context.Context, tx *sql.Tx) (bool, error) {
	var available bool
	err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM sqlite_schema WHERE type='table' AND name='metadata_values')`).Scan(&available)
	return available, err
}
