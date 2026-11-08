package webui

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
)

type ActivityCell struct {
	Date    time.Time
	Count   int64
	Opacity string
	Future  bool
	Week    int
}

type ActivityGrid struct {
	Cells     []ActivityCell
	DayLabels []string
	Streak    int
	Timezone  string
}

type activityReader interface {
	GetListenActivity(context.Context, db.ListenActivityOpts) ([]db.ListenActivityItem, error)
	GetListenStreak(context.Context, db.ListenActivityOpts) (int, error)
}

// readActivity is shared by Home and media. Only the selected entity's filter
// is supplied by the caller; dates and intensity are presentation concerns.
func readActivity(ctx context.Context, store activityReader, opts db.ListenActivityOpts, weekStart int, target int64, now time.Time) (*ActivityGrid, error) {
	opts.Step, opts.Range, opts.Month, opts.Year = db.StepDay, 36*7, 0, 0
	// The database reads and graph share the same clock across calendar boundaries.
	opts.Now = now
	if opts.Timezone == nil {
		opts.Timezone = time.UTC
	}
	items, err := store.GetListenActivity(ctx, opts)
	if err != nil {
		return nil, err
	}
	streak, err := store.GetListenStreak(ctx, opts)
	if err != nil {
		return nil, err
	}
	return buildActivity(items, streak, opts.Timezone, weekStart, target, now)
}

// buildActivity iterates civil dates without normalizing skipped midnights to the previous day.
func buildActivity(items []db.ListenActivityItem, streak int, loc *time.Location, weekStart int, target int64, now time.Time) (*ActivityGrid, error) {
	if loc == nil || weekStart < 0 || weekStart > 6 || target <= 0 || streak < 0 {
		return nil, fmt.Errorf("invalid activity presentation options")
	}
	counts := map[string]int64{}
	for _, item := range items {
		if item.Listens < 0 {
			return nil, fmt.Errorf("negative activity count")
		}
		key := item.Start.In(loc).Format("2006-01-02")
		if item.Listens > math.MaxInt64-counts[key] {
			return nil, fmt.Errorf("activity count overflow")
		}
		counts[key] += item.Listens
	}
	grid := &ActivityGrid{Streak: streak, Timezone: loc.String()}
	now = now.In(loc)
	today := db.CivilDayKey(now)
	offset := (int(today.Weekday()) - weekStart + 7) % 7
	start := today.AddDate(0, 0, -offset-35*7)
	for i := 0; i < 252; i++ {
		civil := start.AddDate(0, 0, i)
		date := db.CivilDayStart(time.Date(civil.Year(), civil.Month(), civil.Day(), 12, 0, 0, 0, loc))
		count := counts[civil.Format("2006-01-02")]
		future := civil.After(today)
		if future {
			count = 0
		}
		opacity := 0.1 + float64(min(count, target))/float64(target)*0.9
		grid.Cells = append(grid.Cells, ActivityCell{date, count, fmt.Sprintf("%.3f", opacity), future, i / 7})
	}
	names := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	for i := 0; i < 7; i++ {
		label := ""
		if i%2 == 0 {
			label = names[(i+weekStart)%7]
		}
		grid.DayLabels = append(grid.DayLabels, label)
	}
	return grid, nil
}
