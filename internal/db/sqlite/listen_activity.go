package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
)

// GetListenActivity counts original seconds in each requested local civil day.
func (s *Sqlite) GetListenActivity(ctx context.Context, opts db.ListenActivityOpts) ([]db.ListenActivityItem, error) {
	if opts.Month != 0 && opts.Year == 0 {
		return nil, errors.New("GetListenActivity: year must be specified with month")
	}
	if opts.Range == 0 {
		opts.Range = db.DefaultRange
	}

	t1, t2 := db.ListenActivityOptsToTimes(opts)

	loc := opts.Timezone
	if loc == nil {
		loc = time.UTC
	}

	var rows *sql.Rows
	var err error

	switch {
	case opts.ArtistID > 0:
		rows, err = s.queryContext(ctx, `
			SELECT l.listened_at AS hour_bucket, COUNT(*) AS listen_count
			FROM listens l
			JOIN artist_tracks at2 ON l.track_id = at2.track_id
			WHERE l.listened_at >= ? AND l.listened_at <= ? AND at2.artist_id = ?
			GROUP BY hour_bucket`,
			t1.Unix(), t2.Unix(), opts.ArtistID,
		)
	case opts.AlbumID > 0:
		rows, err = s.queryContext(ctx, `
			SELECT l.listened_at AS hour_bucket, COUNT(*) AS listen_count
			FROM listens l
			JOIN tracks t ON l.track_id = t.id
			WHERE l.listened_at >= ? AND l.listened_at <= ? AND t.release_id = ?
			GROUP BY hour_bucket`,
			t1.Unix(), t2.Unix(), opts.AlbumID,
		)
	case opts.TrackID > 0:
		rows, err = s.queryContext(ctx, `
			SELECT listened_at AS hour_bucket, COUNT(*) AS listen_count
			FROM listens
			WHERE listened_at >= ? AND listened_at <= ? AND track_id = ?
			GROUP BY hour_bucket`,
			t1.Unix(), t2.Unix(), opts.TrackID,
		)
	default:
		rows, err = s.queryContext(ctx, `
			SELECT listened_at AS hour_bucket, COUNT(*) AS listen_count
			FROM listens
			WHERE listened_at >= ? AND listened_at <= ?
			GROUP BY hour_bucket`,
			t1.Unix(), t2.Unix(),
		)
	}

	if err != nil {
		return nil, fmt.Errorf("GetListenActivity: %w", err)
	}
	defer rows.Close()

	countsByDay := make(map[time.Time]int64)
	dayStarts := make(map[time.Time]time.Time)

	for rows.Next() {
		var hourUnix int64
		var count int64
		if err := rows.Scan(&hourUnix, &count); err != nil {
			return nil, err
		}

		// Group original seconds by local civil date, including fractional offsets.
		t := time.Unix(hourUnix, 0).In(loc)
		day := db.CivilDayKey(t)
		countsByDay[day] += count
		if _, ok := dayStarts[day]; !ok {
			dayStarts[day] = db.CivilDayStart(t)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Convert the map to the final slice and sort it chronologically
	activity := make([]db.ListenActivityItem, 0, len(countsByDay))
	for day, count := range countsByDay {
		activity = append(activity, db.ListenActivityItem{
			Start:   dayStarts[day],
			Listens: count,
		})
	}

	sort.Slice(activity, func(i, j int) bool {
		return activity[i].Start.Before(activity[j].Start)
	})

	return activity, nil
}

// GetListenStreak calculates consecutive civil dates in the requested timezone.
func (s *Sqlite) GetListenStreak(ctx context.Context, opts db.ListenActivityOpts) (int, error) {
	// Honor the response's frozen clock when available.
	now := opts.Now
	if now.IsZero() {
		now = time.Now()
	}
	return s.getListenStreakAt(ctx, opts, now)
}

// getListenStreakAt fixes the clock for a single streak calculation.
func (s *Sqlite) getListenStreakAt(ctx context.Context, opts db.ListenActivityOpts, now time.Time) (int, error) {
	loc := opts.Timezone
	if loc == nil {
		loc = time.UTC
	}

	now = now.In(loc)
	// The next local day is an exclusive boundary, including repeated late hours.
	endOfToday := db.CivilDayStart(time.Date(now.Year(), now.Month(), now.Day()+1, 12, 0, 0, 0, loc))

	var rows *sql.Rows
	var err error

	switch {
	case opts.ArtistID > 0:
		rows, err = s.queryContext(ctx, `
			SELECT l.listened_at AS hour_bucket
			FROM listens l
			JOIN artist_tracks at2 ON l.track_id = at2.track_id
			WHERE l.listened_at < ? AND at2.artist_id = ?
			GROUP BY hour_bucket
			ORDER BY hour_bucket DESC`,
			endOfToday.Unix(), opts.ArtistID,
		)
	case opts.AlbumID > 0:
		rows, err = s.queryContext(ctx, `
			SELECT l.listened_at AS hour_bucket
			FROM listens l
			JOIN tracks t ON l.track_id = t.id
			WHERE l.listened_at < ? AND t.release_id = ?
			GROUP BY hour_bucket
			ORDER BY hour_bucket DESC`,
			endOfToday.Unix(), opts.AlbumID,
		)
	case opts.TrackID > 0:
		rows, err = s.queryContext(ctx, `
			SELECT listened_at AS hour_bucket
			FROM listens
			WHERE listened_at < ? AND track_id = ?
			GROUP BY hour_bucket
			ORDER BY hour_bucket DESC`,
			endOfToday.Unix(), opts.TrackID,
		)
	default:
		rows, err = s.queryContext(ctx, `
			SELECT listened_at AS hour_bucket
			FROM listens
			WHERE listened_at < ?
			GROUP BY hour_bucket
			ORDER BY hour_bucket DESC`,
			endOfToday.Unix(),
		)
	}
	if err != nil {
		return 0, fmt.Errorf("GetListenStreak: %w", err)
	}
	defer rows.Close()

	daySet := make(map[time.Time]struct{})
	for rows.Next() {
		var hourUnix int64
		if err := rows.Scan(&hourUnix); err != nil {
			return 0, fmt.Errorf("GetListenStreak: %w", err)
		}
		t := time.Unix(hourUnix, 0).In(loc)
		day := db.CivilDayKey(t)
		daySet[day] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	days := make([]time.Time, 0, len(daySet))
	for day := range daySet {
		days = append(days, day)
	}
	sort.Slice(days, func(i, j int) bool {
		return days[i].After(days[j])
	})

	if len(days) == 0 {
		return 0, nil
	}

	today := db.CivilDayKey(now)
	yesterday := today.AddDate(0, 0, -1)
	if days[0].Before(yesterday) {
		return 0, nil
	}

	streak := 0
	expected := days[0]
	for _, day := range days {
		if !day.Equal(expected) {
			break
		}
		streak++
		expected = expected.AddDate(0, 0, -1)
	}

	return streak, nil
}

// GetActiveDays counts distinct local civil dates through the end of the current local day.
func (s *Sqlite) GetActiveDays(ctx context.Context, tz *time.Location) (int, error) {
	return s.getActiveDaysAt(ctx, tz, time.Now())
}

// getActiveDaysAt fixes the clock for a single active-day calculation.
func (s *Sqlite) getActiveDaysAt(ctx context.Context, tz *time.Location, now time.Time) (int, error) {
	if tz == nil {
		tz = time.UTC
	}
	now = now.In(tz)
	// Midnight following today is an exclusive bound, including the last second.
	end := time.Date(now.Year(), now.Month(), now.Day()+1, 12, 0, 0, 0, tz)
	end = db.CivilDayStart(end)
	rows, err := s.queryContext(ctx, `SELECT DISTINCT listened_at FROM listens WHERE listened_at < ?`, end.Unix())
	if err != nil {
		return 0, fmt.Errorf("GetActiveDays: %w", err)
	}
	defer rows.Close()
	days := make(map[time.Time]struct{})
	for rows.Next() {
		var stamp int64
		if err := rows.Scan(&stamp); err != nil {
			return 0, fmt.Errorf("GetActiveDays: %w", err)
		}
		days[db.CivilDayKey(time.Unix(stamp, 0).In(tz))] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("GetActiveDays: %w", err)
	}
	return len(days), nil
}

// GetLongestListenStreak compares civil dates independently of DST transitions.
func (s *Sqlite) GetLongestListenStreak(ctx context.Context, opts db.ListenActivityOpts) (int, error) {
	// Honor the caller's frozen clock when available.
	now := opts.Now
	if now.IsZero() {
		now = time.Now()
	}
	return s.getLongestListenStreakAt(ctx, opts, now)
}

// getLongestListenStreakAt fixes the clock for a longest-streak calculation.
func (s *Sqlite) getLongestListenStreakAt(ctx context.Context, opts db.ListenActivityOpts, now time.Time) (int, error) {
	loc := opts.Timezone
	if loc == nil {
		loc = time.UTC
	}

	now = now.In(loc)
	// The next local day is an exclusive boundary, including repeated late hours.
	endOfToday := db.CivilDayStart(time.Date(now.Year(), now.Month(), now.Day()+1, 12, 0, 0, 0, loc))

	var rows *sql.Rows
	var err error

	switch {
	case opts.ArtistID > 0:
		rows, err = s.queryContext(ctx, `
			SELECT l.listened_at AS hour_bucket
			FROM listens l
			JOIN artist_tracks at2 ON l.track_id = at2.track_id
			WHERE l.listened_at < ? AND at2.artist_id = ?
			GROUP BY hour_bucket
			ORDER BY hour_bucket DESC`,
			endOfToday.Unix(), opts.ArtistID,
		)
	case opts.AlbumID > 0:
		rows, err = s.queryContext(ctx, `
			SELECT l.listened_at AS hour_bucket
			FROM listens l
			JOIN tracks t ON l.track_id = t.id
			WHERE l.listened_at < ? AND t.release_id = ?
			GROUP BY hour_bucket
			ORDER BY hour_bucket DESC`,
			endOfToday.Unix(), opts.AlbumID,
		)
	case opts.TrackID > 0:
		rows, err = s.queryContext(ctx, `
			SELECT listened_at AS hour_bucket
			FROM listens
			WHERE listened_at < ? AND track_id = ?
			GROUP BY hour_bucket
			ORDER BY hour_bucket DESC`,
			endOfToday.Unix(), opts.TrackID,
		)
	default:
		rows, err = s.queryContext(ctx, `
			SELECT listened_at AS hour_bucket
			FROM listens
			WHERE listened_at < ?
			GROUP BY hour_bucket
			ORDER BY hour_bucket DESC`,
			endOfToday.Unix(),
		)
	}
	if err != nil {
		return 0, fmt.Errorf("GetListenStreak: %w", err)
	}
	defer rows.Close()

	daySet := make(map[time.Time]struct{})
	for rows.Next() {
		var hourUnix int64
		if err := rows.Scan(&hourUnix); err != nil {
			return 0, fmt.Errorf("GetLongestListenStreak: %w", err)
		}
		t := time.Unix(hourUnix, 0).In(loc)
		day := db.CivilDayKey(t)
		daySet[day] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	days := make([]time.Time, 0, len(daySet))
	for day := range daySet {
		days = append(days, day)
	}
	sort.Slice(days, func(i, j int) bool {
		return days[i].After(days[j])
	})

	longest := 0
	current := 0
	for i, day := range days {
		if i == 0 {
			current = 1
			continue
		}
		if days[i-1].AddDate(0, 0, -1).Equal(day) {
			current++
		} else {
			if current > longest {
				longest = current
			}
			current = 1
		}
	}
	if current > longest {
		longest = current
	}

	return longest, nil
}
