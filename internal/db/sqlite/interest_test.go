package sqlite

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
)

func assertInterestMembership(t *testing.T, stamps []int64, buckets []db.InterestBucket, now int64) {
	t.Helper()
	var sum int64
	for i, b := range buckets {
		if b.BucketStart.After(b.BucketEnd) {
			t.Fatal("reversed interval")
		}
		if i > 0 && !b.BucketStart.Equal(buckets[i-1].BucketEnd) {
			t.Fatal("gap or overlap")
		}
		var expected int64
		for _, stamp := range stamps {
			if stamp >= b.BucketStart.Unix() && (stamp < b.BucketEnd.Unix() || i == len(buckets)-1 && stamp == b.BucketEnd.Unix()) {
				expected++
			}
		}
		if b.ListenCount != expected {
			t.Fatalf("bucket %d: got %d want %d", i, b.ListenCount, expected)
		}
		sum += b.ListenCount
	}
	if sum != int64(len(stamps)) || buckets[len(buckets)-1].BucketEnd.Unix() != now {
		t.Fatal("lost listens or endpoint")
	}
}

func TestInterestBoundariesIncludeFinalSecondAndConserveCounts(t *testing.T) {
	for _, tc := range []struct {
		stamps []int64
		now    int64
		count  int
	}{
		{[]int64{100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110}, 110, 3},
		{[]int64{100, 100, 100}, 100, 16},
		{[]int64{100, 101}, 101, 16},
		{[]int64{-100, 0, 100}, 100, 1},
		{[]int64{-100, 0, 100}, 100, db.MaxInterestBuckets},
		{nil, 110, 16},
	} {
		result, err := buildInterestBuckets(context.Background(), tc.stamps, tc.count, tc.now)
		if err != nil || len(result) != tc.count {
			t.Fatalf("shape: %v", err)
		}
		assertInterestMembership(t, tc.stamps, result, tc.now)
	}
	// A wide span must not overflow when scaled by a bucket index.
	result, err := buildInterestBuckets(context.Background(), []int64{-4_000_000_000, 100}, 16, 100)
	if err != nil {
		t.Fatal(err)
	}
	assertInterestMembership(t, []int64{-4_000_000_000, 100}, result, 100)
}

func TestInterestRejectsInvalidCountsBeforeDatabaseAccess(t *testing.T) {
	s := &Sqlite{} // Invalid requests must return before dereferencing a database.
	for _, count := range []int{-1, 0, db.MaxInterestBuckets + 1, math.MaxInt} {
		if _, err := s.GetInterest(context.Background(), db.GetInterestOpts{TrackID: 1, Buckets: count}); err == nil {
			t.Fatalf("accepted %d", count)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := buildInterestBuckets(ctx, []int64{100}, 16, 110); !errors.Is(err, context.Canceled) {
		t.Fatal("cancellation lost")
	}
	if _, err := buildInterestBuckets(context.Background(), []int64{111}, 16, 110); err == nil {
		t.Fatal("future-only input not rejected by bucketer")
	}
}

func TestInterestSQLiteScopesEntitiesAndExcludesFuture(t *testing.T) {
	s, _, _ := migrationDB(t, 3)
	seedMediaRead(t, s)
	execSQL(t, s, `INSERT INTO listening_profiles(id,slug,display_name,created_at) VALUES(1,'alpha','Alpha',1),(2,'beta','Beta',1),(3,'empty','Empty',1),(4,'future','Future',1)`)
	base := time.Now().Add(-time.Hour).Truncate(time.Second)
	for _, v := range []struct {
		profile int32
		shift   time.Duration
	}{{1, 0}, {1, time.Second}, {1, 2 * time.Second}, {1, 24 * time.Hour}, {2, 0}, {4, 24 * time.Hour}} {
		if err := s.SaveListen(context.Background(), db.SaveListenOpts{ProfileID: v.profile, TrackID: 1, Time: base.Add(v.shift)}); err != nil {
			t.Fatal(err)
		}
	}
	for _, opts := range []db.GetInterestOpts{{ArtistID: 1}, {AlbumID: 1}, {TrackID: 1}} {
		opts.Buckets = 16
		for _, p := range []struct {
			id    int32
			total int64
		}{{1, 3}, {2, 1}, {3, 0}, {4, 0}} {
			before := time.Now().Unix()
			items, err := s.GetInterest(db.WithProfile(context.Background(), p.id), opts)
			after := time.Now().Unix()
			if err != nil || len(items) != 16 {
				t.Fatalf("query: %v", err)
			}
			var sum int64
			for _, b := range items {
				sum += b.ListenCount
				if b.BucketEnd.Unix() > after {
					t.Fatal("future endpoint")
				}
			}
			if sum != p.total {
				t.Fatalf("profile %d: got %d want %d", p.id, sum, p.total)
			}
			if end := items[15].BucketEnd.Unix(); end < before || end > after {
				t.Fatal("axis does not end at now")
			}
			if p.total == 0 && !items[0].BucketStart.Equal(items[15].BucketEnd) {
				t.Fatal("empty history invented a range")
			}
		}
	}
	ctx, cancel := context.WithCancel(db.WithProfile(context.Background(), 1))
	cancel()
	if _, err := s.GetInterest(ctx, db.GetInterestOpts{TrackID: 1, Buckets: 16}); !errors.Is(err, context.Canceled) {
		t.Fatalf("database cancellation: %v", err)
	}
}
