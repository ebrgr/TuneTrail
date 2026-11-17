package webui

import (
	"context"
	"fmt"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/listening"
)

type HomeKPI struct {
	Key   string
	Value int64
	// Change is absent for all-time; New avoids dividing by an empty baseline.
	Change string
	New    bool
}

func homeValues(ctx context.Context, s listening.StatsStore, tf db.Timeframe) ([4]int64, error) {
	// Read only the four dashboard metrics; keep the API's historical averages unchanged.
	var v [4]int64
	seconds, err := s.CountTimeListened(ctx, tf)
	if err != nil {
		return v, fmt.Errorf("home minutes: %w", err)
	}
	v[0] = seconds / 60
	v[1], err = s.CountListens(ctx, tf)
	if err != nil {
		return v, fmt.Errorf("home listens: %w", err)
	}
	v[2], err = s.CountArtists(ctx, tf)
	if err != nil {
		return v, fmt.Errorf("home artists: %w", err)
	}
	streak, err := s.GetLongestListenStreak(ctx, db.ListenActivityOpts{Timezone: tf.Timezone, Timeframe: &tf})
	if err != nil {
		return v, fmt.Errorf("home streak: %w", err)
	}
	v[3] = int64(streak)
	return v, nil
}

func readHomeKPIs(ctx context.Context, s listening.StatsStore, tf db.Timeframe) ([]HomeKPI, int64, error) {
	// Adjacent inclusive ranges have equal elapsed seconds, including across DST.
	current, err := homeValues(ctx, s, tf)
	if err != nil {
		return nil, 0, err
	}
	var previous [4]int64
	compare := !tf.From.IsZero()
	if compare {
		prior := tf
		prior.To = tf.From.Add(-time.Second)
		prior.From = prior.To.Add(-tf.To.Sub(tf.From))
		previous, err = homeValues(ctx, s, prior)
		if err != nil {
			return nil, 0, err
		}
	}
	result := make([]HomeKPI, 4)
	for i, key := range []string{"minutes", "plays", "artists", "streak"} {
		k := HomeKPI{Key: key, Value: current[i]}
		if compare {
			if previous[i] == 0 && current[i] > 0 {
				k.New = true
			} else if previous[i] == 0 {
				k.Change = "0%"
			} else {
				k.Change = fmt.Sprintf("%+.0f%%", 100*(float64(current[i])-float64(previous[i]))/float64(previous[i]))
			}
		}
		result[i] = k
	}
	return result, current[1], nil
}
