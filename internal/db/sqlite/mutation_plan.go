package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/ebrgr/TuneTrail/internal/db"
	sqlite3 "modernc.org/sqlite"
)

// catalogFingerprint conservatively covers the complete catalog and listening history.
// This may cause an unrelated concurrent change to require a fresh confirmation.
func catalogFingerprint(ctx context.Context, tx *sql.Tx, p db.CatalogPlan) (string, error) {
	h := sha256.New()
	enc := json.NewEncoder(h)
	if err := enc.Encode([]any{p.Action, p.Kind, p.From, p.To, p.ReplaceImage}); err != nil {
		return "", err
	}
	for _, q := range []string{
		`SELECT id,musicbrainz_id,image,image_source FROM artists ORDER BY id`,
		`SELECT artist_id,alias,source,is_primary FROM artist_aliases ORDER BY artist_id,alias`,
		`SELECT id,musicbrainz_id,image,image_source,various_artists FROM releases ORDER BY id`,
		`SELECT release_id,alias,source,is_primary FROM release_aliases ORDER BY release_id,alias`,
		`SELECT artist_id,release_id,is_primary FROM artist_releases ORDER BY artist_id,release_id`,
		`SELECT id,musicbrainz_id,release_id,duration FROM tracks ORDER BY id`,
		`SELECT track_id,alias,source,is_primary FROM track_aliases ORDER BY track_id,alias`,
		`SELECT artist_id,track_id,is_primary FROM artist_tracks ORDER BY artist_id,track_id`,
		`SELECT profile_id,track_id,listened_at,client FROM listens ORDER BY profile_id,track_id,listened_at`,
		`SELECT id,slug,display_name FROM listening_profiles ORDER BY id`,
	} {
		if err := enc.Encode(q); err != nil {
			return "", err
		}
		rows, err := tx.QueryContext(ctx, q)
		if err != nil {
			return "", err
		}
		cols, err := rows.Columns()
		if err != nil {
			rows.Close()
			return "", err
		}
		for rows.Next() {
			vals := make([]any, len(cols))
			ptr := make([]any, len(cols))
			for i := range vals {
				ptr[i] = &vals[i]
			}
			if err = rows.Scan(ptr...); err != nil {
				rows.Close()
				return "", err
			}
			if err = enc.Encode(vals); err != nil {
				rows.Close()
				return "", err
			}
		}
		if err = rows.Err(); err != nil {
			rows.Close()
			return "", err
		}
		if err = rows.Close(); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
func planCatalogIn(ctx context.Context, tx *sql.Tx, p db.CatalogPlan) (db.CatalogPlan, error) {
	if p.From <= 0 || (p.Action != "delete" && p.Action != "merge") || (p.Action == "delete" && (p.To != 0 || p.ReplaceImage)) || (p.Action == "merge" && (p.To <= 0 || p.From == p.To)) {
		return p, db.ErrInvalidInput
	}
	p.Profiles = nil
	p.DestinationProfiles = nil
	p.AffectedTrackIDs = nil
	p.SourceAliases = nil
	p.DestinationAliases = nil
	p.SourceArtists = nil
	p.DestinationArtists = nil
	var table, aliases, foreign string
	switch p.Kind {
	case "artist":
		table, aliases, foreign = "artists", "artist_aliases", "artist_id"
	case "album":
		table, aliases, foreign = "releases", "release_aliases", "release_id"
	case "track":
		table, aliases, foreign = "tracks", "track_aliases", "track_id"
		if p.ReplaceImage {
			return p, db.ErrInvalidInput
		}
	default:
		return p, db.ErrInvalidInput
	}
	name := func(id int32) (string, string, string, error) {
		q := `SELECT COALESCE((SELECT alias FROM ` + aliases + ` WHERE ` + foreign + `=x.id AND is_primary=1 LIMIT 1),''),`
		if p.Kind == "track" {
			q += `COALESCE((SELECT image FROM releases WHERE id=x.release_id),'')`
		} else {
			q += `COALESCE(x.image,'')`
		}
		q += `,COALESCE(x.musicbrainz_id,'') FROM ` + table + ` x WHERE x.id=?`
		var n, img, mbid string
		e := tx.QueryRowContext(ctx, q, id).Scan(&n, &img, &mbid)
		if errors.Is(e, sql.ErrNoRows) {
			return "", "", "", db.ErrNotFound
		}
		return n, img, mbid, e
	}
	var err error
	p.SourceName, p.SourceImage, p.SourceMBID, err = name(p.From)
	if err != nil {
		return p, err
	}
	if p.Action == "merge" {
		p.DestinationName, p.DestinationImage, p.DestinationMBID, err = name(p.To)
		if err != nil {
			return p, err
		}
	}
	getAliases := func(id int32) ([]string, error) {
		rows, e := tx.QueryContext(ctx, `SELECT alias FROM `+aliases+` WHERE `+foreign+`=? ORDER BY is_primary DESC,alias`, id)
		if e != nil {
			return nil, e
		}
		defer rows.Close()
		var out []string
		for rows.Next() {
			var a string
			if e = rows.Scan(&a); e != nil {
				return nil, e
			}
			out = append(out, a)
		}
		return out, rows.Err()
	}
	p.SourceAliases, err = getAliases(p.From)
	if err != nil {
		return p, err
	}
	if p.Action == "merge" {
		p.DestinationAliases, err = getAliases(p.To)
		if err != nil {
			return p, err
		}
	}
	getArtists := func(id int32) ([]db.CatalogImpactArtist, error) {
		var rel, col string
		if p.Kind == "album" {
			rel, col = "artist_releases", "release_id"
		} else if p.Kind == "track" {
			rel, col = "artist_tracks", "track_id"
		} else {
			return nil, nil
		}
		rows, e := tx.QueryContext(ctx, `SELECT r.artist_id,COALESCE((SELECT alias FROM artist_aliases WHERE artist_id=r.artist_id AND is_primary=1 LIMIT 1),''),r.is_primary FROM `+rel+` r WHERE r.`+col+`=? ORDER BY r.artist_id`, id)
		if e != nil {
			return nil, e
		}
		defer rows.Close()
		var out []db.CatalogImpactArtist
		for rows.Next() {
			var a db.CatalogImpactArtist
			if e = rows.Scan(&a.ID, &a.Name, &a.Primary); e != nil {
				return nil, e
			}
			out = append(out, a)
		}
		return out, rows.Err()
	}
	p.SourceArtists, err = getArtists(p.From)
	if err != nil {
		return p, err
	}
	if p.Action == "merge" {
		p.DestinationArtists, err = getArtists(p.To)
		if err != nil {
			return p, err
		}
	}

	// Source-linked tracks determine affected personal histories; deletion of an artist
	// additionally includes albums removed by the existing last-artist trigger.
	var tracks string
	var args []any
	switch p.Kind {
	case "track":
		tracks = `SELECT id FROM tracks WHERE id=?`
		args = []any{p.From}
	case "album":
		tracks = `SELECT id FROM tracks WHERE release_id=?`
		args = []any{p.From}
	case "artist":
		// Album-level artist relationships also affect listeners of those tracks.
		tracks = `SELECT track_id FROM artist_tracks WHERE artist_id=? UNION SELECT id FROM tracks WHERE release_id IN (SELECT release_id FROM artist_releases WHERE artist_id=?)`
		args = []any{p.From, p.From}

	}
	count := func(q string, a ...any) (int64, error) {
		var n int64
		e := tx.QueryRowContext(ctx, q, a...).Scan(&n)
		return n, e
	}
	trackRows, e := tx.QueryContext(ctx, `SELECT id FROM tracks WHERE id IN (`+tracks+`) ORDER BY id`, args...)
	if e != nil {
		return p, e
	}
	for trackRows.Next() {
		var id int32
		if e = trackRows.Scan(&id); e != nil {
			trackRows.Close()
			return p, e
		}
		p.AffectedTrackIDs = append(p.AffectedTrackIDs, id)
	}
	if e = trackRows.Err(); e != nil {
		trackRows.Close()
		return p, e
	}
	trackRows.Close()
	p.SourceTracks, err = count(`SELECT COUNT(*) FROM (`+tracks+`)`, args...)
	if err != nil {
		return p, err
	}
	p.SourceListens, err = count(`SELECT COUNT(*) FROM listens WHERE track_id IN (`+tracks+`)`, args...)
	if err != nil {
		return p, err
	}
	switch p.Kind {
	case "artist":
		p.SourceAlbums, err = count(`SELECT COUNT(*) FROM artist_releases WHERE artist_id=?`, p.From)
	case "album":
		p.SourceAlbums = 1
	case "track":
		p.SourceAlbums = 1
	}
	if err != nil {
		return p, err
	}
	rows, err := tx.QueryContext(ctx, `SELECT p.id,p.display_name,COUNT(*) FROM listens l JOIN listening_profiles p ON p.id=l.profile_id WHERE l.track_id IN (`+tracks+`) GROUP BY p.id,p.display_name ORDER BY p.id`, args...)
	if err != nil {
		return p, err
	}
	for rows.Next() {
		var x db.CatalogImpactProfile
		if err = rows.Scan(&x.ID, &x.Name, &x.Listens); err != nil {
			rows.Close()
			return p, err
		}
		p.Profiles = append(p.Profiles, x)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return p, err
	}
	rows.Close()
	if p.Action == "merge" {
		var destinationTracks string
		var destArgs []any
		switch p.Kind {
		case "artist":
			destinationTracks = `SELECT track_id FROM artist_tracks WHERE artist_id=? UNION SELECT id FROM tracks WHERE release_id IN (SELECT release_id FROM artist_releases WHERE artist_id=?)`
			destArgs = []any{p.To, p.To}
		case "album":
			destinationTracks = `SELECT id FROM tracks WHERE release_id=?`
			destArgs = []any{p.To}
		case "track":
			destinationTracks = `SELECT id FROM tracks WHERE release_id=(SELECT release_id FROM tracks WHERE id=?)`
			destArgs = []any{p.To}
		}
		destRows, e := tx.QueryContext(ctx, `SELECT p.id,p.display_name,COUNT(*) FROM listens l JOIN listening_profiles p ON p.id=l.profile_id WHERE l.track_id IN (`+destinationTracks+`) GROUP BY p.id,p.display_name ORDER BY p.id`, destArgs...)
		if e != nil {
			return p, e
		}
		for destRows.Next() {
			var x db.CatalogImpactProfile
			if e = destRows.Scan(&x.ID, &x.Name, &x.Listens); e != nil {
				destRows.Close()
				return p, e
			}
			p.DestinationProfiles = append(p.DestinationProfiles, x)
		}
		if e = destRows.Err(); e != nil {
			destRows.Close()
			return p, e
		}
		destRows.Close()
	}
	if p.Action == "delete" {
		if p.Kind == "artist" {
			p.RemovedAlbums, err = count(`SELECT COUNT(*) FROM artist_releases ar WHERE ar.artist_id=? AND NOT EXISTS (SELECT 1 FROM artist_releases other WHERE other.release_id=ar.release_id AND other.artist_id<>?)`, p.From, p.From)
		} else if p.Kind == "album" {
			p.RemovedAlbums = 1
		}
		if err != nil {
			return p, err
		}
		removed := tracks
		removedArgs := args
		if p.Kind == "artist" {
			removed = `SELECT id FROM tracks WHERE release_id IN (SELECT ar.release_id FROM artist_releases ar WHERE ar.artist_id=? AND NOT EXISTS (SELECT 1 FROM artist_releases other WHERE other.release_id=ar.release_id AND other.artist_id<>?))`
			removedArgs = []any{p.From, p.From}
		}
		p.RemovedTracks, err = count(`SELECT COUNT(*) FROM (`+removed+`)`, removedArgs...)
		if err != nil {
			return p, err
		}
		p.RemovedListens, err = count(`SELECT COUNT(*) FROM listens WHERE track_id IN (`+removed+`)`, removedArgs...)
		if err != nil {
			return p, err
		}
	}
	if p.Action == "merge" && p.Kind == "track" {
		p.DuplicateListens, err = count(`SELECT COUNT(*) FROM listens a JOIN listens b ON b.profile_id=a.profile_id AND b.listened_at=a.listened_at WHERE a.track_id=? AND b.track_id=?`, p.From, p.To)
		if err != nil {
			return p, err
		}
	}
	p.Fingerprint, err = catalogFingerprint(ctx, tx, p)
	return p, err
}

// PreviewCatalogMutation reads review information in one SQLite transaction.
func (s *Sqlite) PreviewCatalogMutation(ctx context.Context, p db.CatalogPlan) (db.CatalogPlan, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return p, err
	}
	defer tx.Rollback()
	p, err = planCatalogIn(ctx, tx, p)
	if err != nil {
		return p, err
	}
	if err = tx.Commit(); err != nil {
		return p, err
	}
	return p, nil
}

// staleWriteConflict maps SQLite snapshot/write-lock races to an explicit re-review.
func staleWriteConflict(err error) error {
	var sqliteErr *sqlite3.Error
	if errors.As(err, &sqliteErr) && (sqliteErr.Code()&255 == 5 || sqliteErr.Code()&255 == 6) {
		return fmt.Errorf("%w: concurrent catalog change", db.ErrConflict)
	}
	return err
}

// ApplyCatalogMutation recalculates the review and changes the catalog inside the same transaction.
func (s *Sqlite) ApplyCatalogMutation(ctx context.Context, p db.CatalogPlan) error {
	if len(p.Fingerprint) != 64 || strings.TrimSpace(p.Fingerprint) != p.Fingerprint {
		return db.ErrInvalidInput
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	current, err := planCatalogIn(ctx, tx, p)
	if errors.Is(err, db.ErrNotFound) {
		return db.ErrConflict
	}
	if err != nil {
		return err
	}
	if current.Fingerprint != p.Fingerprint {
		return db.ErrConflict
	}
	if p.Action == "merge" {
		switch p.Kind {
		case "artist":
			err = mergeArtistsIn(ctx, tx, p.From, p.To, p.ReplaceImage)
		case "album":
			err = mergeAlbumsIn(ctx, tx, p.From, p.To, p.ReplaceImage)
		case "track":
			err = mergeTracksIn(ctx, tx, p.From, p.To)
		}
	} else {
		table := map[string]string{"artist": "artists", "album": "releases", "track": "tracks"}[p.Kind]
		var result sql.Result
		result, err = tx.ExecContext(ctx, `DELETE FROM `+table+` WHERE id=?`, p.From)
		if err == nil {
			var n int64
			n, err = result.RowsAffected()
			if err == nil && n != 1 {
				err = db.ErrConflict
			}
		}
	}
	if err != nil {
		return staleWriteConflict(fmt.Errorf("catalog operation: %w", err))
	}
	return staleWriteConflict(tx.Commit())
}
