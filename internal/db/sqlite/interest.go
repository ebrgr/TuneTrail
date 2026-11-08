package sqlite

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
)

// GetInterest computes N evenly-spaced listen-count buckets spanning the entity's
// entire listen history. The PG implementation used generate_series + EXTRACT(EPOCH);
// here we fetch all timestamps in Go and do the bucketing in memory.
func (s *Sqlite) GetInterest(ctx context.Context, opts db.GetInterestOpts) ([]db.InterestBucket, error) {
	if err := db.ValidateInterestBuckets(opts.Buckets); err != nil {
		return nil, err
	}
	now := time.Now().Unix()

	var query string
	var arg any
	switch {
	case opts.ArtistID != 0:
		query = `
			SELECT l.listened_at FROM listens l
			JOIN tracks t ON t.id = l.track_id
			JOIN artist_tracks at2 ON at2.track_id = t.id
			WHERE at2.artist_id = ? AND l.listened_at <= ?
			ORDER BY l.listened_at`
		arg = opts.ArtistID
	case opts.AlbumID != 0:
		query = `
			SELECT l.listened_at FROM listens l
			JOIN tracks t ON t.id = l.track_id
			WHERE t.release_id = ? AND l.listened_at <= ?
			ORDER BY l.listened_at`
		arg = opts.AlbumID
	case opts.TrackID != 0:
		query = `
			SELECT listened_at FROM listens WHERE track_id = ? AND listened_at <= ? ORDER BY listened_at`
		arg = opts.TrackID
	default:
		return nil, errors.New("GetInterest: artist id, album id, or track id must be provided")
	}

	rows, err := s.queryContext(ctx, query, arg, now)
	if err != nil {
		return nil, fmt.Errorf("GetInterest: query: %w", err)
	}
	defer rows.Close()

	var timestamps []int64
	for rows.Next() {
		var t int64
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		timestamps = append(timestamps, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return buildInterestBuckets(ctx, timestamps, opts.Buckets, now)
}

// Input timestamps are sorted and no later than now. Every interval is half-open
// [start,end), except the last, which includes now. Repeated boundaries describe
// empty intervals, so even an instant-only history retains the requested shape.
func buildInterestBuckets(ctx context.Context, timestamps []int64, count int, now int64) ([]db.InterestBucket, error) {
	if err := db.ValidateInterestBuckets(count); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	start := now
	if len(timestamps) > 0 {
		start = timestamps[0]
	}
	if start > now {
		return nil, fmt.Errorf("interest timestamps exceed end")
	}
	// Unsigned subtraction and quotient/remainder avoid overflow for ancient
	// timestamps without multiplying the entire time span by the bucket count.
	span := uint64(now) - uint64(start)
	n := uint64(count)
	boundaries := make([]int64, count+1)
	for i := range boundaries {
		k := uint64(i)
		boundaries[i] = int64(uint64(start) + (span/n)*k + (span%n)*k/n)
	}
	buckets := make([]db.InterestBucket, count)
	for i := range buckets {
		buckets[i] = db.InterestBucket{BucketStart: time.Unix(boundaries[i], 0).UTC(), BucketEnd: time.Unix(boundaries[i+1], 0).UTC()}
	}
	for _, stamp := range timestamps {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if stamp < start || stamp > now {
			return nil, fmt.Errorf("interest timestamp outside interval")
		}
		index := sort.Search(count, func(i int) bool { return stamp < boundaries[i+1] || i == count-1 })
		buckets[index].ListenCount++
	}
	return buckets, nil
}
