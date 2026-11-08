package listening

import (
	"github.com/ebrgr/TuneTrail/internal/db"
	"time"
)

// FillActivity supplies empty buckets using the same calendar rules as the API.
func FillActivity(
	items []db.ListenActivityItem,
	opts db.ListenActivityOpts,
) []db.ListenActivityItem {
	from, to := db.ListenActivityOptsToTimes(opts)

	buckets := make(map[string]int64)

	for _, item := range items {
		bucketStart := normalizeToStep(item.Start, opts.Step)
		key := bucketStart.Format("2006-01-02")
		buckets[key] += item.Listens
	}

	var result []db.ListenActivityItem

	for t := normalizeToStep(from, opts.Step); t.Before(to); t = addStep(t, opts.Step) {
		key := t.Format("2006-01-02")

		result = append(result, db.ListenActivityItem{
			Start:   t,
			Listens: buckets[key],
		})
	}

	return result
}

// normalizeToStep preserves local civil boundaries, including midnight transitions.
func normalizeToStep(t time.Time, step db.StepInterval) time.Time {
	switch step {
	case db.StepDay:
		return db.CivilDayStart(t)

	case db.StepWeek:
		weekday := int(t.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		start := time.Date(t.Year(), t.Month(), t.Day()-weekday+1, 12, 0, 0, 0, t.Location())
		return db.CivilDayStart(start)

	case db.StepMonth:
		return db.CivilDayStart(time.Date(t.Year(), t.Month(), 1, 12, 0, 0, 0, t.Location()))

	case db.StepYear:
		return db.CivilDayStart(time.Date(t.Year(), 1, 1, 12, 0, 0, 0, t.Location()))

	default:
		return t
	}
}

// addStep advances calendar buckets without carrying a shifted midnight forward.
func addStep(t time.Time, step db.StepInterval) time.Time {
	switch step {
	case db.StepDay:
		return db.CivilDayStart(time.Date(t.Year(), t.Month(), t.Day()+1, 12, 0, 0, 0, t.Location()))
	case db.StepWeek:
		return db.CivilDayStart(time.Date(t.Year(), t.Month(), t.Day()+7, 12, 0, 0, 0, t.Location()))
	case db.StepMonth:
		return db.CivilDayStart(time.Date(t.Year(), t.Month()+1, 1, 12, 0, 0, 0, t.Location()))
	case db.StepYear:
		return db.CivilDayStart(time.Date(t.Year()+1, 1, 1, 12, 0, 0, 0, t.Location()))
	default:
		return t.AddDate(0, 0, 1)
	}
}
