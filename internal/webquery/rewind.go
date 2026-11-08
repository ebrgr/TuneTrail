package webquery

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
)

type RewindSelection struct {
	Year, Month          int
	ShowTime, InProgress bool
	Timeframe            db.Timeframe
	Query                url.Values
}

// Rewind resolves only the selection; the summary service freezes the interval.
func Rewind(r *http.Request, now time.Time) (RewindSelection, error) {
	return rewindAt(r, now, Timezone(r))
}

func rewindAt(r *http.Request, now time.Time, loc *time.Location) (RewindSelection, error) {
	q := r.URL.Query()
	invalid := func(key string) (RewindSelection, error) { return RewindSelection{}, fmt.Errorf("invalid %s", key) }
	for key, values := range q {
		if (key != "year" && key != "month" && key != "show_time" && key != "tz") || len(values) != 1 || values[0] == "" {
			return invalid(key)
		}
	}
	now = now.In(loc)
	year, month := 0, 0
	if q.Has("year") {
		v, err := strconv.Atoi(q.Get("year"))
		if err != nil || v < 1 || v > 9999 {
			return invalid("year")
		}
		year = v
	}
	if q.Has("month") {
		v, err := strconv.Atoi(q.Get("month"))
		if err != nil || v < 0 || v > 12 {
			return invalid("month")
		}
		month = v
	}
	showTime := false
	if q.Has("show_time") {
		switch q.Get("show_time") {
		case "0":
			q.Del("show_time")
		case "1":
			showTime = true
		default:
			return invalid("show_time")
		}
	}
	switch {
	case !q.Has("year") && !q.Has("month"):
		year = now.Year()
		if now.Month() == time.January {
			year--
		} else {
			month = int(now.Month()) - 1
		}
	case !q.Has("year"):
		year = now.Year()
		if month > int(now.Month()) {
			year--
		}
	}
	if year < 1 || year > 9999 {
		return invalid("year")
	}
	if year > now.Year() || (year == now.Year() && month > int(now.Month())) {
		return invalid("future period")
	}
	q.Set("year", strconv.Itoa(year))
	if month == 0 {
		q.Del("month")
	} else {
		q.Set("month", strconv.Itoa(month))
	}
	return RewindSelection{
		Year: year, Month: month, ShowTime: showTime,
		InProgress: year == now.Year() && (month == 0 || month == int(now.Month())),
		Timeframe:  db.Timeframe{Year: year, Month: month, Timezone: loc}, Query: q,
	}, nil
}
