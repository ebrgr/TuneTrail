// The Python launcher supplies a private temporary directory for this replay.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/ebrgr/TuneTrail/internal/cfg"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/db/sqlite"
	"github.com/ebrgr/TuneTrail/internal/dedup"
	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/google/uuid"
)

type count struct {
	Different int `json:"different"`
	Same      int `json:"same"`
	Review    int `json:"review"`
	Skipped   int `json:"skipped"`
}

func (c *count) record(results []ingest.Result) {
	for _, r := range results {
		switch r.Decision {
		case "different":
			c.Different++
		case "same":
			c.Same++
		case "review":
			c.Review++
		}
	}
}

func main() {
	if len(os.Args) != 3 || (os.Args[1] != "copy" && os.Args[1] != "synthetic") {
		fmt.Fprintln(os.Stderr, "Use scripts/validate_ingest_shadow.py to launch the replay")
		os.Exit(2)
	}
	if err := run(os.Args[1], os.Args[2]); err != nil {
		fmt.Fprintln(os.Stderr, "Shadow replay failed:", err)
		os.Exit(1)
	}
}

// run uses the production SQLite receiver and prints aggregates only.
func run(mode, dir string) error {
	// The launcher marks its scratch directory after creating it.
	if _, err := os.Stat(filepath.Join(dir, ".shadow-replay-copy")); err != nil {
		return errors.New("launcher-created scratch directory required")
	}
	ctx := context.Background()
	if err := cfg.Load(func(key string) string {
		if key == cfg.CONFIG_DIR_ENV {
			return dir
		}
		return ""
	}, "test"); err != nil {
		return err
	}

	var store *sqlite.Sqlite
	var err error
	if mode == "copy" {
		store, err = sqlite.NewExisting()
	} else {
		store, err = sqlite.New()
	}
	if err != nil {
		return err
	}
	defer store.Close(ctx)
	var groups map[string]*count
	if mode == "copy" {
		groups, err = historical(ctx, store, dir)
	} else {
		groups, err = synthetic(ctx, store)
	}
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"rule_version": dedup.Shadow.Version, "groups": groups})
}

// historical pages saved listens before replaying each chunk as unkeyed observations.
func historical(ctx context.Context, store *sqlite.Sqlite, dir string) (map[string]*count, error) {
	uri := url.URL{Scheme: "file", Path: filepath.Join(dir, "tunetrail.db"), RawQuery: "mode=ro"}
	history, err := sql.Open("sqlite", uri.String())
	if err != nil {
		return nil, err
	}
	defer history.Close()
	report := make(map[string]*count)
	profiles, err := store.ListListeningProfiles(ctx)
	if err != nil {
		return nil, err
	}
	source := "shadow-replay-" + uuid.NewString()
	for _, profile := range profiles {
		c := &count{}
		report[fmt.Sprint(profile.ID)] = c
		var after int64
		for {
			results, next, err := historicalPage(ctx, store, history, c, profile.ID, after, source)
			if err != nil {
				return nil, err
			}
			after = next
			c.record(results)
			if len(results) == 0 && next == 0 {
				break
			}
		}
	}
	return report, nil
}

// historicalPage uses the catalog projection available for already stored history.
func historicalPage(ctx context.Context, store *sqlite.Sqlite, history *sql.DB, c *count, profile int32, after int64, source string) ([]ingest.Result, int64, error) {
	// Scalar subqueries avoid duplicate history rows even if aliases are inconsistent.
	const query = `SELECT l.rowid,l.listened_at,
		coalesce((SELECT title FROM tracks_with_title WHERE id=l.track_id ORDER BY title LIMIT 1),''),
		coalesce((SELECT r.title FROM tracks t JOIN releases_with_title r ON r.id=t.release_id WHERE t.id=l.track_id ORDER BY r.title LIMIT 1),''),
		coalesce((SELECT a.name FROM artist_tracks c JOIN artists_with_name a ON a.id=c.artist_id
			WHERE c.track_id=l.track_id ORDER BY c.is_primary DESC,a.id,a.name LIMIT 1),''),
		coalesce((SELECT musicbrainz_id FROM tracks WHERE id=l.track_id),''),
		coalesce((SELECT duration FROM tracks WHERE id=l.track_id),0)
		FROM listens l WHERE l.profile_id=? AND l.rowid>? ORDER BY l.rowid LIMIT 250`
	batch := ingest.Batch{ProfileID: profile, Source: source, Operation: "historical_replay", ReceivedAt: time.Now()}
	var next int64
	rows, err := history.QueryContext(ctx, query, profile, after)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	for rows.Next() {
		var item ingest.Item
		var second int64
		metadata := &item.Snapshot
		if err := rows.Scan(&next, &second, &metadata.Title, &metadata.Release, &metadata.Artist, &metadata.Recording, &metadata.DurationMS); err != nil {
			return nil, 0, err
		}
		if metadata.Title == "" || metadata.Artist == "" {
			c.Skipped++
			continue
		}
		metadata.DurationMS *= 1000
		item.Ordinal = len(batch.Items)
		item.ListenedAt = time.Unix(second, 0)
		batch.Items = append(batch.Items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if err := rows.Close(); err != nil {
		return nil, 0, err
	}
	results, err := store.ReceiveBatch(ctx, batch)
	return results, next, err
}

// synthetic contrasts nearby cross-source observations and a same-second replay with a control.
func synthetic(ctx context.Context, store *sqlite.Sqlite) (map[string]*count, error) {
	_, err := store.BootstrapAdministrator(ctx, db.BootstrapAdminOpts{
		Username: "replay", Password: "disposable-replay-password", APIKey: "disposable-replay-key",
	})
	if err != nil {
		return nil, err
	}
	profiles, err := store.ListListeningProfiles(ctx)
	if err != nil {
		return nil, err
	}
	if len(profiles) != 1 {
		return nil, errors.New("synthetic database must contain only the disposable profile")
	}
	report := make(map[string]*count)
	for i, scenario := range []struct {
		name, secondSource string
		delta              int
	}{
		{"cross_source_0s", "file", 0},
		{"cross_source_1s", "file", 1},
		{"cross_source_2s", "file", 2},
		{"same_source_same_second", "listenbrainz", 0},
		{"cross_source_3s_control", "file", 3},
	} {
		c := &count{}
		report[scenario.name] = c
		batch := ingest.Batch{ProfileID: profiles[0].ID, Source: "listenbrainz", Operation: "synthetic_replay", ReceivedAt: time.Now(),
			Items: []ingest.Item{{ListenedAt: time.Unix(1700000000+int64(i*60), 0),
				Snapshot: ingest.Snapshot{Artist: "Björk", Title: "Jóga Live", Release: "Concert Edition"}}}}
		results, err := store.ReceiveBatch(ctx, batch)
		if err != nil {
			return nil, err
		}
		c.record(results)
		batch.Source = scenario.secondSource
		batch.Items[0].ListenedAt = batch.Items[0].ListenedAt.Add(time.Duration(scenario.delta) * time.Second)
		batch.Items[0].Snapshot = ingest.Snapshot{Artist: " BJO\u0308RK ", Title: " JO\u0301GA  LIVE ", Release: "concert - edition"}
		results, err = store.ReceiveBatch(ctx, batch)
		if err != nil {
			return nil, err
		}
		c.record(results)
	}
	return report, nil
}
