package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/google/uuid"
)

// RestoreArchive commits the catalog mappings and accepted events together.
func (s *Sqlite) RestoreArchive(ctx context.Context, profile int32, items []*db.ExportItem) (int, error) {
	if profile <= 0 || (db.ImportedCatalogProfile(ctx) != 0 && db.ImportedCatalogProfile(ctx) != profile) {
		return 0, db.ErrInvalidInput
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var namespace string
	if err := tx.QueryRowContext(ctx, `SELECT namespace FROM archive_identity`).Scan(&namespace); err != nil {
		return 0, err
	}
	count := 0
	for _, item := range items {
		var exists bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(
			SELECT 1 FROM archive_events WHERE profile_id=? AND namespace=? AND source_event_id=?
			UNION ALL SELECT 1 FROM listens WHERE profile_id=? AND event_id=? AND ?=?)`,
			profile, item.EventNamespace, item.EventID, profile, item.EventID, item.EventNamespace, namespace).Scan(&exists); err != nil {
			return 0, err
		}
		if exists {
			continue
		}
		c := archiveCatalog{tx: tx, profile: profile, namespace: item.CatalogNamespace, localNamespace: namespace}
		occurrence, err := c.mapID(ctx, "occurrences", int64(item.TrackID), func() (int64, error) {
			return c.createOccurrence(ctx, item)
		})
		if err != nil {
			return 0, err
		}
		// Reuse a local event ID when available and apply the destination catalog limit.
		var eventID string
		if err := tx.QueryRowContext(ctx, `INSERT INTO listens(profile_id,track_id,listened_at,client,origin,event_id,duration_played)
			VALUES(?,?,?,coalesce(?,''),'archive_restore',CASE
				WHEN ?=? AND NOT EXISTS (SELECT 1 FROM listens WHERE event_id=?) THEN ?
				ELSE lower(hex(randomblob(16))) END,`+boundedPlayedSeconds+`) RETURNING event_id`,
			profile, occurrence, item.ListenedAt.Unix(), item.Client, item.EventNamespace, namespace, item.EventID, item.EventID, item.DurationPlayed, occurrence).Scan(&eventID); err != nil {
			return 0, err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO archive_events VALUES(?,?,?,?,?,?)`, profile,
			item.EventNamespace, item.EventID, eventID, item.Origin, string(item.Provenance)); err != nil {
			return 0, err
		}
		if err := c.restoreDeclarations(ctx, occurrence, item); err != nil {
			return 0, err
		}
		count++
	}
	return count, tx.Commit()
}

type archiveCatalog struct {
	tx             *sql.Tx
	profile        int32
	namespace      string
	localNamespace string
}

// mapID resolves source IDs without assuming numeric IDs or names are portable.
func (c archiveCatalog) mapID(ctx context.Context, table string, source int64, create func() (int64, error)) (int64, error) {
	var local int64
	err := c.tx.QueryRowContext(ctx, `SELECT m.local_id FROM archive_catalog m JOIN `+table+` t ON t.id=m.local_id
		WHERE m.profile_id=? AND m.namespace=? AND m.kind=? AND m.source_id=?`,
		c.profile, c.namespace, table, source).Scan(&local)
	if !errors.Is(err, sql.ErrNoRows) {
		return local, err
	}
	trusted := db.ImportedCatalogProfile(ctx) == 0
	// Only administrator restores can share a verified mapping across destinations.
	if trusted {
		err = c.tx.QueryRowContext(ctx, `SELECT m.local_id FROM archive_catalog m JOIN `+table+` t ON t.id=m.local_id
			WHERE m.namespace=? AND m.kind=? AND m.source_id=? AND m.trusted=1 LIMIT 1`,
			c.namespace, table, source).Scan(&local)
		if err == nil {
			return local, c.saveMapping(ctx, table, source, local, trusted)
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return 0, err
		}
	}
	// A local source ID is authoritative for root. Profile owners can only reuse
	// a local occurrence already visible through their own listening history.
	if c.namespace == c.localNamespace {
		switch {
		case trusted:
			err = c.tx.QueryRowContext(ctx, `SELECT id FROM `+table+` WHERE id=?`, source).Scan(&local)
		case table == "occurrences":
			err = c.tx.QueryRowContext(ctx, `SELECT o.id FROM occurrences o WHERE o.id=? AND EXISTS (
				SELECT 1 FROM listens WHERE profile_id=? AND track_id=o.id)`, source, c.profile).Scan(&local)
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return local, err
		}
	}
	local, err = create()
	if err != nil {
		return 0, err
	}
	return local, c.saveMapping(ctx, table, source, local, trusted)
}

// saveMapping pins a restored catalog identity to its destination and namespace.
func (c archiveCatalog) saveMapping(ctx context.Context, table string, source, local int64, trusted bool) error {
	_, err := c.tx.ExecContext(ctx, `INSERT INTO archive_catalog VALUES(?,?,?,?,?,?)
		ON CONFLICT(profile_id,namespace,kind,source_id) DO UPDATE SET local_id=excluded.local_id,trusted=excluded.trusted`,
		c.profile, c.namespace, table, source, local, trusted)
	return err
}

// createOccurrence restores the bridge and its presentation independently of recording identity.
func (c archiveCatalog) createOccurrence(ctx context.Context, item *db.ExportItem) (int64, error) {
	trusted := db.ImportedCatalogProfile(ctx) == 0
	recording, err := c.mapID(ctx, "recordings", item.RecordingID, func() (int64, error) {
		var id int64
		source := "legacy_declared"
		if trusted && item.MBIDSource == "validated" {
			source = "validated"
		}
		mbid := nullableUUID(item.TrackMbid)
		declaredSource, _ := ingest.DeclaredMBIDProvenance(item.MBIDSource, item.MBIDDeclaredAt)
		if !trusted || declaredSource != "" {
			mbid = sql.NullString{}
		}
		err := c.tx.QueryRowContext(ctx, `INSERT INTO recordings(musicbrainz_id,mbid_source) VALUES(?,?)
			ON CONFLICT(musicbrainz_id) DO NOTHING RETURNING id`, mbid, source).Scan(&id)
		if errors.Is(err, sql.ErrNoRows) {
			err = c.tx.QueryRowContext(ctx, `SELECT id FROM recordings WHERE musicbrainz_id=?`, mbid).Scan(&id)
		}
		return id, err
	})
	if err != nil {
		return 0, err
	}
	var occurrence int64
	ref := item.TrackReference
	if !trusted {
		ref = ""
	}
	if item.ReleaseID == 0 {
		var title string
		for _, alias := range item.TrackAliases {
			if alias.Primary {
				title = alias.Alias
				break
			}
		}
		err = c.tx.QueryRowContext(ctx, `INSERT INTO occurrences(recording_id,presented_title) VALUES(?,?) RETURNING id`, recording, title).Scan(&occurrence)
		if err != nil {
			return 0, err
		}
		for i, artist := range item.Artists {
			var artistID any
			if artist.ID > 0 {
				id, err := c.restoreArtist(ctx, artist)
				if err != nil {
					return 0, err
				}
				artistID = id
			}
			if _, err := c.tx.ExecContext(ctx, `INSERT INTO occurrence_credits(occurrence_id,position,name,artist_id) VALUES(?,?,?,?)`, occurrence, i, artist.Name, artistID); err != nil {
				return 0, err
			}
		}
	} else {
		release, err := c.mapID(ctx, "releases", int64(item.ReleaseID), func() (int64, error) {
			id, created, err := c.createMedia(ctx, "release", nullableUUID(item.ReleaseMbid), item.ReleaseImageSource, item.VariousArtists, item.ReleaseAliases)
			if err != nil || !created {
				return id, err
			}
			for _, artist := range item.ReleaseArtists {
				credit, err := c.restoreArtist(ctx, artist)
				if err != nil {
					return 0, err
				}
				if _, err := c.tx.ExecContext(ctx, `INSERT INTO artist_releases VALUES(?,?,?) ON CONFLICT DO NOTHING`, credit, id, artist.IsPrimary); err != nil {
					return 0, err
				}
			}
			return id, nil
		})
		if err != nil {
			return 0, err
		}
		if err := c.tx.QueryRowContext(ctx, insertBridgedTrackSQL, nil, release, item.TrackDuration).Scan(&occurrence); err != nil {
			return 0, err
		}
		if err := attachBridgedRecording(ctx, c.tx, occurrence, recording); err != nil {
			return 0, err
		}
		if err := c.saveAliases(ctx, "track", occurrence, item.TrackAliases); err != nil {
			return 0, err
		}
		for _, a := range item.Artists {
			artist, err := c.restoreArtist(ctx, a)
			if err != nil {
				return 0, err
			}
			if _, err := c.tx.ExecContext(ctx, `INSERT INTO artist_tracks VALUES(?,?,?) ON CONFLICT DO NOTHING`, artist, occurrence, a.IsPrimary); err != nil {
				return 0, err
			}
		}
	}
	_, err = c.tx.ExecContext(ctx, `UPDATE occurrences SET track_mbid=nullif(?,''),disc_number=nullif(?,0),track_number=nullif(?,0) WHERE id=?`,
		ref, item.DiscNumber, item.TrackNumber, occurrence)
	return occurrence, err
}

// restoreArtist uses the same namespace for album and track credits.
func (c archiveCatalog) restoreArtist(ctx context.Context, artist models.ArtistWithFullAliases) (int64, error) {
	return c.mapID(ctx, "artists", int64(artist.ID), func() (int64, error) {
		mbid := nullableUUID(artist.MbzID)
		if artist.MBIDSource != "" {
			mbid = sql.NullString{}
		}
		id, _, err := c.createMedia(ctx, "artist", mbid, artist.ImageSource, false, artist.Aliases)
		return id, err
	})
}

// restoreDeclarations keeps imported external claims scoped and never promotes them to catalog proof.
func (c archiveCatalog) restoreDeclarations(ctx context.Context, occurrence int64, item *db.ExportItem) error {
	instance := "archive:" + c.namespace
	for _, declaration := range item.RecordingDeclarations {
		source, at := ingest.DeclaredMBIDProvenance(declaration.Source, declaration.DeclaredAt)
		if source == "" || at == "" || declaration.MBID == uuid.Nil {
			continue
		}
		digest := sha256.Sum256([]byte(source + "\x00" + declaration.MBID.String() + "\x00" + at + "\x00" + declaration.Instance))
		if _, err := c.tx.ExecContext(ctx, `INSERT OR IGNORE INTO enrichment_recording_claims VALUES(?,?,?,?,?,?)`, c.profile, occurrence, instance+":"+hex.EncodeToString(digest[:]), declaration.MBID.String(), source, at); err != nil {
			return err
		}
	}
	if source, at := ingest.DeclaredMBIDProvenance(item.MBIDSource, item.MBIDDeclaredAt); source != "" && at != "" && item.TrackMbid != nil && *item.TrackMbid != uuid.Nil {
		found := false
		for _, declaration := range item.RecordingDeclarations {
			if declaration.MBID == *item.TrackMbid && declaration.Source == source && declaration.DeclaredAt == at {
				found = true
				break
			}
		}
		if !found {
			if _, err := c.tx.ExecContext(ctx, `INSERT OR IGNORE INTO enrichment_recording_claims VALUES(?,?,?,?,?,?)`, c.profile, occurrence, instance, item.TrackMbid.String(), source, at); err != nil {
				return err
			}
		}
	}
	for _, artist := range item.Artists {
		if artist.ID <= 0 {
			continue
		}
		declarations := append([]models.MBIDDeclaration(nil), artist.MBIDDeclarations...)
		if source, at := ingest.DeclaredMBIDProvenance(artist.MBIDSource, artist.MBIDDeclaredAt); source != "" && at != "" && artist.MbzID != nil && *artist.MbzID != uuid.Nil {
			found := false
			for _, declaration := range declarations {
				if declaration.MBID == *artist.MbzID && declaration.Source == source && declaration.DeclaredAt == at {
					found = true
					break
				}
			}
			if !found {
				declarations = append(declarations, models.MBIDDeclaration{MBID: *artist.MbzID, Source: source, DeclaredAt: at})
			}
		}
		if len(declarations) == 0 {
			continue
		}
		var local int64
		err := c.tx.QueryRowContext(ctx, `SELECT local_id FROM archive_catalog WHERE profile_id=? AND namespace=? AND kind='artists' AND source_id=?`, c.profile, c.namespace, artist.ID).Scan(&local)
		if errors.Is(err, sql.ErrNoRows) && c.namespace == c.localNamespace {
			err = c.tx.QueryRowContext(ctx, `SELECT id FROM artists WHERE id=?`, artist.ID).Scan(&local)
		}
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return err
		}
		for _, declaration := range declarations {
			source, at := ingest.DeclaredMBIDProvenance(declaration.Source, declaration.DeclaredAt)
			if source == "" || at == "" || declaration.MBID == uuid.Nil {
				continue
			}
			digest := sha256.Sum256([]byte(source + "\x00" + declaration.MBID.String() + "\x00" + at + "\x00" + declaration.Instance))
			declarationInstance := instance + ":" + hex.EncodeToString(digest[:])
			// An archive mapping is usable only when this occurrence still credits that artist.
			_, err = c.tx.ExecContext(ctx, `INSERT OR IGNORE INTO enrichment_artist_claims SELECT ?,a.id,?,?,?,? FROM artists a
				WHERE a.id=? AND (EXISTS(SELECT 1 FROM artist_tracks WHERE track_id=? AND artist_id=a.id)
				OR EXISTS(SELECT 1 FROM occurrence_credits WHERE occurrence_id=? AND artist_id=a.id))`, c.profile, declarationInstance, declaration.MBID.String(), source, at, local, occurrence, occurrence)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// createMedia only claims canonical MBIDs for trusted imports.
func (c archiveCatalog) createMedia(ctx context.Context, kind string, mbid sql.NullString, image string, various bool, aliases []models.Alias) (int64, bool, error) {
	if db.ImportedCatalogProfile(ctx) > 0 {
		mbid = sql.NullString{}
	}
	var id int64
	table := "artists"
	query := `INSERT INTO artists(musicbrainz_id,image_source) VALUES(?,nullif(?,''))`
	args := []any{mbid, image}
	if kind == "release" {
		table = "releases"
		query = `INSERT INTO releases(musicbrainz_id,image_source,various_artists) VALUES(?,nullif(?,''),?)`
		args = append(args, various)
	}
	err := c.tx.QueryRowContext(ctx, query+` ON CONFLICT(musicbrainz_id) DO NOTHING RETURNING id`, args...).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		err = c.tx.QueryRowContext(ctx, `SELECT id FROM `+table+` WHERE musicbrainz_id=?`, mbid).Scan(&id)
		return id, false, err
	}
	if err != nil {
		return 0, false, err
	}
	return id, true, c.saveAliases(ctx, kind, id, aliases)
}

// saveAliases keeps imported aliases scoped under the same rules as legacy imports.
func (c archiveCatalog) saveAliases(ctx context.Context, kind string, id int64, aliases []models.Alias) error {
	for _, alias := range aliases {
		source := db.ImportedAliasSource(ctx, alias.Source)
		if _, err := c.tx.ExecContext(ctx, fmt.Sprintf("INSERT INTO %s_aliases VALUES(?,?,?,?) ON CONFLICT DO NOTHING", kind),
			id, alias.Alias, source, alias.Primary); err != nil {
			return err
		}
	}
	return nil
}
