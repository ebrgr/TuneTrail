package summary

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
)

type firstReader struct {
	unix    int64
	err     error
	profile int32
}

func (f *firstReader) GetFirstListenUnix(ctx context.Context) (int64, error) {
	f.profile = db.ProfileID(ctx)
	return f.unix, f.err
}
func TestSummaryCalendarDayCounts(t *testing.T) {
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	zones := map[string]*time.Location{}
	for _, name := range []string{"UTC", "America/New_York", "Australia/Lord_Howe", "America/Asuncion"} {
		loc, err := time.LoadLocation(name)
		if err != nil {
			t.Fatal(err)
		}
		zones[name] = loc
	}
	cases := []struct {
		year, month, week int
		zone              string
		days              int64
	}{{2026, 1, 0, "UTC", 31}, {2026, 2, 0, "UTC", 28}, {2024, 2, 0, "UTC", 29}, {2024, 0, 0, "UTC", 366}, {2026, 3, 0, "America/New_York", 31}, {2026, 11, 0, "America/New_York", 30}, {2026, 10, 0, "Australia/Lord_Howe", 31}, {2017, 10, 0, "America/Asuncion", 31}, {1, 1, 0, "UTC", 31}, {9999, 12, 0, "UTC", 31}, {9999, 0, 52, "UTC", 7}}
	for _, tc := range cases {
		tf := db.Timeframe{Year: tc.year, Month: tc.month, Week: tc.week, Timezone: zones[tc.zone]}
		resolved, days, err := resolveInterval(context.Background(), &firstReader{}, tf, now)
		if err != nil || days != tc.days || !resolved.Resolved {
			t.Fatalf("%+v: days=%d %v", tc, days, err)
		}
		a, b := db.TimeframeToTimeRangeAt(tf, now)
		x, y := db.TimeframeToTimeRangeAt(resolved, now.Add(24*time.Hour))
		if !a.Equal(x) || !b.Equal(y) {
			t.Fatal("query boundaries changed")
		}
	}
}
func TestSummaryGenericAndAllTimeIntervals(t *testing.T) {
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	from := time.Date(2026, 1, 1, 23, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		duration time.Duration
		days     int64
	}{{0, 1}, {time.Hour, 1}, {24 * time.Hour, 1}, {48 * time.Hour, 2}} {
		tf := db.Timeframe{From: from, To: from.Add(tc.duration), Year: 2024, Month: 2, Timezone: time.UTC}
		_, days, err := resolveInterval(context.Background(), &firstReader{}, tf, now)
		if err != nil || days != tc.days {
			t.Fatalf("explicit %v: %d %v", tc.duration, days, err)
		}
	}
	for _, tc := range []struct {
		tf   db.Timeframe
		days int64
	}{{db.Timeframe{}, 1}, {db.Timeframe{From: time.Date(1, 1, 1, 0, 0, 1, 0, time.UTC), To: time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)}, 3652057}} {
		_, days, err := resolveInterval(context.Background(), &firstReader{}, tc.tf, now)
		if err != nil || days != tc.days {
			t.Fatalf("long/empty interval: %d %v", days, err)
		}
	}
	if _, _, err := resolveInterval(context.Background(), &firstReader{}, db.Timeframe{From: now, To: now.Add(-time.Second)}, now); err == nil {
		t.Fatal("inverted interval accepted")
	}
	if _, _, err := resolveInterval(context.Background(), &firstReader{}, db.Timeframe{Resolved: true, From: now}, now); err == nil {
		t.Fatal("already resolved interval accepted")
	}
	for _, tc := range []struct {
		first int64
		err   error
		days  int64
	}{{0, fmt.Errorf("wrapped: %w", sql.ErrNoRows), 1}, {now.Unix(), nil, 1}, {now.AddDate(0, 0, -4).Unix(), nil, 5}, {now.AddDate(0, 0, 1).Unix(), nil, 1}} {
		reader := &firstReader{unix: tc.first, err: tc.err}
		ctx := db.WithProfile(context.Background(), 7)
		tf, days, err := resolveInterval(ctx, reader, db.Timeframe{Period: db.PeriodAllTime, Timezone: time.UTC}, now)
		if err != nil || days != tc.days || reader.profile != 7 || !tf.From.IsZero() || !tf.To.Equal(now) || !tf.Resolved {
			t.Fatalf("all time: %+v %d %v", tf, days, err)
		}
	}
	reader := &firstReader{err: errors.New("db failed")}
	if _, _, err := resolveInterval(context.Background(), reader, db.Timeframe{Period: db.PeriodAllTime}, now); err == nil {
		t.Fatal("first listen error hidden")
	}
	loc, _ := time.LoadLocation("America/New_York")
	reader = &firstReader{unix: time.Time{}.Unix()}
	if _, days, err := resolveInterval(context.Background(), reader, db.Timeframe{Period: db.PeriodAllTime, Timezone: loc}, now); err != nil || days < 700000 {
		t.Fatalf("year zero timezone boundary: %d %v", days, err)
	}
}

// TestImplicitMonthDayCountUsesLocalYear matches the interval's local leap year.
func TestImplicitMonthDayCountUsesLocalYear(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Kathmandu")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2023, 12, 31, 23, 30, 0, 0, time.UTC)
	tf := db.Timeframe{Month: 2, Timezone: loc}
	_, days, err := resolveInterval(context.Background(), &firstReader{}, tf, now)
	if err != nil || days != 28 {
		t.Fatalf("days=%d err=%v", days, err)
	}
}
