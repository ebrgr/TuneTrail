package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/stretchr/testify/require"
)

// TestProfileScopeAndCatalogEscape documents scoped reads and the intentional internal global context.
func TestProfileScopeAndCatalogEscape(t *testing.T) {
	s, _, _ := migrationDB(t, 3)
	seedMediaRead(t, s)
	execSQL(t, s, `INSERT INTO listening_profiles(id,slug,display_name,created_at) VALUES(1,'alpha','Alpha',1),(2,'beta','Beta',1),(3,'empty','Empty',1)`)
	for _, profile := range []int32{1, 1, 2} {
		stamp := int64(1700000000) + int64(profile)
		if profile == 1 && scalar(t, s, `SELECT COUNT(*) FROM listens WHERE profile_id=1`) > 0 {
			stamp++
		}
		require.NoError(t, s.SaveListen(context.Background(), db.SaveListenOpts{ProfileID: profile, TrackID: 1, Time: time.Unix(stamp, 0)}))
	}
	for _, tc := range []struct {
		id   int32
		want int64
	}{{1, 2}, {2, 1}, {3, 0}} {
		ctx := db.WithProfile(context.Background(), tc.id)
		n, err := s.CountListens(ctx, db.Timeframe{Period: db.PeriodAllTime, Timezone: time.UTC})
		require.NoError(t, err)
		require.Equal(t, tc.want, n)
		items, err := s.GetListensPaginated(ctx, db.GetItemsOpts{Page: 1, Limit: 100, Timeframe: db.Timeframe{Period: db.PeriodAllTime, Timezone: time.UTC}})
		require.NoError(t, err)
		require.Equal(t, tc.want, items.TotalCount)
	}
	// Unscoped context is an intentional internal catalog capability, never a public actor.
	ctx := db.WithoutProfile(db.WithProfile(context.Background(), 1))
	n, err := s.CountListens(ctx, db.Timeframe{Period: db.PeriodAllTime, Timezone: time.UTC})
	require.NoError(t, err)
	require.EqualValues(t, 3, n)
	n, err = s.CountListens(context.Background(), db.Timeframe{Period: db.PeriodAllTime, Timezone: time.UTC})
	require.NoError(t, err)
	require.EqualValues(t, 3, n)
}
