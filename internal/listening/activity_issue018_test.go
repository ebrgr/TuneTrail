package listening

import (
	"github.com/ebrgr/TuneTrail/internal/db"
	"testing"
	"time"
)

// TestActivityFillsWeeklyAndYearlyRanges verifies bucket count and ISO alignment.
func TestActivityFillsWeeklyAndYearlyRanges(t *testing.T) {
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		step  db.StepInterval
		count int
	}{{db.StepDay, 12}, {db.StepWeek, 1}, {db.StepYear, 3}} {
		n := tc.count
		if tc.step == db.StepDay {
			n = 0
		}
		items := FillActivity(nil, db.ListenActivityOpts{Step: tc.step, Range: n, Timezone: time.UTC, Now: now})
		if len(items) != tc.count {
			t.Errorf("%s: got %d want %d", tc.step, len(items), tc.count)
		}
		if tc.step == db.StepWeek && len(items) > 0 && items[0].Start.Weekday() != time.Monday {
			t.Error("week must start Monday")
		}
		if tc.step == db.StepWeek && len(items) > 0 {
			if items[0].Start.After(now) || now.Sub(items[0].Start) >= 7*24*time.Hour {
				t.Error("range 1 must include the current ISO week")
			}
		}
	}
}

// TestFillActivityDoesNotDuplicateSkippedMidnight checks a historical DST month.
func TestFillActivityDoesNotDuplicateSkippedMidnight(t *testing.T) {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		t.Fatal(err)
	}
	items := []db.ListenActivityItem{{Start: time.Date(2018, 11, 4, 1, 0, 0, 0, loc), Listens: 3}}
	filled := FillActivity(items, db.ListenActivityOpts{Step: db.StepDay, Year: 2018, Month: 11, Timezone: loc, Now: time.Date(2018, 12, 1, 12, 0, 0, 0, loc)})
	if len(filled) != 30 {
		t.Fatalf("got %d days", len(filled))
	}
	for i, item := range filled {
		expected := time.Date(2018, 11, i+1, 12, 0, 0, 0, loc).Format("2006-01-02")
		if item.Start.Format("2006-01-02") != expected {
			t.Fatalf("%d: got %s want %s", i, item.Start, expected)
		}
		if i == 3 && item.Listens != 3 {
			t.Fatal("lost DST-day listens")
		}
	}
}
