package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"maps"
	"slices"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/ingest"
)

// saveNativeOccurrenceAliases adds only names with an unambiguous owner under the resolver lock.
func saveNativeOccurrenceAliases(ctx context.Context, tx *sql.Conn, profile int32, match occurrenceMatch, aliases *ingest.NativeAliases) (int, error) {
	source := fmt.Sprintf("%s%d", db.ImportedAliasPrefix, profile)
	var track int64
	if err := tx.QueryRowContext(ctx, `SELECT coalesce(legacy_track_id,0) FROM occurrences WHERE id=?`, match.id).Scan(&track); err != nil {
		return 0, err
	}
	var conflicts int
	add := func(table, key string, id int64, names []string) error {
		if id == 0 {
			return nil
		}
		query := fmt.Sprintf(`SELECT EXISTS(SELECT 1 FROM %s WHERE alias=? AND %s<>?)`, table, key)
		insert := fmt.Sprintf(`INSERT INTO %s(%s,alias,source,is_primary) VALUES(?,?,?,0) ON CONFLICT(%s,alias) DO NOTHING`, table, key, key)
		for _, alias := range slices.Compact(slices.Sorted(slices.Values(names))) {
			var ownedElsewhere bool
			if err := tx.QueryRowContext(ctx, query, alias, id).Scan(&ownedElsewhere); err != nil {
				return err
			}
			if ownedElsewhere {
				conflicts++
				continue
			}
			if _, err := tx.ExecContext(ctx, insert, id, alias, source); err != nil {
				return err
			}
		}
		return nil
	}
	for _, name := range slices.Sorted(maps.Keys(aliases.Artists)) {
		var artist int64
		if err := tx.QueryRowContext(ctx, `SELECT coalesce(min(artist_id),0) FROM (
			SELECT a.artist_id FROM artist_aliases a JOIN artist_tracks t USING(artist_id)
			WHERE t.track_id=? AND a.alias=?
			UNION ALL SELECT artist_id FROM occurrence_credits WHERE occurrence_id=? AND name=?
		)`, track, name, match.id, name).Scan(&artist); err != nil {
			return 0, err
		}
		if err := add("artist_aliases", "artist_id", artist, aliases.Artists[name]); err != nil {
			return 0, err
		}
	}
	if err := add("release_aliases", "release_id", match.release, aliases.Release); err != nil {
		return 0, err
	}
	if err := add("track_aliases", "track_id", track, aliases.Track); err != nil {
		return 0, err
	}
	return conflicts, nil
}
