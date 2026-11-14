package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	migrations_sqlite "github.com/ebrgr/TuneTrail/db/migrations_sqlite"
	"github.com/ebrgr/TuneTrail/internal/cfg"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/google/uuid"
	_ "modernc.org/sqlite"

	"github.com/pressly/goose/v3"
)

const defaultItemsPerPage = 20

type Sqlite struct {
	db *sql.DB
}

func New() (*Sqlite, error) {
	return newDatabase(true)
}

// NewExisting refuses to create a database when resetting credentials.
func NewExisting() (*Sqlite, error) {
	return newDatabase(false)
}

// newDatabase selects and validates the persistent database before use.
func newDatabase(allowCreate bool) (*Sqlite, error) {
	databasePath, err := bootstrap(cfg.ConfigDir(), allowCreate)
	if err != nil {
		return nil, fmt.Errorf("sqlite.New: bootstrap: %w", err)
	}
	dsn := sqliteURI(databasePath) + "?mode=rw&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite.New: open: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("sqlite.New: ping: %w", err)
	}

	return &Sqlite{db: db}, nil
}

// NewInMemory opens an isolated in-memory SQLite database and runs migrations.
// Each call produces an independent database, making it safe for parallel tests.
// Not intended for production use.
func NewInMemory() (*Sqlite, error) {
	// Named in-memory URIs with cache=shared let multiple sql.DB connections
	// share the same logical database within a process. A unique name per call
	// prevents parallel test instances from seeing each other's data.
	// SetMaxOpenConns(1) ensures goose and all subsequent queries use the same
	// underlying connection (required for in-memory databases to persist).
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_pragma=foreign_keys(ON)", uuid.New().String())
	sqldb, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite.NewInMemory: open: %w", err)
	}
	sqldb.SetMaxOpenConns(1)

	if err := sqldb.Ping(); err != nil {
		sqldb.Close()
		return nil, fmt.Errorf("sqlite.NewInMemory: ping: %w", err)
	}

	goose.SetBaseFS(migrations_sqlite.Files)
	goose.SetDialect("sqlite3")
	if err := goose.Up(sqldb, "."); err != nil {
		sqldb.Close()
		return nil, fmt.Errorf("sqlite.NewInMemory: goose: %w", err)
	}

	return &Sqlite{db: sqldb}, nil
}

// Not part of the DB interface this package implements. Only used for testing.
func (s *Sqlite) Exec(query string, args ...any) error {
	_, err := s.db.Exec(query, args...)
	return err
}

// Not part of the DB interface this package implements. Only used for testing.
func (s *Sqlite) RowExists(query string, args ...any) (bool, error) {
	var exists bool
	err := s.db.QueryRow(query, args...).Scan(&exists)
	return exists, err
}

func (s *Sqlite) Count(query string, args ...any) (count int, err error) {
	err = s.db.QueryRow(query, args...).Scan(&count)
	return
}

// Exposes db.QueryRow. Only used for testing. Not part of the DB interface this package implements.
func (s *Sqlite) QueryRow(query string, args ...any) *sql.Row {
	return s.db.QueryRow(query, args...)
}

func (s *Sqlite) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *Sqlite) Close(_ context.Context) {
	s.db.Close()
}

// artistsForTrack fetches artists for a track as []models.SimpleArtist,
// replacing the PG get_artists_for_track() function.
func (s *Sqlite) artistsForTrack(ctx context.Context, trackID int32) ([]models.SimpleArtist, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT a.id, awn.name, at2.is_primary
		FROM artist_tracks at2
		JOIN artists_with_name awn ON awn.id = at2.artist_id
		JOIN artists a ON a.id = at2.artist_id
		WHERE at2.track_id = ?
		ORDER BY at2.is_primary DESC, awn.name`,
		trackID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var artists []models.SimpleArtist
	for rows.Next() {
		var a models.SimpleArtist
		var isPrimary int
		if err := rows.Scan(&a.ID, &a.Name, &isPrimary); err != nil {
			return nil, err
		}
		artists = append(artists, a)
	}
	return artists, rows.Err()
}

// artistsForRelease fetches artists for a release as []models.SimpleArtist,
// replacing the PG get_artists_for_release() function.
func (s *Sqlite) artistsForRelease(ctx context.Context, releaseID int32) ([]models.SimpleArtist, error) {
	// Ordinary reads use the pool; transactional callers pass their transaction below.
	return artistsForReleaseIn(ctx, s.db, releaseID)
}

type catalogQuerier interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

// artistsForReleaseIn keeps merge reads on the same SQLite connection and snapshot.
func artistsForReleaseIn(ctx context.Context, q catalogQuerier, releaseID int32) ([]models.SimpleArtist, error) {
	return albumArtistsIn(ctx, q, releaseID, albumArtistRelations)
}

// albumArtistsIn selects live participation or persisted credits on the caller's snapshot.
func albumArtistsIn(ctx context.Context, q catalogQuerier, releaseID int32, relation string) ([]models.SimpleArtist, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT a.id, awn.name, ar.is_primary
		FROM `+relation+` ar
		JOIN artists_with_name awn ON awn.id = ar.artist_id
		JOIN artists a ON a.id = ar.artist_id
		WHERE ar.release_id = ?
		ORDER BY ar.is_primary DESC, awn.name`,
		releaseID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var artists []models.SimpleArtist
	for rows.Next() {
		var a models.SimpleArtist
		var isPrimary int
		if err := rows.Scan(&a.ID, &a.Name, &isPrimary); err != nil {
			return nil, err
		}
		artists = append(artists, a)
	}
	return artists, rows.Err()
}

// artistsWithAliasesForTrack fetches fully-detailed artists (with aliases) for a track.
// Used for export.
func (s *Sqlite) artistsWithAliasesForTrack(ctx context.Context, trackID int32) ([]models.ArtistWithFullAliases, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT a.id, awn.name, a.musicbrainz_id, a.image, a.image_source, at2.is_primary
		FROM artist_tracks at2
		JOIN artists_with_name awn ON awn.id = at2.artist_id
		JOIN artists a ON a.id = at2.artist_id
		WHERE at2.track_id = ?
		ORDER BY at2.is_primary DESC, awn.name`,
		trackID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var artists []models.ArtistWithFullAliases
	for rows.Next() {
		var a models.ArtistWithFullAliases
		var mbzID, image, imageSrc sql.NullString
		var primary bool
		if err := rows.Scan(&a.ID, &a.Name, &mbzID, &image, &imageSrc, &primary); err != nil {
			return nil, err
		}
		a.MbzID = parseNullableUUID(mbzID)
		a.Image = parseNullableUUID(image)
		a.ImageSource = imageSrc.String
		a.IsPrimary = primary

		artists = append(artists, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	for i := range artists {
		aliases, err := s.getAliasesForEntity(ctx, "artist_aliases", "artist_id", artists[i].ID)
		if err != nil {
			return nil, err
		}
		artists[i].Aliases = aliases
	}
	return artists, nil
}

// getAliasesForEntity fetches all aliases for an entity from the given alias table.
func (s *Sqlite) getAliasesForEntity(ctx context.Context, table, idCol string, id int32) ([]models.Alias, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT alias, source, is_primary FROM `+table+` WHERE `+idCol+` = ?`, id)
	if err != nil {
		return nil, err
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

// tzOffset returns the UTC offset in seconds for a time.Location.
// Used to convert UTC Unix timestamps to local calendar dates in SQL.
func tzOffset(loc *time.Location) int {
	if loc == nil {
		return 0
	}
	_, offset := time.Now().In(loc).Zone()
	return offset
}

// nullableUUID converts a *uuid.UUID to a sql.NullString for storage.
func nullableUUID(u *uuid.UUID) sql.NullString {
	if u == nil || *u == uuid.Nil {
		return sql.NullString{}
	}
	return sql.NullString{String: u.String(), Valid: true}
}

// parseNullableUUID converts a sql.NullString from storage back to *uuid.UUID.
func parseNullableUUID(s sql.NullString) *uuid.UUID {
	if !s.Valid || s.String == "" {
		return nil
	}
	u, err := uuid.Parse(s.String)
	if err != nil {
		return nil
	}
	return &u
}

func (s *Sqlite) PurgeAllData(ctx context.Context) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("PurgeAllData: BeginTx: %w", err)
	}
	defer tx.Rollback()
	// Order respects foreign-key dependencies; cascades clean up junction and
	// alias tables automatically.
	for _, stmt := range []string{
		`DELETE FROM listens`,
		`DELETE FROM tracks`,
		`DELETE FROM releases`,
		`DELETE FROM artists`,
	} {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("PurgeAllData: %w", err)
		}
	}
	return tx.Commit()
}

// compile-time assertion that *Sqlite implements db.DB
var _ db.DB = (*Sqlite)(nil)
