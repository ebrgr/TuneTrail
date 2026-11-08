package summary

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
)

type firstListenReader interface {
	GetFirstListenUnix(context.Context) (int64, error)
}

// resolveInterval preserves the legacy SQL boundaries while giving every query
// in one summary identical endpoints. Calendar averages count civil dates;
// rolling and explicit ranges retain complete elapsed 24-hour periods.
func resolveInterval(ctx context.Context, store firstListenReader, tf db.Timeframe, now time.Time) (db.Timeframe, int64, error) {
	// A frozen interval alone does not preserve whether its bounds came from a
	// calendar or an explicit range. Resolve only once, inside this service.
	if tf.Resolved {
		return tf, 0, fmt.Errorf("summary requires an unresolved timeframe")
	}
	calendar := tf.From.IsZero() && tf.FromUnix == 0 && (tf.Year != 0 || tf.Month != 0 || tf.Week != 0)
	allTime := tf.From.IsZero() && tf.FromUnix == 0 && tf.Year == 0 && tf.Month == 0 && tf.Week == 0 && tf.Period == db.PeriodAllTime
	from, to := db.TimeframeToTimeRangeAt(tf, now)
	if to.Before(from) {
		return tf, 0, fmt.Errorf("summary interval ends before it starts")
	}
	tf.From, tf.To, tf.Resolved = from, to, true
	if from.IsZero() && to.IsZero() {
		return tf, 1, nil
	}
	loc := tf.Timezone
	if loc == nil {
		loc = time.UTC
	}
	if allTime {
		first, err := store.GetFirstListenUnix(ctx)
		if errors.Is(err, sql.ErrNoRows) {
			return tf, 1, nil
		}
		if err != nil {
			return tf, 0, fmt.Errorf("first listen: %w", err)
		}
		firstTime := time.Unix(first, 0)
		if firstTime.After(to) {
			return tf, 1, nil
		}
		if firstTime.Before(from) {
			firstTime = from
		}
		days, err := civilDays(firstTime, to, loc)
		return tf, days, err
	}
	if calendar {
		// A local midnight may not exist (for example Asuncion, October 2017).
		// Preserve the legacy SQL bounds but count the selected calendar period,
		// rather than counting dates after Go normalizes that missing midnight.
		return tf, calendarDays(tf, now), nil
	}
	// Unsigned subtraction is exact for ordered int64 Unix seconds, even when
	// the duration exceeds time.Duration's approximately 290-year range.
	seconds := uint64(to.Unix()) - uint64(from.Unix())
	days := int64(seconds / 86400)
	if days < 1 {
		days = 1
	}
	return tf, days, nil
}

func calendarDays(tf db.Timeframe, now time.Time) int64 {
	// Infer an omitted calendar year in the same zone used to resolve the range.
	if tf.Timezone != nil {
		now = now.In(tf.Timezone)
	} else {
		now = now.UTC()
	}
	if tf.Month != 0 {
		year := tf.Year
		if year == 0 {
			year = now.Year()
			if int(now.Month()) < tf.Month {
				year--
			}
		}
		month := time.Date(year, time.Month(tf.Month), 1, 12, 0, 0, 0, time.UTC)
		return int64(time.Date(month.Year(), month.Month()+1, 0, 12, 0, 0, 0, time.UTC).Day())
	}
	if tf.Week != 0 {
		return 7
	}
	year := tf.Year
	if year%400 == 0 || year%4 == 0 && year%100 != 0 {
		return 366
	}
	return 365
}

func civilDays(from, to time.Time, loc *time.Location) (int64, error) {
	a, b := from.In(loc), to.In(loc)
	if a.Year() < 0 || a.Year() > 10000 || b.Year() < 0 || b.Year() > 10000 {
		return 0, fmt.Errorf("summary calendar date outside supported years")
	}
	aDay := time.Date(a.Year(), a.Month(), a.Day(), 0, 0, 0, 0, time.UTC).Unix()
	bDay := time.Date(b.Year(), b.Month(), b.Day(), 0, 0, 0, 0, time.UTC).Unix()
	days := (bDay-aDay)/86400 + 1
	if days < 1 {
		return 0, fmt.Errorf("summary interval contains no calendar days")
	}
	return days, nil
}
