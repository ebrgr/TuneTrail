package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCatalogPlanStaleAndScopedMerge(t *testing.T) {
	for _, kind := range []string{"artist", "album", "track"} {
		t.Run(kind, func(t *testing.T) {
			s := mergeFixture(t)
			ctx := context.Background()
			request := db.CatalogPlan{Action: "merge", Kind: kind, From: 1, To: 2}
			plan, err := s.PreviewCatalogMutation(ctx, request)
			require.NoError(t, err)
			require.NotEmpty(t, plan.Fingerprint)
			require.Len(t, plan.Profiles, 2)
			require.Equal(t, "One", plan.SourceName)
			require.Equal(t, "Two", plan.DestinationName)
			if kind == "track" {
				require.EqualValues(t, 2, plan.DuplicateListens)
			}
			table := map[string]string{"artist": "artist_aliases", "album": "release_aliases", "track": "track_aliases"}[kind]
			require.NoError(t, s.Exec(`UPDATE `+table+` SET source='changed' WHERE alias='One'`))
			require.ErrorIs(t, s.ApplyCatalogMutation(ctx, plan), db.ErrConflict)
			plan, err = s.PreviewCatalogMutation(ctx, request)
			require.NoError(t, err)
			require.NoError(t, s.ApplyCatalogMutation(ctx, plan))
			require.ErrorIs(t, s.ApplyCatalogMutation(ctx, plan), db.ErrConflict)
		})
	}
}
func TestPlanAccountsForArtistAlbumTrigger(t *testing.T) {
	s := mergeFixture(t)
	ctx := context.Background()
	require.NoError(t, s.Exec(`INSERT INTO artist_tracks(artist_id,track_id,is_primary) VALUES(2,1,0)`))
	plan, err := s.PreviewCatalogMutation(ctx, db.CatalogPlan{Action: "delete", Kind: "artist", From: 1})
	require.NoError(t, err)
	require.EqualValues(t, 1, plan.RemovedTracks)
	require.EqualValues(t, 2, plan.RemovedListens)
	require.Len(t, plan.Profiles, 2)
	require.NoError(t, s.ApplyCatalogMutation(ctx, plan))
	require.Zero(t, scalar(t, s, `SELECT COUNT(*) FROM tracks WHERE id=1`))
	require.Equal(t, 1, scalar(t, s, `SELECT COUNT(*) FROM tracks WHERE id=2`))
}
func TestPlanRollbackAndIntegrity(t *testing.T) {
	s := mergeFixture(t)
	ctx := context.Background()
	plan, err := s.PreviewCatalogMutation(ctx, db.CatalogPlan{Action: "merge", Kind: "track", From: 1, To: 2})
	require.NoError(t, err)
	before := mergeSnapshot(t, s)
	require.NoError(t, s.Exec(`CREATE TRIGGER fail_confirmed_merge BEFORE UPDATE ON listens BEGIN SELECT RAISE(ABORT,'injected'); END`))
	require.Error(t, s.ApplyCatalogMutation(ctx, plan))
	require.Equal(t, before, mergeSnapshot(t, s))
	for _, p := range []db.CatalogPlan{{Action: "merge", Kind: "track", From: 1, To: 1}, {Action: "merge", Kind: "album", From: 1, To: 999}, {Action: "delete", Kind: "artist", From: -1}} {
		_, err = s.PreviewCatalogMutation(ctx, p)
		require.Error(t, err)
	}
}

func TestWALSnapshotRaceMapsToConflict(t *testing.T) {
	s, _, filename := migrationDB(t, 3)
	ctx := context.Background()
	for _, q := range []string{`INSERT INTO listening_profiles(id,slug,display_name,created_at) VALUES(1,'wal-test','WAL',1)`, `INSERT INTO artists(id) VALUES(1)`, `INSERT INTO artist_aliases VALUES(1,'Artist','User',1)`, `INSERT INTO releases(id,various_artists) VALUES(1,0)`, `INSERT INTO release_aliases VALUES(1,'Album','User',1)`, `INSERT INTO artist_releases VALUES(1,1,1)`, `INSERT INTO tracks(id,release_id,duration) VALUES(1,1,1),(2,1,1)`, `INSERT INTO track_aliases VALUES(1,'Source','User',1),(2,'Target','User',1)`, `INSERT INTO artist_tracks VALUES(1,1,1),(1,2,1)`, `INSERT INTO listens(profile_id,track_id,listened_at) VALUES(1,1,100)`} {
		require.NoError(t, s.Exec(q))
	}
	tx, err := s.db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()
	_, err = planCatalogIn(ctx, tx, db.CatalogPlan{Action: "merge", Kind: "track", From: 1, To: 2})
	require.NoError(t, err)
	other, err := sql.Open("sqlite", filename+"?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)")
	require.NoError(t, err)
	defer other.Close()
	_, err = other.ExecContext(ctx, `INSERT INTO listens(profile_id,track_id,listened_at) VALUES(1,1,101)`)
	require.NoError(t, err)
	err = mergeTracksIn(ctx, tx, 1, 2)
	require.Error(t, err)
	require.ErrorIs(t, staleWriteConflict(err), db.ErrConflict)
	require.NoError(t, tx.Rollback())
	var n int
	require.NoError(t, other.QueryRowContext(ctx, `SELECT COUNT(*) FROM listens WHERE track_id=1`).Scan(&n))
	require.Equal(t, 2, n)
}

func TestArtistAlbumOnlyRelationIncludesAffectedListeners(t *testing.T) {
	s := mergeFixture(t)
	ctx := context.Background()
	require.NoError(t, s.Exec(`INSERT INTO artist_releases(artist_id,release_id,is_primary) VALUES(2,1,0)`))
	require.NoError(t, s.Exec(`DELETE FROM artist_tracks WHERE artist_id=1 AND track_id=1`))
	require.NoError(t, s.Exec(`INSERT INTO artist_tracks(artist_id,track_id,is_primary) VALUES(2,1,1)`))
	plan, err := s.PreviewCatalogMutation(ctx, db.CatalogPlan{Action: "delete", Kind: "artist", From: 1})
	require.NoError(t, err)
	require.EqualValues(t, 1, plan.SourceTracks)
	require.EqualValues(t, 2, plan.SourceListens)
	require.Len(t, plan.Profiles, 2)
	require.Zero(t, plan.RemovedTracks)
	require.Zero(t, plan.RemovedListens)
	merge, err := s.PreviewCatalogMutation(ctx, db.CatalogPlan{Action: "merge", Kind: "artist", From: 2, To: 1})
	require.NoError(t, err)
	require.Len(t, merge.DestinationProfiles, 2)
}

func TestCatalogMergeBothDirectionsPerProfile(t *testing.T) {
	for _, ids := range [][2]int32{{1, 2}, {2, 1}} {
		t.Run(fmt.Sprintf("%d-to-%d", ids[0], ids[1]), func(t *testing.T) {
			s := mergeFixture(t)
			ctx := context.Background()
			plan, err := s.PreviewCatalogMutation(ctx, db.CatalogPlan{Action: "merge", Kind: "track", From: ids[0], To: ids[1]})
			require.NoError(t, err)
			require.EqualValues(t, 2, plan.DuplicateListens)
			require.NoError(t, s.ApplyCatalogMutation(ctx, plan))
			require.Equal(t, 1, scalar(t, s, `SELECT COUNT(*) FROM tracks WHERE id=?`, ids[1]))
			for _, p := range []int{1, 2} {
				require.Equal(t, 1, scalar(t, s, `SELECT COUNT(*) FROM listens WHERE profile_id=? AND track_id=?`, p, ids[1]))
			}
		})
	}
}
