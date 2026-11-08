package admin_test

import (
	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestLocalListenTimeHandlesDST(t *testing.T) {
	_, err := admin.LocalListenTime("2024-03-10T02:30", "America/New_York", "")
	require.ErrorIs(t, err, db.ErrInvalidInput)
	_, err = admin.LocalListenTime("2024-11-03T01:30", "America/New_York", "")
	require.ErrorIs(t, err, db.ErrConflict)
	a, err := admin.LocalListenTime("2024-11-03T01:30", "America/New_York", "-04:00")
	require.NoError(t, err)
	b, err := admin.LocalListenTime("2024-11-03T01:30", "America/New_York", "-05:00")
	require.NoError(t, err)
	require.Equal(t, time.Hour, b.Sub(a))
	_, err = admin.LocalListenTime("2024-11-03T01:30", "America/New_York", "-06:00")
	require.ErrorIs(t, err, db.ErrInvalidInput)
	for _, bad := range []string{"+-1:00", "+00:-1"} {
		_, err = admin.LocalListenTime("2024-11-03T01:30", "America/New_York", bad)
		require.ErrorIs(t, err, db.ErrInvalidInput)
	}
}
