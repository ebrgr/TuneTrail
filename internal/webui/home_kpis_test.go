package webui

import (
	"context"
	"github.com/ebrgr/TuneTrail/internal/memkv"
	"strings"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
)

func TestHomeKPIsPeriodsAndEmptyBaseline(t *testing.T) {
	// Real SQLite reads prove profile scope, inclusive boundaries, zero baselines and all-time.
	s := pageStore(t)
	if err := s.Exec("DELETE FROM listens"); err != nil {
		t.Fatal(err)
	}
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2024, 3, 11, 12, 0, 0, 0, loc)
	for _, period := range []db.Period{db.PeriodDay, db.PeriodWeek, db.PeriodMonth, db.PeriodYear, db.PeriodAllTime} {
		t.Run(string(period), func(t *testing.T) {
			if err := s.Exec("DELETE FROM listens"); err != nil {
				t.Fatal(err)
			}
			from, to := db.TimeframeToTimeRangeAt(db.Timeframe{Period: period, Timezone: loc}, now)
			tf := db.Timeframe{From: from, To: to, Timezone: loc, Resolved: true}
			ctx := db.WithProfile(context.Background(), 1)
			save := func(profile int32, at time.Time) {
				t.Helper()
				if err := s.SaveListen(ctx, db.SaveListenOpts{ProfileID: profile, TrackID: 1, Time: at}); err != nil {
					t.Fatal(err)
				}
			}
			got, count, err := readHomeKPIs(ctx, s, tf)
			if err != nil || count != 0 {
				t.Fatalf("empty: %v %d", err, count)
			}
			for _, k := range got {
				if k.New || k.Change != "" || (period != db.PeriodAllTime && k.Direction != "same") {
					t.Fatalf("empty: %+v", k)
				}
			}
			if period != db.PeriodAllTime {
				save(1, from.Add(-time.Second))
				priorOnly, _, err := readHomeKPIs(ctx, s, tf)
				if err != nil {
					t.Fatal(err)
				}
				for _, k := range priorOnly {
					if k.Change != "100" || k.Direction != "less" {
						t.Fatalf("empty current: %+v", k)
					}
				}
				if err := s.Exec("DELETE FROM listens"); err != nil {
					t.Fatal(err)
				}
			}
			save(1, to)
			save(2, to)
			got, count, err = readHomeKPIs(ctx, s, tf)
			if err != nil || count != 1 || got[0].Value != 3 || got[2].Value != 1 || got[3].Value != 1 {
				t.Fatalf("values: %+v %d %v", got, count, err)
			}
			if period == db.PeriodAllTime {
				for _, k := range got {
					if k.Change != "" || k.New || k.Direction != "" {
						t.Fatalf("all time compares: %+v", k)
					}
				}
				return
			}
			for _, k := range got {
				if !k.New {
					t.Fatalf("zero baseline: %+v", k)
				}
			}
			save(1, from.Add(-time.Second))
			save(1, from)
			got, count, err = readHomeKPIs(ctx, s, tf)
			if err != nil || count != 2 || got[1].Change != "100" || got[1].Direction != "more" {
				t.Fatalf("adjacent bounds: %+v %d %v", got, count, err)
			}
			save(1, from.Add(-2*time.Second))
			equal, _, err := readHomeKPIs(ctx, s, tf)
			if err != nil || equal[1].Change != "" || equal[1].Direction != "same" {
				t.Fatalf("unchanged KPI: %+v %v", equal, err)
			}
			// A midnight transition counts consecutive local dates, rather than elapsed 24-hour days.
			if period == db.PeriodDay && got[3].Value != 2 {
				t.Fatalf("DST streak: %+v", got)
			}
		})
	}
}

func TestHomeNowPlayingAppearsBeforePeriodSummary(t *testing.T) {
	// The Home promotes playing metadata once; other profiles never inherit it.
	s := pageStore(t)
	memkv.Store.Set("now-playing:1", int32(1))
	t.Cleanup(func() { memkv.Store.Delete("now-playing:1") })
	h, err := NewHandler(s, Options{})
	if err != nil {
		t.Fatal(err)
	}
	body := pageRequest(h, "/alpha").Body.String()
	playing, summary := strings.Index(body, `class="card home-playing"`), strings.Index(body, `class="home-kpis"`)
	if playing < 0 || summary < playing || strings.Count(body, `class="now-dot"`) != 1 {
		t.Fatal("playing must appear once above the summary")
	}
	if strings.Contains(pageRequest(h, "/empty").Body.String(), `class="card home-playing"`) {
		t.Fatal("playing leaked to empty profile")
	}
}

func TestHomeHistoricalTotalsAndActivityTooltips(t *testing.T) {
	// Historical totals remain independent of the selected KPI period; the hidden graph keeps its table.
	s := pageStore(t)
	if err := s.SaveListen(context.Background(), db.SaveListenOpts{ProfileID: 1, TrackID: 1, Time: time.Date(2000, 1, 1, 12, 0, 0, 0, time.UTC)}); err != nil {
		t.Fatal(err)
	}
	h, err := NewHandler(s, Options{})
	if err != nil {
		t.Fatal(err)
	}
	body := pageRequest(h, "/alpha?period=day").Body.String()
	if !strings.Contains(body, `data-listen-count="4"`) || !strings.Contains(body, `class="stats home-totals"`) {
		t.Fatal("historical totals lost")
	}
	if !strings.Contains(body, `data-tip="`) || !strings.Contains(body, `class="activity-calendar" aria-hidden="true"`) || !strings.Contains(body, "Activity values") {
		t.Fatal("tooltip/table contract lost")
	}
	start, end := strings.Index(body, `class="activity-cells"`), strings.Index(body, `class="activity-footer"`)
	if start < 0 || end < start || strings.Contains(body[start:end], `title="`) {
		t.Fatal("native activity tooltip remains")
	}
}
