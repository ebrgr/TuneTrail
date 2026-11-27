package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/imageurl"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/google/uuid"
)

// boundedPlayedSeconds takes the optional measurement and the resolved occurrence ID.
// A missing catalog duration leaves the measurement intact, including for standalone occurrences.
// boundedPlayedSeconds stores a measured playback time capped at the catalog track length. NULLIF makes a zero
// (for example from an older export) unknown, and min() keeps NULL as NULL, so such a listen counts the whole track.
const boundedPlayedSeconds = `min(NULLIF(?,0), coalesce((SELECT duration FROM tracks WHERE id=? AND duration>0), 2147483647))`

// SaveManualListen inserts an explicitly selected occurrence unless that exact second already exists.
func (s *Sqlite) SaveManualListen(ctx context.Context, opts db.SaveListenOpts) (bool, error) {
	if opts.ProfileID <= 0 || opts.TrackID <= 0 || opts.Time.IsZero() {
		return false, db.ErrInvalidInput
	}
	var inserted bool
	err := s.withIngestWriter(ctx, func(tx *sql.Conn) error {
		// During the schema 20 transition, register provenance before the first manual write.
		if _, err := tx.ExecContext(ctx, `INSERT INTO event_origins VALUES('manual','Manual listen')
			ON CONFLICT(origin) DO NOTHING`); err != nil {
			return err
		}
		stamp := opts.Time.Unix()
		result, err := tx.ExecContext(ctx, `INSERT INTO listens(profile_id,track_id,listened_at,client,origin)
			SELECT ?,?,?,?,'manual' WHERE NOT EXISTS (
				SELECT 1 FROM listens WHERE profile_id=? AND track_id=? AND listened_at=?
			)`, opts.ProfileID, opts.TrackID, stamp, opts.Client, opts.ProfileID, opts.TrackID, stamp)
		if err != nil {
			return err
		}
		count, err := result.RowsAffected()
		inserted = count == 1
		return err
	})
	return inserted && err == nil, err
}

// DeleteListen removes scoped history, receipts and open relay items atomically.
func (s *Sqlite) DeleteListen(ctx context.Context, trackId int32, listenedAt time.Time) error {
	if trackId == 0 {
		return errors.New("DeleteListen: required parameter 'trackId' missing")
	}
	if trackId <= 0 || db.ProfileID(ctx) <= 0 {
		return db.ErrInvalidInput
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	profile := db.ProfileID(ctx)
	result, err := tx.ExecContext(ctx,
		`DELETE FROM listens WHERE track_id = ? AND listened_at = ? AND profile_id = ?`,
		trackId, listenedAt.Unix(), profile,
	)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return db.ErrNotFound
	}
	// The legacy tuple cannot identify a single event when several share a second.
	if n > 1 {
		return db.ErrConflict
	}
	if err := deleteListenReceipts(ctx, tx, profile, trackId, listenedAt); err != nil {
		return err
	}
	return tx.Commit()
}

// DeleteListenEvent addresses a single event within the authorized profile.
func (s *Sqlite) DeleteListenEvent(ctx context.Context, event string) error {
	profile := db.ProfileID(ctx)
	if profile <= 0 || event == "" {
		return db.ErrInvalidInput
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var track int32
	var at int64
	var origin string
	err = tx.QueryRowContext(ctx, `DELETE FROM listens WHERE profile_id=? AND event_id=?
		RETURNING track_id,listened_at,origin`, profile, event).Scan(&track, &at, &origin)
	if errors.Is(err, sql.ErrNoRows) {
		return db.ErrNotFound
	}
	if err != nil {
		return err
	}
	if origin == "legacy" {
		if err := deleteListenReceipts(ctx, tx, profile, track, time.Unix(at, 0)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// listenRow is an intermediate scan target used to decouple the main rows
// query from the per-row artistsForTrack sub-query. With MaxOpenConns(1),
// both the count query and the artistsForTrack call would deadlock if
// executed while the outer *sql.Rows is still holding the only connection.
type listenRow struct {
	listenedAt int64
	trackID    int32
	title      string
	client     string
	eventID    string
}

// GetListensPaginated includes the original client in each profile-scoped history record.
func (s *Sqlite) GetListensPaginated(ctx context.Context, opts db.GetItemsOpts) (*db.PaginatedResponse[*models.Listen], error) {
	if opts.Limit == 0 {
		opts.Limit = defaultItemsPerPage
	}
	offset := (opts.Page - 1) * opts.Limit
	t1, t2 := db.TimeframeToTimeRange(opts.Timeframe)

	// Use the same occurrence filters for history rows and the pagination count.
	from := ` FROM listens l JOIN occurrences o ON o.id = l.track_id`
	where := ` WHERE l.listened_at BETWEEN ? AND ?`
	args := []any{t1.Unix(), t2.Unix()}
	switch {
	case opts.TrackID > 0:
		where += " AND o.id = ?"
		args = append(args, opts.TrackID)
	case opts.AlbumID > 0:
		where += " AND o.release_id = ?"
		args = append(args, opts.AlbumID)
	case opts.ArtistID > 0:
		from += " JOIN artist_tracks at ON at.track_id = o.id"
		where += " AND at.artist_id = ?"
		args = append(args, opts.ArtistID)
	}

	// Management lookups can identify one event even when it shares a timestamp.
	if opts.EventID != "" {
		where += " AND l.event_id=?"
		args = append(args, opts.EventID)
	}

	var count int64
	if err := s.queryRowContext(ctx, "SELECT COUNT(*)"+from+where, args...).Scan(&count); err != nil {
		return nil, fmt.Errorf("count listens: %w", err)
	}

	rows, err := s.queryContext(ctx, `
		SELECT l.listened_at, l.track_id, COALESCE(o.presented_title, (
		    SELECT alias FROM track_aliases
		    WHERE track_id = o.legacy_track_id AND is_primary = 1 ORDER BY alias LIMIT 1
		), ''), l.client, l.event_id`+from+where+`
		ORDER BY l.listened_at DESC, l.track_id DESC, l.event_id DESC LIMIT ? OFFSET ?`,
		append(args, opts.Limit, offset)...)

	if err != nil {
		return nil, fmt.Errorf("GetListensPaginated: %w", err)
	}

	var raw []listenRow
	for rows.Next() {
		var r listenRow
		if err := rows.Scan(&r.listenedAt, &r.trackID, &r.title, &r.client, &r.eventID); err != nil {
			rows.Close()
			return nil, err
		}
		raw = append(raw, r)
	}
	// Explicitly close before calling artistsForTrack so the connection is free.
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	listens := make([]*models.Listen, 0, len(raw))
	for _, r := range raw {
		l := &models.Listen{
			Time:   time.Unix(r.listenedAt, 0).UTC(),
			Client: r.client, EventID: r.eventID,
			Track: models.SimpleTrack{
				ID:    r.trackID,
				Title: r.title,
			},
		}
		l.Track.Artists, err = s.artistsForTrack(ctx, r.trackID)
		if err != nil {
			return nil, err
		}
		imgid, err := s.imageForTrack(ctx, l.Track.ID)
		if err != nil {
			return nil, err
		}
		l.Track.Image = imageurl.List(ctx, imgid)
		listens = append(listens, l)
	}

	return &db.PaginatedResponse[*models.Listen]{
		Items:        listens,
		TotalCount:   count,
		ItemsPerPage: int32(opts.Limit),
		HasNextPage:  int64(offset+len(listens)) < count,
		CurrentPage:  int32(opts.Page),
	}, nil
}

// imageForTrack uses only the release of the displayed occurrence.
func (s *Sqlite) imageForTrack(ctx context.Context, trackId int32) (*uuid.UUID, error) {
	row := s.queryRowContext(ctx, `
		SELECT r.image
		FROM releases r
		JOIN occurrences t ON r.id = t.release_id
		WHERE t.id = ? AND r.image IS NOT NULL`,
		trackId,
	)
	var imageid *uuid.UUID
	err := row.Scan(&imageid)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("imageForTrack: %w", err)
	}
	return imageid, nil
}

func (s *Sqlite) GetFirstListenUnix(ctx context.Context) (int64, error) {
	row := s.queryRowContext(ctx, `
		SELECT listened_at FROM listens ORDER BY listened_at ASC LIMIT 1;`)
	var unix int64
	err := row.Scan(&unix)
	// An empty history is reported with the error of the store contract.
	if errors.Is(err, sql.ErrNoRows) {
		err = db.ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("GetFirstListenUnix: %w", err)
	}

	return unix, nil
}
