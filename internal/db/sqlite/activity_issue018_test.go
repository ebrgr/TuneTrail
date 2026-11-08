package sqlite

import (
	"context"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

// TestActivityFractionalOffsetAndFinalSecond checks actual civil dates and inclusive month end.
func TestActivityFractionalOffsetAndFinalSecond(t *testing.T) {
	s, _, _ := migrationDB(t, 3)
	seedMediaRead(t, s)
	execSQL(t, s, `INSERT INTO listening_profiles(id,slug,display_name,created_at) VALUES(1,'alpha','Alpha',1),(2,'beta','Beta',1)`)
	loc, err := time.LoadLocation("Asia/Kathmandu")
	require.NoError(t, err)
	for _, d := range []time.Time{time.Date(2024, 5, 1, 0, 5, 0, 0, loc), time.Date(2024, 5, 31, 23, 59, 59, 0, loc)} {
		require.NoError(t, s.SaveListen(context.Background(), db.SaveListenOpts{ProfileID: 1, TrackID: 1, Time: d}))
		require.NoError(t, s.SaveListen(context.Background(), db.SaveListenOpts{ProfileID: 2, TrackID: 1, Time: d}))
	}
	for _, opt := range []db.ListenActivityOpts{{}, {ArtistID: 1}, {AlbumID: 1}, {TrackID: 1}} {
		opt.Year = 2024
		opt.Month = 5
		opt.Timezone = loc
		items, e := s.GetListenActivity(db.WithProfile(context.Background(), 1), opt)
		require.NoError(t, e)
		require.Len(t, items, 2)
		require.Equal(t, "2024-05-01", items[0].Start.In(loc).Format("2006-01-02"))
		require.EqualValues(t, 1, items[0].Listens)
		require.Equal(t, "2024-05-31", items[1].Start.In(loc).Format("2006-01-02"))
		require.EqualValues(t, 1, items[1].Listens)
	}
}

// TestActiveDaysUsesCivilDatesNotUTCDays checks dates straddling local midnight.
func TestActiveDaysUsesCivilDatesNotUTCDays(t *testing.T) {
	s, _, _ := migrationDB(t, 3)
	seedMediaRead(t, s)
	execSQL(t, s, `INSERT INTO listening_profiles(id,slug,display_name,created_at) VALUES(1,'alpha','Alpha',1)`)
	loc, err := time.LoadLocation("Asia/Kathmandu")
	require.NoError(t, err)
	for _, d := range []time.Time{time.Date(2024, 4, 30, 23, 55, 0, 0, loc), time.Date(2024, 5, 1, 0, 5, 0, 0, loc)} {
		require.NoError(t, s.SaveListen(context.Background(), db.SaveListenOpts{ProfileID: 1, TrackID: 1, Time: d}))
	}
	days, err := s.GetActiveDays(db.WithProfile(context.Background(), 1), loc)
	require.NoError(t, err)
	require.Equal(t, 2, days)
}

// TestCivilStreakAndBoundsPreserveDatesThroughMidnightDST checks fixed clocks and profile scope.
func TestCivilStreakAndBoundsPreserveDatesThroughMidnightDST(t *testing.T) {
	s, _, _ := migrationDB(t, 3)
	seedMediaRead(t, s)
	execSQL(t, s, `INSERT INTO listening_profiles(id,slug,display_name,created_at) VALUES(1,'alpha','Alpha',1),(2,'beta','Beta',1)`)
	loc, err := time.LoadLocation("America/Sao_Paulo")
	require.NoError(t, err)
	for _, d := range []time.Time{time.Date(2018, 11, 3, 23, 59, 59, 0, loc), time.Date(2018, 11, 4, 1, 0, 0, 0, loc), time.Date(2018, 11, 5, 0, 0, 0, 0, loc)} {
		require.NoError(t, s.SaveListen(context.Background(), db.SaveListenOpts{ProfileID: 1, TrackID: 1, Time: d}))
	}
	require.NoError(t, s.SaveListen(context.Background(), db.SaveListenOpts{ProfileID: 2, TrackID: 1, Time: time.Date(2018, 11, 4, 1, 0, 0, 0, loc)}))
	now := time.Date(2018, 11, 4, 12, 0, 0, 0, loc)
	ctx := db.WithProfile(context.Background(), 1)
	days, err := s.getActiveDaysAt(ctx, loc, now)
	require.NoError(t, err)
	require.Equal(t, 2, days)
	opts := db.ListenActivityOpts{Year: 2018, Month: 11, Timezone: loc}
	items, err := s.GetListenActivity(ctx, opts)
	require.NoError(t, err)
	require.Len(t, items, 3)
	require.Equal(t, "2018-11-04 01:00:00", items[1].Start.In(loc).Format("2006-01-02 15:04:05"))
	streak, err := s.getListenStreakAt(ctx, opts, now)
	require.NoError(t, err)
	require.Equal(t, 2, streak)
	longest, err := s.getLongestListenStreakAt(ctx, opts, now)
	require.NoError(t, err)
	require.Equal(t, 2, longest)
	other, err := s.getActiveDaysAt(db.WithProfile(context.Background(), 2), loc, now)
	require.NoError(t, err)
	require.Equal(t, 1, other)
}

// TestActivityIncludesOnlyCalendarBoundaries checks start−1, start, end and end+1.
func TestActivityIncludesOnlyCalendarBoundaries(t *testing.T) {
	for _, zone := range []string{"UTC", "America/New_York", "Asia/Kathmandu", "Australia/Lord_Howe"} {
		t.Run(zone, func(t *testing.T) {
			s, _, _ := migrationDB(t, 3)
			seedMediaRead(t, s)
			execSQL(t, s, `INSERT INTO listening_profiles(id,slug,display_name,created_at) VALUES(1,'alpha','Alpha',1)`)
			loc, err := time.LoadLocation(zone)
			require.NoError(t, err)
			start := time.Date(2024, 3, 1, 0, 0, 0, 0, loc)
			next := time.Date(2024, 4, 1, 0, 0, 0, 0, loc)
			for _, d := range []time.Time{start.Add(-time.Second), start, next.Add(-time.Second), next} {
				require.NoError(t, s.SaveListen(context.Background(), db.SaveListenOpts{ProfileID: 1, TrackID: 1, Time: d}))
			}
			items, err := s.GetListenActivity(db.WithProfile(context.Background(), 1), db.ListenActivityOpts{Year: 2024, Month: 3, Timezone: loc})
			require.NoError(t, err)
			var total int64
			for _, item := range items {
				total += item.Listens
			}
			require.EqualValues(t, 2, total)
		})
	}
}

// TestStreakIncludesRepeatedLastHour checks the second occurrence before local midnight.
func TestStreakIncludesRepeatedLastHour(t *testing.T) {
	s, _, _ := migrationDB(t, 3)
	seedMediaRead(t, s)
	execSQL(t, s, `INSERT INTO listening_profiles(id,slug,display_name,created_at) VALUES(1,'alpha','Alpha',1)`)
	loc, err := time.LoadLocation("America/Sao_Paulo")
	require.NoError(t, err)
	listen := time.Date(2019, 2, 17, 2, 30, 0, 0, time.UTC)
	require.Equal(t, "2019-02-16 23:30 -0300", listen.In(loc).Format("2006-01-02 15:04 -0700"))
	require.NoError(t, s.SaveListen(context.Background(), db.SaveListenOpts{ProfileID: 1, TrackID: 1, Time: listen}))
	now := time.Date(2019, 2, 16, 12, 0, 0, 0, loc)
	ctx := db.WithProfile(context.Background(), 1)
	opts := db.ListenActivityOpts{Timezone: loc}
	streak, err := s.getListenStreakAt(ctx, opts, now)
	require.NoError(t, err)
	require.Equal(t, 1, streak)
	longest, err := s.getLongestListenStreakAt(ctx, opts, now)
	require.NoError(t, err)
	require.Equal(t, 1, longest)
}

// TestFractionalOffsetsAndNegativeUnixTimestamps preserves each original second's civil date.
func TestFractionalOffsetsAndNegativeUnixTimestamps(t *testing.T) {
	for _, tc := range []struct {
		zone        string
		year, month int
	}{{"Australia/Lord_Howe", 2024, 10}, {"Asia/Kathmandu", 2024, 5}, {"UTC", 1969, 12}} {
		t.Run(tc.zone, func(t *testing.T) {
			s, _, _ := migrationDB(t, 3)
			seedMediaRead(t, s)
			execSQL(t, s, `INSERT INTO listening_profiles(id,slug,display_name,created_at) VALUES(1,'alpha','Alpha',1)`)
			loc, err := time.LoadLocation(tc.zone)
			require.NoError(t, err)
			for _, d := range []time.Time{time.Date(tc.year, time.Month(tc.month), 1, 23, 55, 0, 0, loc), time.Date(tc.year, time.Month(tc.month), 2, 0, 5, 0, 0, loc)} {
				require.NoError(t, s.SaveListen(context.Background(), db.SaveListenOpts{ProfileID: 1, TrackID: 1, Time: d}))
			}
			got, err := s.GetListenActivity(db.WithProfile(context.Background(), 1), db.ListenActivityOpts{Year: tc.year, Month: tc.month, Timezone: loc})
			require.NoError(t, err)
			require.Len(t, got, 2)
			require.Equal(t, 1, got[0].Start.In(loc).Day())
			require.Equal(t, 2, got[1].Start.In(loc).Day())
			require.EqualValues(t, 1, got[0].Listens)
			require.EqualValues(t, 1, got[1].Listens)
		})
	}
}
