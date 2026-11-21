package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ebrgr/TuneTrail/internal/catalog"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/google/uuid"
)

func (s *Sqlite) SaveListen(ctx context.Context, opts db.SaveListenOpts) error {
	_, err := s.SaveListenWithResult(ctx, opts)
	return err
}

// SaveListenWithResult reports whether the profile-scoped listen was actually inserted.
func (s *Sqlite) SaveListenWithResult(ctx context.Context, opts db.SaveListenOpts) (bool, error) {
	if opts.TrackID == 0 {
		return false, errors.New("SaveListen: required parameter TrackID missing")
	}
	if opts.Time.IsZero() {
		opts.Time = time.Now()
	}
	profileID, err := s.resolveWriteProfile(ctx, opts.ProfileID, opts.UserID)
	if err != nil {
		return false, fmt.Errorf("SaveListen: %w", err)
	}
	client := ""
	if opts.Client != "" {
		client = opts.Client
	}
	result, err := s.db.ExecContext(ctx,
		`INSERT INTO listens (track_id, listened_at, user_id, profile_id, client) VALUES (?,?,?,?,?)
         ON CONFLICT (profile_id, track_id, listened_at) DO NOTHING`,
		opts.TrackID, opts.Time.Unix(), nullableLegacyUserID(opts.UserID), profileID, client,
	)
	// Manual legacy writes cannot bypass a converted profile's admission.
	if isOccurrenceGuard(err, "Legacy listen conflicts") {
		return false, fmt.Errorf("SaveListen: %w: %w", db.ErrConflict, err)
	}
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n > 0, err
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
	if err := deleteListenReceipts(ctx, tx, profile, trackId, listenedAt); err != nil {
		return err
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

	var count int64
	if err := s.queryRowContext(ctx, "SELECT COUNT(*)"+from+where, args...).Scan(&count); err != nil {
		return nil, fmt.Errorf("count listens: %w", err)
	}

	rows, err := s.queryContext(ctx, `
		SELECT l.listened_at, l.track_id, COALESCE(o.presented_title, (
		    SELECT alias FROM track_aliases
		    WHERE track_id = o.legacy_track_id AND is_primary = 1 ORDER BY alias LIMIT 1
		), ''), l.client`+from+where+`
		ORDER BY l.listened_at DESC, l.track_id DESC, l.event_id DESC LIMIT ? OFFSET ?`,
		append(args, opts.Limit, offset)...)

	if err != nil {
		return nil, fmt.Errorf("GetListensPaginated: %w", err)
	}

	var raw []listenRow
	for rows.Next() {
		var r listenRow
		if err := rows.Scan(&r.listenedAt, &r.trackID, &r.title, &r.client); err != nil {
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
			Client: r.client,
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
		l.Track.Image = catalog.BuildImageList(ctx, imgid)
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
	if err != nil {
		return 0, fmt.Errorf("GetFirstListenUnix: %w", err)
	}

	return unix, nil
}
