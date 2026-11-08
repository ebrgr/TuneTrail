package webui

import (
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
)

// TestInterestCurveStaysWithinEachInterval detects overshoot below zero and above peaks.
func TestInterestCurveStaysWithinEachInterval(t *testing.T) {
	for _, counts := range [][]int64{{0, 0, 4, 0, 0}, {0, 10, 0, 10, 0}, {1, 2, 3, 4}, {7, 7, 7}, {9, 2, 0}} {
		t.Run(fmt.Sprint(counts), func(t *testing.T) {
			items := make([]db.InterestBucket, len(counts))
			var peak int64
			for i, count := range counts {
				items[i] = db.InterestBucket{BucketStart: time.Unix(int64(i), 0), BucketEnd: time.Unix(int64(i+1), 0), ListenCount: count}
				peak = max(peak, count)
			}
			graph, err := buildInterest(items, time.UTC, "interest")
			if err != nil {
				t.Fatal(err)
			}
			parts := strings.Split(graph.Line, " C")
			if len(parts) != len(counts) {
				t.Fatalf("unexpected segments: %q", graph.Line)
			}
			for i, segment := range parts[1:] {
				var x1, y1, x2, y2, x3, y3 float64
				if n, err := fmt.Sscanf(segment, "%f,%f %f,%f %f,%f", &x1, &y1, &x2, &y2, &x3, &y3); err != nil || n != 6 {
					t.Fatalf("invalid segment %q: %v", segment, err)
				}
				start := 108 - float64(counts[i])/float64(peak)*96
				end := 108 - float64(counts[i+1])/float64(peak)*96
				lo, hi := math.Min(start, end)-0.001, math.Max(start, end)+0.001
				for step := 0; step <= 100; step++ {
					u := float64(step) / 100
					v := 1 - u
					y := v*v*v*start + 3*v*v*u*y1 + 3*v*u*u*y2 + u*u*u*y3
					if y < lo || y > hi {
						t.Fatalf("segment %d exceeds [%f,%f]: %f", i, lo, hi, y)
					}
				}
			}
		})
	}
}
