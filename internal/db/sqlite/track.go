package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/imageurl"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/ebrgr/TuneTrail/internal/utils"
	"github.com/google/uuid"
	sqlitecodes "modernc.org/sqlite/lib"
)

func (s *Sqlite) GetTrack(ctx context.Context, opts db.GetTrackOpts) (*models.Track, error) {
	if opts.MusicBrainzID != uuid.Nil {
		var id int32
		err := s.queryRowContext(ctx,
			`SELECT id FROM tracks WHERE musicbrainz_id = ? LIMIT 1`, opts.MusicBrainzID.String()).Scan(&id)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("GetTrack: by MbzID: %w", db.ErrNotFound)
		}
		if err != nil {
			return nil, fmt.Errorf("GetTrack: by MbzID: %w", err)
		}
		opts.ID = id
	} else if len(opts.ArtistIDs) > 0 && opts.ReleaseID != 0 {
		id, err := s.trackByTrackInfo(ctx, opts.Title, opts.ReleaseID, opts.ArtistIDs)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("GetTrack: by track info: %w", db.ErrNotFound)
		}
		if err != nil {
			return nil, fmt.Errorf("GetTrack: by track info: %w", err)
		}
		opts.ID = id
	}

	return s.getTrackByID(ctx, opts.ID, opts.AllowUnplayed)
}

// getTrackByID keeps occurrence URLs usable even without a legacy track or release.
func (s *Sqlite) getTrackByID(ctx context.Context, id int32, allowUnplayed bool) (*models.Track, error) {
	if !allowUnplayed {
		if err := s.requireRelated(ctx, "track", id); err != nil {
			return nil, err
		}
	}

	var track models.Track
	var mbzID, image sql.NullString
	err := s.queryRowContext(ctx, `
		SELECT o.id, rec.musicbrainz_id, COALESCE(t.duration, 0), COALESCE(o.release_id, 0),
		       COALESCE(o.presented_title, t.title, ''), r.image
		FROM occurrences o
		JOIN recordings rec ON rec.id = o.recording_id
		LEFT JOIN tracks_with_title t ON o.legacy_track_id = t.id
		LEFT JOIN releases r ON o.release_id = r.id
		WHERE o.id = ? LIMIT 1`, id).
		Scan(&track.ID, &mbzID, &track.Duration, &track.AlbumID, &track.Title, &image)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("getTrackByID: %w", db.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("getTrackByID: %w", err)
	}
	track.MbzID = parseNullableUUID(mbzID)
	// Only the requesting profile may see an unambiguous external declaration.
	if track.MbzID == nil && db.ProfileID(ctx) > 0 {
		var claim sql.NullString
		err := s.db.QueryRowContext(ctx, `SELECT min(c.value) FROM metadata_values c
			WHERE c.profile_id=? AND c.recording_id=(SELECT recording_id FROM occurrences WHERE id=?) AND c.kind='mbid' AND NOT EXISTS (
			 SELECT 1 FROM ingest_events e WHERE e.profile_id=c.profile_id AND e.recording_id=c.recording_id AND e.state IN ('accepted','duplicate')
			 AND (coalesce(json_extract(e.original_snapshot,'$.recording_mbid'),'')<>'' OR coalesce(json_extract(e.original_snapshot,'$.mapped_recording_mbid'),'')<>''))
			HAVING count(DISTINCT c.value)=1`, db.ProfileID(ctx), id).Scan(&claim)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		if err == nil {
			track.MbzID = parseNullableUUID(claim)
		}
	}
	track.Image = imageurl.List(ctx, parseNullableUUID(image))

	artists, err := s.artistsForTrack(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getTrackByID: artists: %w", err)
	}
	track.Artists = artists

	var listenCount int64
	if err := s.queryRowContext(ctx, `SELECT COUNT(*) FROM listens WHERE track_id = ?`, id).Scan(&listenCount); err != nil {
		return nil, fmt.Errorf("getTrackByID: count listens: %w", err)
	}
	track.ListenCount = listenCount

	// Standalone occurrences can have measured time without a catalog track.
	var timeListened int64
	if err := s.queryRowContext(ctx, `
		SELECT COALESCE(SUM(COALESCE(l.duration_played, t.duration)), 0) FROM listens l LEFT JOIN tracks t ON l.track_id = t.id WHERE l.track_id = ?`,
		id).Scan(&timeListened); err != nil {
		return nil, fmt.Errorf("getTrackByID: time listened: %w", err)
	}
	track.TimeListened = timeListened

	var firstListenUnix int64
	err = s.queryRowContext(ctx,
		`SELECT listened_at FROM listens WHERE track_id = ? ORDER BY listened_at ASC LIMIT 1`, id).
		Scan(&firstListenUnix)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("getTrackByID: first listen: %w", err)
	}
	track.FirstListen = firstListenUnix

	var rank int64
	if err := s.queryRowContext(ctx, `
		SELECT rank FROM (
			SELECT track_id, RANK() OVER (ORDER BY COUNT(*) DESC) AS rank
			FROM listens GROUP BY track_id
		) WHERE track_id = ?`, id).Scan(&rank); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("getTrackByID: rank: %w", err)
	}
	track.AllTimeRank = rank

	return &track, nil
}

// trackByTrackInfo matches every exact alias and requires all supplied artists, in any order.
func (s *Sqlite) trackByTrackInfo(ctx context.Context, title string, releaseID int32, artistIDs []int32) (int32, error) {
	if len(artistIDs) == 0 {
		return 0, errors.New("trackByTrackInfo: no artist IDs provided")
	}
	artistIDs = slices.Clone(artistIDs)
	slices.Sort(artistIDs)
	artistIDs = slices.Compact(artistIDs)
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(artistIDs)), ",")
	query := fmt.Sprintf(`
		SELECT t.id FROM tracks t
		JOIN artist_tracks at2 ON at2.track_id = t.id
		WHERE EXISTS (SELECT 1 FROM track_aliases ta WHERE ta.track_id=t.id AND ta.alias=? AND (ta.source NOT LIKE 'ProfileImport:%%' OR ta.source=?))
		AND t.release_id = ? AND at2.artist_id IN (%s)
		GROUP BY t.id
		HAVING COUNT(DISTINCT at2.artist_id) = ? LIMIT 2`, placeholders)
	args := []any{title, db.CatalogAliasSource(ctx), releaseID}
	for _, id := range artistIDs {
		args = append(args, id)
	}
	args = append(args, len(artistIDs))
	rows, err := s.queryContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	var id int32
	count := 0
	for rows.Next() {
		if err := rows.Scan(&id); err != nil {
			return 0, err
		}
		count++
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if count == 0 {
		return 0, sql.ErrNoRows
	}
	if count > 1 {
		return 0, db.ErrAmbiguousTrack
	}
	return id, nil
}

// SaveTrack keeps owner-import identities out of the global MBID namespace.
func (s *Sqlite) SaveTrack(ctx context.Context, opts db.SaveTrackOpts) (*models.Track, error) {
	if db.ImportedCatalogProfile(ctx) > 0 {
		opts.RecordingMbzID = uuid.Nil
	}
	if len(opts.ArtistIDs) < 1 {
		return nil, errors.New("SaveTrack: required parameter 'ArtistIDs' missing")
	}
	if slices.Contains(opts.ArtistIDs, 0) {
		return nil, errors.New("SaveTrack: none of 'ArtistIDs' may be 0")
	}
	if opts.AlbumID == 0 {
		return nil, errors.New("SaveTrack: required parameter 'AlbumID' missing")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("SaveTrack: BeginTx: %w", err)
	}
	defer tx.Rollback()

	// A surviving recording can outlive the primary bridge that carried its MBID.
	var recording, id64 int64
	if opts.RecordingMbzID != uuid.Nil {
		err = tx.QueryRowContext(ctx, `SELECT coalesce((SELECT id FROM recordings WHERE musicbrainz_id=? AND NOT EXISTS (SELECT 1 FROM tracks WHERE musicbrainz_id=recordings.musicbrainz_id)),0)`, opts.RecordingMbzID.String()).Scan(&recording)
		if err != nil {
			return nil, err
		}
	}
	var mbid any = nullableUUID(&opts.RecordingMbzID)
	if recording != 0 {
		mbid = nil
	}
	if err := tx.QueryRowContext(ctx, insertBridgedTrackSQL, mbid, opts.AlbumID, opts.Duration).Scan(&id64); err != nil {
		return nil, fmt.Errorf("SaveTrack: insert: %w", err)
	}
	if recording != 0 {
		if err := attachBridgedRecording(ctx, tx, id64, recording); err != nil {
			return nil, err
		}
	}
	id := int32(id64)

	for i, artistID := range opts.ArtistIDs {
		isPrimary := 0
		if i == 0 {
			isPrimary = 1
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT OR IGNORE INTO artist_tracks (artist_id, track_id, is_primary) VALUES (?,?,?)`,
			artistID, id, isPrimary); err != nil {
			return nil, fmt.Errorf("SaveTrack: associate artist to track: %w", err)
		}
		// Album participation is derived from artist_tracks rather than persisted as an independent credit.
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO track_aliases (track_id, alias, source, is_primary) VALUES (?,?,?,1)`,
		id, opts.Title, db.ImportedAliasSource(ctx, "Canonical")); err != nil {
		return nil, fmt.Errorf("SaveTrack: canonical alias: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("SaveTrack: commit: %w", err)
	}

	ret := &models.Track{
		ID:       id,
		Title:    opts.Title,
		Duration: opts.Duration,
		AlbumID:  opts.AlbumID,
	}
	if opts.RecordingMbzID != uuid.Nil && recording == 0 {
		u := opts.RecordingMbzID
		ret.MbzID = &u
	}
	return ret, nil
}

// Allocate past the occurrence high-water mark even after deleting the highest track.
const insertBridgedTrackSQL = `INSERT INTO tracks (id,musicbrainz_id,release_id,duration)
	VALUES (max(coalesce((SELECT seq FROM sqlite_sequence WHERE name='occurrences'),0),
	            coalesce((SELECT max(id) FROM tracks),0))+1,?,?,?) RETURNING id`

type bridgedTrackWriter interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

// attachBridgedRecording replaces the provisional recording before the first receipt link.
func attachBridgedRecording(ctx context.Context, tx bridgedTrackWriter, occurrence, recording int64) error {
	var provisional int64
	if err := tx.QueryRowContext(ctx, `SELECT recording_id FROM occurrences WHERE id=?`, occurrence).Scan(&provisional); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE occurrences SET recording_id=? WHERE id=?`, recording, occurrence); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `DELETE FROM recordings WHERE id=? AND NOT EXISTS (SELECT 1 FROM occurrences WHERE recording_id=?)`, provisional, provisional)
	return err
}

// SaveTrackAliases retains owner-import provenance for every alias.
func (s *Sqlite) SaveTrackAliases(ctx context.Context, id int32, aliases []string, source string) error {
	source = db.ImportedAliasSource(ctx, source)
	if id == 0 {
		return errors.New("SaveTrackAliases: track id not specified")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("SaveTrackAliases: BeginTx: %w", err)
	}
	defer tx.Rollback()
	// Automatic enrichment must not turn an imported alias into a global resolver input.
	if !db.CatalogEditorial(ctx) {
		var provenance string
		err := tx.QueryRowContext(ctx, `SELECT source FROM track_aliases WHERE track_id=? AND is_primary=1 AND source LIKE 'ProfileImport:%' LIMIT 1`, id).Scan(&provenance)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if err == nil {
			source = provenance
		}
	}

	rows, err := tx.QueryContext(ctx, `SELECT alias FROM track_aliases WHERE track_id = ?`, id)
	if err != nil {
		return fmt.Errorf("SaveTrackAliases: fetch existing: %w", err)
	}
	for rows.Next() {
		var a string
		rows.Scan(&a)
		aliases = append(aliases, a)
	}
	rows.Close()

	utils.Unique(&aliases)
	for _, alias := range aliases {
		alias = strings.TrimSpace(alias)
		if alias == "" {
			return errors.New("SaveTrackAliases: aliases cannot be blank")
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT OR IGNORE INTO track_aliases (track_id, alias, source, is_primary) VALUES (?,?,?,0)`,
			id, alias, source); err != nil {
			return fmt.Errorf("SaveTrackAliases: insert: %w", err)
		}
	}
	return tx.Commit()
}

func (s *Sqlite) AddArtistsToTrack(ctx context.Context, opts db.AddArtistsToTrackOpts) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("AddArtistsToTrack: BeginTx: %w", err)
	}
	defer tx.Rollback()
	for _, artistID := range opts.ArtistIDs {
		if _, err := tx.ExecContext(ctx,
			`INSERT OR IGNORE INTO artist_tracks (artist_id, track_id, is_primary) VALUES (?,?,0)`,
			artistID, opts.TrackID); err != nil {
			return fmt.Errorf("AddArtistsToTrack: insert: %w", err)
		}
	}
	return tx.Commit()
}

// UpdateTrack applies identity checks and reserves global MBID promotion for explicit root edits.
func (s *Sqlite) UpdateTrack(ctx context.Context, opts db.UpdateTrackOpts) error {
	if opts.ID == 0 {
		return errors.New("UpdateTrack: track id not specified")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("UpdateTrack: BeginTx: %w", err)
	}
	defer tx.Rollback()
	var currentMBID sql.NullString
	err = tx.QueryRowContext(ctx, `SELECT musicbrainz_id FROM tracks WHERE id=?`, opts.ID).Scan(&currentMBID)
	if errors.Is(err, sql.ErrNoRows) {
		return db.ErrNotFound
	}
	if err != nil {
		return err
	}

	if opts.MusicBrainzID != uuid.Nil {
		allowed, err := importedMBIDAllowed(ctx, tx, "track_aliases", "track_id", opts.ID)
		if err != nil {
			return err
		}
		if !allowed {
			opts.MusicBrainzID = uuid.Nil
		}
	}

	if opts.MusicBrainzID != uuid.Nil {
		if opts.OnlyFillMusicBrainzID && currentMBID.Valid && currentMBID.String != uuid.Nil.String() && currentMBID.String != opts.MusicBrainzID.String() {
			return db.ErrTrackIdentityConflict
		}
		if opts.OnlyFillMusicBrainzID {
			var belongsElsewhere bool
			err := tx.QueryRowContext(ctx, `SELECT EXISTS (
				SELECT 1 FROM recordings r WHERE r.musicbrainz_id=? AND NOT EXISTS (
					SELECT 1 FROM occurrences o WHERE o.legacy_track_id=? AND o.recording_id=r.id))`,
				opts.MusicBrainzID.String(), opts.ID).Scan(&belongsElsewhere)
			if err != nil {
				return err
			}
			if belongsElsewhere {
				opts.MusicBrainzID = uuid.Nil
			}
		}
	}
	if opts.MusicBrainzID != uuid.Nil {
		if _, err := tx.ExecContext(ctx,
			`UPDATE tracks SET musicbrainz_id = ? WHERE id = ?`,
			opts.MusicBrainzID.String(), opts.ID); err != nil {
			if isOccurrenceGuard(err, "shared or validated") {
				if !opts.OnlyFillMusicBrainzID {
					return fmt.Errorf("UpdateTrack: MBID protected: %w: %w", db.ErrConflict, err)
				}
				// Leave the shared identity intact and continue the legacy listen.
			} else {
				// A surviving recording may own the MBID after its primary track was removed.
				if matchesSQLiteError(err, sqlitecodes.SQLITE_CONSTRAINT_UNIQUE, "recordings.musicbrainz_id") {
					return fmt.Errorf("UpdateTrack: recording MBID already assigned: %w: %w", db.ErrConflict, err)
				}
				return fmt.Errorf("UpdateTrack: mbzid: %w", err)
			}
		}
	}
	if opts.Duration != 0 {
		if _, err := tx.ExecContext(ctx,
			`UPDATE tracks SET duration = ? WHERE id = ?`,
			opts.Duration, opts.ID); err != nil {
			return fmt.Errorf("UpdateTrack: duration: %w", err)
		}
	}
	if len(opts.AddArtists) > 0 {
		for _, artistID := range opts.AddArtists {
			if artistID <= 0 {
				return db.ErrInvalidInput
			}
			var exists int
			err = tx.QueryRowContext(ctx, `SELECT 1 FROM artists WHERE id=?`, artistID).Scan(&exists)
			if errors.Is(err, sql.ErrNoRows) {
				return db.ErrNotFound
			}
			if err != nil {
				return fmt.Errorf("UpdateTrack: artist: %w", err)
			}
			if _, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO artist_tracks(artist_id,track_id,is_primary) VALUES(?,?,0)`, artistID, opts.ID); err != nil {
				return fmt.Errorf("UpdateTrack: add artist: %w", err)
			}
			// Album participation follows this track relationship and disappears with it.
		}
	}

	if len(opts.RemoveArtists) > 0 {
		for _, aid := range opts.RemoveArtists {
			if aid <= 0 {
				return db.ErrInvalidInput
			}
			result, e := tx.ExecContext(ctx, `DELETE FROM artist_tracks WHERE artist_id=? AND track_id=? AND is_primary=0`, aid, opts.ID)
			if e != nil {
				return fmt.Errorf("UpdateTrack: remove artist: %w", e)
			}
			n, e := result.RowsAffected()
			if e != nil {
				return e
			}
			if n == 0 {
				var primary int
				e = tx.QueryRowContext(ctx, `SELECT is_primary FROM artist_tracks WHERE artist_id=? AND track_id=?`, aid, opts.ID).Scan(&primary)
				if errors.Is(e, sql.ErrNoRows) {
					return db.ErrNotFound
				}
				if e != nil {
					return e
				}
				return db.ErrConflict
			}
			// The original listen no longer supports observations for the removed credit.
			if err := removeUncreditedMetadata(ctx, tx, aid, opts.ID); err != nil {
				return fmt.Errorf("UpdateTrack: remove metadata for artist: %w", err)
			}
		}
		// A relationship edit must not run the global orphan sweep: it can delete unrelated unplayed media.
	}

	return tx.Commit()
}

func (s *Sqlite) DeleteTrack(ctx context.Context, id int32) error {
	if id <= 0 {
		return db.ErrInvalidInput
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("DeleteTrack: BeginTx: %w", err)
	}
	defer tx.Rollback()

	result, err := deleteOccurrenceIn(ctx, tx, id)
	if err != nil {
		return fmt.Errorf("DeleteTrack: delete: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return db.ErrNotFound
	}
	return tx.Commit()
}

// deleteOccurrenceIn uses the bridge cascade only for occurrences with a legacy track.
func deleteOccurrenceIn(ctx context.Context, tx *sql.Tx, id int32) (sql.Result, error) {
	result, err := tx.ExecContext(ctx, `DELETE FROM tracks WHERE id=?`, id)
	if err != nil {
		return nil, err
	}
	n, err := result.RowsAffected()
	if err != nil || n != 0 {
		return result, err
	}
	return tx.ExecContext(ctx, `DELETE FROM occurrences WHERE id=? AND legacy_track_id IS NULL`, id)
}

// DeleteTrackAlias never removes a primary alias or reports success for a missing alias.
func (s *Sqlite) DeleteTrackAlias(ctx context.Context, id int32, alias string) error {
	if id <= 0 || strings.TrimSpace(alias) == "" {
		return db.ErrInvalidInput
	}
	result, err := s.db.ExecContext(ctx, `DELETE FROM track_aliases WHERE track_id=? AND alias=? AND is_primary=0`, id, alias)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	var primary int
	err = s.db.QueryRowContext(ctx, `SELECT is_primary FROM track_aliases WHERE track_id=? AND alias=?`, id, alias).Scan(&primary)
	if errors.Is(err, sql.ErrNoRows) {
		return db.ErrNotFound
	}
	if err != nil {
		return err
	}
	return db.ErrConflict
}

func (s *Sqlite) GetAllTrackAliases(ctx context.Context, id int32) ([]models.Alias, error) {
	if err := s.requireRelated(ctx, "track", id); err != nil {
		return nil, err
	}

	rows, err := s.queryContext(ctx,
		`SELECT alias, source, is_primary FROM track_aliases WHERE track_id = ?`, id)
	if err != nil {
		return nil, fmt.Errorf("GetAllTrackAliases: %w", err)
	}
	defer rows.Close()
	var aliases []models.Alias
	for rows.Next() {
		var a models.Alias
		var isPrimary int
		if err := rows.Scan(&a.Alias, &a.Source, &isPrimary); err != nil {
			return nil, err
		}
		a.ID = id
		a.Primary = isPrimary == 1
		aliases = append(aliases, a)
	}
	return aliases, rows.Err()
}

func (s *Sqlite) SetPrimaryTrackAlias(ctx context.Context, id int32, alias string) error {
	if id == 0 {
		return errors.New("SetPrimaryTrackAlias: track id not specified")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("SetPrimaryTrackAlias: BeginTx: %w", err)
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx,
		`SELECT alias, is_primary FROM track_aliases WHERE track_id = ?`, id)
	if err != nil {
		return fmt.Errorf("SetPrimaryTrackAlias: fetch: %w", err)
	}
	var primary, exists string
	for rows.Next() {
		var a string
		var isPrimary int
		if err := rows.Scan(&a, &isPrimary); err != nil {
			rows.Close()
			return err
		}
		if isPrimary == 1 {
			primary = a
		}
		if a == alias {
			exists = a
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	if primary == alias {
		return nil
	}
	if exists == "" {
		return db.ErrNotFound
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE track_aliases SET is_primary = 1 WHERE track_id = ? AND alias = ?`, id, alias); err != nil {
		return fmt.Errorf("SetPrimaryTrackAlias: set new: %w", err)
	}
	if primary != "" {
		if _, err := tx.ExecContext(ctx,
			`UPDATE track_aliases SET is_primary = 0 WHERE track_id = ? AND alias = ?`, id, primary); err != nil {
			return fmt.Errorf("SetPrimaryTrackAlias: clear old: %w", err)
		}
	}
	return tx.Commit()
}

func (s *Sqlite) GetTracksWithNoDurationButHaveMbzID(ctx context.Context, from int32) ([]*models.Track, error) {
	rows, err := s.queryContext(ctx, `
		SELECT id, musicbrainz_id, duration, release_id, title
		FROM tracks_with_title
		WHERE duration = 0 AND musicbrainz_id IS NOT NULL AND id > ?
		ORDER BY id ASC LIMIT 20`,
		from)
	if err != nil {
		return nil, fmt.Errorf("GetTracksWithNoDurationButHaveMbzID: %w", err)
	}
	defer rows.Close()
	var tracks []*models.Track
	for rows.Next() {
		var t models.Track
		var mbzID sql.NullString
		if err := rows.Scan(&t.ID, &mbzID, &t.Duration, &t.AlbumID, &t.Title); err != nil {
			return nil, err
		}
		t.MbzID = parseNullableUUID(mbzID)
		tracks = append(tracks, &t)
	}
	return tracks, rows.Err()
}

// GetTopTracksPaginated applies ranking and pagination after grouping the scoped listen events.
func (s *Sqlite) GetTopTracksPaginated(ctx context.Context, opts db.GetItemsOpts) (*db.PaginatedResponse[db.RankedItem[*models.Track]], error) {
	if opts.Limit == 0 {
		opts.Limit = defaultItemsPerPage
	}
	offset := (opts.Page - 1) * opts.Limit
	t1, t2 := db.TimeframeToTimeRange(opts.Timeframe)

	query, args := topTracksQuery(opts, t1.Unix(), t2.Unix(), offset)
	rows, err := s.queryContext(ctx, query, args...)

	if err != nil {
		return nil, fmt.Errorf("GetTopTracksPaginated: %w", err)
	}
	defer rows.Close()

	// Pre-allocate slice
	tracks := make([]db.RankedItem[*models.Track], 0, opts.Limit)
	var totalCount int64

	// Single iteration loop
	for rows.Next() {
		var t models.Track
		var mbzID, image sql.NullString
		var item db.RankedItem[*models.Track]

		// Scan totalCount directly alongside the row data
		if err := rows.Scan(&t.ID, &t.Title, &mbzID, &t.AlbumID, &t.AlbumTitle, &image, &t.ListenCount, &item.Rank, &totalCount); err != nil {
			return nil, err
		}

		t.MbzID = parseNullableUUID(mbzID)
		t.Image = imageurl.List(ctx, parseNullableUUID(image))

		item.Item = &t
		tracks = append(tracks, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := rows.Close(); err != nil {
		return nil, err
	}
	for i := range tracks {
		tracks[i].Item.Artists, err = s.artistsForTrack(ctx, tracks[i].Item.ID)
		if err != nil {
			return nil, err
		}
	}

	return &db.PaginatedResponse[db.RankedItem[*models.Track]]{
		Items:        tracks,
		TotalCount:   totalCount,
		ItemsPerPage: int32(opts.Limit),
		HasNextPage:  int64(offset+len(tracks)) < totalCount,
		CurrentPage:  int32(opts.Page),
	}, nil
}

// topTracksQuery aggregates eligible events and picks a stable, heard occurrence for each link.
func topTracksQuery(opts db.GetItemsOpts, from, to int64, offset int) (string, []any) {
	group := "o.recording_id"
	if opts.TrackGrouping == db.TrackByRelease {
		group += ", o.release_id"
	}
	args := []any{from, to}
	join, filter := "", ""
	switch {
	case opts.AlbumID > 0:
		filter = " AND o.release_id = ?"
		args = append(args, opts.AlbumID)
	case opts.ArtistID > 0:
		join = " JOIN artist_tracks at ON at.track_id = o.id"
		filter = " AND at.artist_id = ?"
		args = append(args, opts.ArtistID)
	}

	query := `
		WITH TrackCounts AS (
			SELECT MIN(o.id) AS track_id, COUNT(*) AS listen_count
			FROM listens l
			JOIN occurrences o ON o.id = l.track_id` + join + `
			WHERE l.listened_at BETWEEN ? AND ?` + filter + `
			GROUP BY ` + group + `
		),
		RankedTracks AS (
			SELECT track_id, listen_count,
			       RANK() OVER (ORDER BY listen_count DESC) AS rank,
			       COUNT(*) OVER () AS total_count
			FROM TrackCounts
			ORDER BY listen_count DESC, track_id
			LIMIT ? OFFSET ?
		)
		SELECT r.track_id, COALESCE(o.presented_title, (
		           SELECT alias FROM track_aliases
		           WHERE track_id = o.legacy_track_id AND is_primary = 1 ORDER BY alias LIMIT 1
		       ), ''), rec.musicbrainz_id, COALESCE(o.release_id, 0), COALESCE((
		           SELECT alias FROM release_aliases
		           WHERE release_id = o.release_id AND is_primary = 1 ORDER BY alias LIMIT 1
		       ), ''), rls.image,
		       r.listen_count, r.rank, r.total_count
		FROM RankedTracks r
		JOIN occurrences o ON o.id = r.track_id
		JOIN recordings rec ON rec.id = o.recording_id
		LEFT JOIN releases rls ON rls.id = o.release_id
		ORDER BY r.rank, r.track_id`
	return query, append(args, opts.Limit, offset)
}

// MergeTracks validates catalog identities before changing any relationships.
func (s *Sqlite) MergeTracks(ctx context.Context, fromId, toId int32) error {
	if fromId <= 0 || toId <= 0 || fromId == toId {
		return db.ErrInvalidInput
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("MergeTracks: BeginTx: %w", err)
	}
	defer tx.Rollback()
	if err := mergeTracksIn(ctx, tx, fromId, toId); err != nil {
		return err
	}
	return tx.Commit()
}

// Undo entries expire after 30 days. Catalog edits keep unexpired entries,
// occurrence deletion cascades, and track merge/undo transactions prune expired rows.
const recordingUndoSeconds = 30 * 24 * 60 * 60

// mergeTracksIn joins recording identity while leaving occurrences and listens in place.
func mergeTracksIn(ctx context.Context, tx *sql.Tx, fromId, toId int32) error {
	lookup := func(id int32) (int64, sql.NullString, error) {
		var recording int64
		var mbid sql.NullString
		err := tx.QueryRowContext(ctx, `SELECT r.id,r.musicbrainz_id FROM occurrences o
			JOIN recordings r ON r.id=o.recording_id WHERE o.id=?`, id).Scan(&recording, &mbid)
		if errors.Is(err, sql.ErrNoRows) {
			err = db.ErrNotFound
		}
		return recording, mbid, err
	}
	from, fromMBID, err := lookup(fromId)
	if err != nil {
		return err
	}
	to, toMBID, err := lookup(toId)
	if err != nil {
		return err
	}
	if from == to || (fromMBID.Valid && toMBID.Valid) {
		return db.ErrConflict
	}
	if fromMBID.Valid {
		from, to = to, from
	}
	now := time.Now().Unix()
	if _, err := tx.ExecContext(ctx, `DELETE FROM recording_moves WHERE stage=1 AND expires_at<=?`, now); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO recording_moves
		SELECT ?,?,id,?,?,0,? FROM occurrences WHERE recording_id=?`, fromId, toId, from, to, now+recordingUndoSeconds, from); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE occurrences SET recording_id=? WHERE recording_id=?`, to, from); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE recording_moves SET stage=1 WHERE source_track=? AND destination_track=?`, fromId, toId)
	return err
}

// UndoTrackMerge restores the affected occurrences and revises any receipts linked since the merge.
func (s *Sqlite) UndoTrackMerge(ctx context.Context, from, to int32) error {
	if from <= 0 || to <= 0 || from == to {
		return db.ErrInvalidInput
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Clear expired entries before checking for an available undo.
	if _, err := tx.ExecContext(ctx, `DELETE FROM recording_moves WHERE stage=1 AND expires_at<=?`, time.Now().Unix()); err != nil {
		return err
	}
	var overlap bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(
		SELECT 1 FROM recording_moves m JOIN occurrences o ON o.id=m.occurrence_id
		WHERE m.source_track=? AND m.destination_track=? AND
		(o.recording_id<>m.merged_recording OR NOT EXISTS (
			SELECT 1 FROM recordings r WHERE r.id=m.original_recording)))`, from, to).Scan(&overlap); err != nil {
		return err
	}
	if overlap {
		return db.ErrConflict
	}
	result, err := tx.ExecContext(ctx, `UPDATE recording_moves SET stage=2 WHERE source_track=? AND destination_track=?`, from, to)
	if err != nil {
		return err
	}
	if count, err := result.RowsAffected(); err != nil {
		return err
	} else if count == 0 {
		if err := tx.Commit(); err != nil {
			return err
		}
		return db.ErrNotFound
	}
	if _, err := tx.ExecContext(ctx, `UPDATE occurrences AS o SET recording_id=m.original_recording FROM recording_moves m
		WHERE m.stage=2 AND m.occurrence_id=o.id AND m.source_track=? AND m.destination_track=?`, from, to); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM recording_moves WHERE source_track=? AND destination_track=?`, from, to); err != nil {
		return err
	}
	return tx.Commit()
}

// CountTracks counts distinct heard recordings in the selected profile and period.
func (s *Sqlite) CountTracks(ctx context.Context, timeframe db.Timeframe) (int64, error) {
	t1, t2 := db.TimeframeToTimeRange(timeframe)
	var count int64
	err := s.queryRowContext(ctx,
		`SELECT COUNT(DISTINCT o.recording_id)
		 FROM listens l JOIN occurrences o ON o.id = l.track_id
		 WHERE l.listened_at BETWEEN ? AND ?`,
		t1.Unix(), t2.Unix()).Scan(&count)
	return count, err
}

// CountNewTracks uses the first listen to a recording across all its occurrences.
func (s *Sqlite) CountNewTracks(ctx context.Context, timeframe db.Timeframe) (int64, error) {
	t1, t2 := db.TimeframeToTimeRange(timeframe)
	var count int64
	err := s.queryRowContext(ctx, `
		SELECT COUNT(*) FROM (
			SELECT o.recording_id FROM listens l JOIN occurrences o ON o.id = l.track_id
			GROUP BY o.recording_id
			HAVING MIN(listened_at) BETWEEN ? AND ?
		)`,
		t1.Unix(), t2.Unix()).Scan(&count)
	return count, err
}

func (s *Sqlite) SearchTracks(ctx context.Context, q string) ([]*models.Track, error) {
	rows, err := s.queryContext(ctx, `
		SELECT DISTINCT t.id, t.title, t.musicbrainz_id, t.release_id, r.image
		FROM track_aliases ta
		JOIN tracks_with_title t ON ta.track_id = t.id
		JOIN releases r ON t.release_id = r.id
		WHERE (ta.alias LIKE ? OR ta.alias LIKE ?)
        AND (? = 0 OR EXISTS (SELECT 1 FROM listens l WHERE l.track_id=t.id))
		LIMIT 50`,
		q+"%", "%"+q+"%", db.ProfileID(ctx),
	)
	if err != nil {
		return nil, fmt.Errorf("SearchTracks: %w", err)
	}
	defer rows.Close()

	type candidate struct {
		track *models.Track
		score float64
	}
	seen := map[int32]float64{}
	var candidates []candidate

	for rows.Next() {
		var t models.Track
		var mbzID, image sql.NullString
		if err := rows.Scan(&t.ID, &t.Title, &mbzID, &t.AlbumID, &image); err != nil {
			return nil, err
		}
		t.MbzID = parseNullableUUID(mbzID)
		t.Image = imageurl.List(ctx, parseNullableUUID(image))
		score := fuzzyScore(q, t.Title)
		if prev, ok := seen[t.ID]; !ok || score > prev {
			seen[t.ID] = score
			candidates = append(candidates, candidate{&t, score})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var out []*models.Track
	for _, c := range sliceTopN(candidates, func(c candidate) float64 { return c.score }, 8) {
		artists, err := s.artistsForTrack(ctx, c.track.ID)
		if err != nil {
			return nil, err
		}
		c.track.Artists = artists
		out = append(out, c.track)
	}
	return out, nil
}
