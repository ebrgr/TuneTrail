package summary_test

import (
	"context"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/cfg"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/db/sqlite"
	"github.com/ebrgr/TuneTrail/internal/summary"
)

func TestGenerateSummaryEmptySQLite(t *testing.T) {
	if err := cfg.Load(func(string) string { return "" }, "test"); err != nil {
		t.Fatal(err)
	}
	store, err := sqlite.NewInMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close(context.Background())
	if err := store.Exec(`INSERT INTO listening_profiles(id,slug,display_name,created_at) VALUES(1,'empty','Empty',1)`); err != nil {
		t.Fatal(err)
	}
	result, err := summary.GenerateSummary(context.Background(), store, 1, db.Timeframe{Year: 2024, Month: 2, Timezone: time.UTC}, "February")
	if err != nil {
		t.Fatal(err)
	}
	if result.Title != "February" || result.Plays != 0 || result.AvgPlaysPerDay != 0 || len(result.TopTracks) != 0 {
		t.Fatalf("unexpected summary: %+v", result)
	}
	if result.IntervalFrom != time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC) || result.IntervalTo != time.Date(2024, 2, 29, 23, 59, 59, 0, time.UTC) {
		t.Fatalf("unexpected interval: %v – %v", result.IntervalFrom, result.IntervalTo)
	}
}
