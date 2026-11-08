package webquery

import (
	"fmt"
	"github.com/ebrgr/TuneTrail/internal/db"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// HTML validates presentation requests without changing the legacy JSON parser.
func HTML(r *http.Request, home bool) (db.GetItemsOpts, url.Values, error) {
	q := r.URL.Query()
	o := db.GetItemsOpts{Page: 1, Limit: 100, Timeframe: TimeframeFromRequest(r)}
	bad := func(key string) (db.GetItemsOpts, url.Values, error) { return o, q, fmt.Errorf("invalid %s", key) }
	for _, key := range []string{"page", "limit", "year", "month", "week", "artist_id", "album_id", "track_id"} {
		raw := q.Get(key)
		if raw == "" {
			q.Del(key)
			continue
		}
		n, e := strconv.ParseInt(raw, 10, 32)
		if e != nil || n < 0 {
			return bad(key)
		}
		switch key {
		case "page":
			if n == 0 {
				n = 1
			}
			o.Page = int(n)
		case "limit":
			if n < 1 || n > 500 {
				return bad(key)
			}
			o.Limit = int(n)
		case "year":
			if n < 1 || n > 9999 {
				return bad(key)
			}
			o.Timeframe.Year = int(n)
		case "month":
			if n < 1 || n > 12 {
				return bad(key)
			}
			o.Timeframe.Month = int(n)
		case "week":
			if n < 1 || n > 53 {
				return bad(key)
			}
			o.Timeframe.Week = int(n)
		case "artist_id":
			if n < 1 {
				return bad(key)
			}
			o.ArtistID = int(n)
		case "album_id":
			if n < 1 {
				return bad(key)
			}
			o.AlbumID = int(n)
		case "track_id":
			if n < 1 {
				return bad(key)
			}
			o.TrackID = int(n)
		}
		q.Set(key, strconv.FormatInt(n, 10))
	}
	if o.Page > (math.MaxInt32 / o.Limit) {
		return bad("page")
	}
	period := q.Get("period")
	if period == "" {
		period = "all_time"
		if home {
			period = "week"
		}
	}
	switch period {
	case "day", "week", "month", "year", "all_time":
	default:
		return bad("period")
	}
	o.Timeframe.Period = db.Period(period)
	if o.Timeframe.Month > 0 && o.Timeframe.Week > 0 {
		return bad("month/week")
	}
	for _, key := range []string{"from", "to"} {
		raw := q.Get(key)
		if raw == "" {
			q.Del(key)
			continue
		}
		n, e := strconv.ParseInt(raw, 10, 64)
		if e != nil || n < 1 || n > 253402300799 {
			return bad(key)
		}
		if key == "from" {
			o.Timeframe.FromUnix = n
		} else {
			o.Timeframe.ToUnix = n
		}
	}
	if o.Timeframe.ToUnix != 0 && o.Timeframe.FromUnix == 0 {
		return bad("from")
	}
	calendar := o.Timeframe.Year > 0 || o.Timeframe.Month > 0 || o.Timeframe.Week > 0
	if o.Timeframe.FromUnix != 0 {
		if calendar || o.Timeframe.ToUnix != 0 && o.Timeframe.ToUnix < o.Timeframe.FromUnix {
			return bad("date range")
		}
		o.Timeframe.Period = ""
		q.Del("period")
	}
	if calendar {
		o.Timeframe.Period = ""
		q.Del("period")
		if o.Timeframe.Week > 0 {
			from, _ := db.TimeframeToTimeRange(o.Timeframe)
			year, week := from.ISOWeek()
			expected := o.Timeframe.Year
			if expected == 0 {
				expected = year
			}
			if year != expected || week != o.Timeframe.Week {
				return bad("ISO week")
			}
		}
	}
	if home && (calendar || o.Timeframe.FromUnix > 0 || o.ArtistID > 0 || o.AlbumID > 0 || o.TrackID > 0) {
		return bad("home filters")
	}
	o.Timeframe = ResolveTimeframe(o.Timeframe)
	return o, q, nil
}
func RangeLabel(tf db.Timeframe) string {
	from, to := db.TimeframeToTimeRange(tf)
	if from.IsZero() {
		return "All Time"
	}
	loc := tf.Timezone
	if loc == nil {
		loc = time.UTC
	}
	return from.In(loc).Format("02 Jan 2006 15:04") + " – " + to.In(loc).Format("02 Jan 2006 15:04")
}

// ResolveTimeframe fixes rolling bounds once for a read and its displayed label.
// All-time keeps its zero start, preserving the store's existing sentinel.
func ResolveTimeframe(tf db.Timeframe) db.Timeframe {
	from, to := db.TimeframeToTimeRange(tf)
	if !from.IsZero() {
		tf.From = from
		tf.To = to
	}
	return tf
}
