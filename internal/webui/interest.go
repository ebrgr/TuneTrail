package webui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
)

type InterestRow struct {
	From, To    time.Time
	Count       int64
	Last, Empty bool
}

type InterestGraph struct {
	ID, Line, Area   string
	Rows             []InterestRow
	Total            int64
	Empty            bool
	SingleX, SingleY string
}

// buildInterest produces only numeric SVG coordinates. Catalog strings are never
// interpreted as markup, SVG paths or CSS. The table keeps the backend intervals.
func buildInterest(items []db.InterestBucket, loc *time.Location, id string) (*InterestGraph, error) {
	if len(items) > db.MaxInterestBuckets {
		return nil, fmt.Errorf("too many interest buckets")
	}
	if loc == nil {
		loc = time.UTC
	}
	g := &InterestGraph{ID: id, Empty: true}
	var peak int64
	for i, item := range items {
		if item.ListenCount < 0 || item.ListenCount > math.MaxInt64-g.Total || item.BucketEnd.Before(item.BucketStart) {
			return nil, fmt.Errorf("invalid interest bucket")
		}
		last := i == len(items)-1
		if i > 0 && !item.BucketStart.Equal(items[i-1].BucketEnd) {
			return nil, fmt.Errorf("discontinuous interest buckets")
		}
		if !last && item.BucketStart.Equal(item.BucketEnd) && item.ListenCount != 0 {
			return nil, fmt.Errorf("nonempty zero-width interest bucket")
		}
		g.Rows = append(g.Rows, InterestRow{item.BucketStart.In(loc), item.BucketEnd.In(loc), item.ListenCount, last, !last && item.BucketStart.Equal(item.BucketEnd)})
		g.Total += item.ListenCount
		peak = max(peak, item.ListenCount)
	}
	if g.Total == 0 {
		return g, nil
	}
	g.Empty = false
	const left, right, top, bottom = 8.0, 442.0, 12.0, 108.0
	xs, ys := make([]float64, len(items)), make([]float64, len(items))
	for i, item := range items {
		xs[i] = (left + right) / 2
		if len(items) > 1 {
			xs[i] = left + float64(i)*(right-left)/float64(len(items)-1)
		}
		ys[i] = bottom - float64(item.ListenCount)/float64(peak)*(bottom-top)
	}
	g.Line = boundedCurve(xs, ys)
	g.Area = g.Line + fmt.Sprintf(" L%.3f,%.3f L%.3f,%.3f Z", xs[len(xs)-1], bottom, xs[0], bottom)
	if len(items) == 1 {
		g.SingleX = fmt.Sprintf("%.3f", xs[0])
		g.SingleY = fmt.Sprintf("%.3f", ys[0])
	}
	return g, nil
}

// boundedCurve uses horizontal tangents at each point. Each cubic Bézier's
// control points stay between its endpoint values, preventing overshoot.
func boundedCurve(xs, ys []float64) string {
	if len(xs) == 0 {
		return ""
	}
	var path strings.Builder
	fmt.Fprintf(&path, "M%.3f,%.3f", xs[0], ys[0])
	for i := 0; i < len(xs)-1; i++ {
		step := xs[i+1] - xs[i]
		fmt.Fprintf(&path, " C%.3f,%.3f %.3f,%.3f %.3f,%.3f", xs[i]+step/3, ys[i], xs[i+1]-step/3, ys[i+1], xs[i+1], ys[i+1])
	}
	return path.String()
}
