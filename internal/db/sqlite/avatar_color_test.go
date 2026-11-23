package sqlite

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ebrgr/TuneTrail/internal/testfixtures"
	"github.com/stretchr/testify/require"
)

// TestAvatarColorMigration upgrades a schema 21 file to 22: existing profiles keep their avatar, get no
// colour, and the pre-upgrade backup still opens at 21 (ISSUE-202).
func TestAvatarColorMigration(t *testing.T) {
	s, _, source := migrationDB(t, 21)
	testfixtures.Profile(t, s, 1, "one", "One", 0)
	execSQL(t, s, `UPDATE listening_profiles SET avatar_icon='music' WHERE id=1`)
	dir := t.TempDir()
	target := filepath.Join(dir, "tunetrail.db")
	require.NoError(t, copyDatabase(source, target))
	_, err := bootstrapAtVersion(dir, false, 22)
	require.NoError(t, err)

	backups, err := filepath.Glob(target + ".pre-ingest-*.bak")
	require.NoError(t, err)
	require.Len(t, backups, 1)
	backup, err := openExisting(backups[0])
	require.NoError(t, err)
	version, err := schemaVersion(backup)
	require.NoError(t, err)
	require.Equal(t, 21, version)
	require.NoError(t, backup.Close())

	conn, err := openExisting(target)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })
	require.NoError(t, checkDatabase(conn))
	version, err = schemaVersion(conn)
	require.NoError(t, err)
	require.Equal(t, 22, version)
	var icon string
	var color *string
	require.NoError(t, conn.QueryRow(`SELECT avatar_icon,avatar_color FROM listening_profiles WHERE id=1`).Scan(&icon, &color))
	require.Equal(t, "music", icon)
	require.Nil(t, color)
}

// TestAvatarColorStore round-trips the colour with the photo and icon, clears it with nil, and lets the
// column CHECK refuse any value that is not a lower-case #rrggbb.
func TestAvatarColorStore(t *testing.T) {
	s, err := NewInMemory()
	require.NoError(t, err)
	t.Cleanup(func() { s.Close(context.Background()) })
	ctx := context.Background()
	testfixtures.Profile(t, s, 1, "one", "One", 0)

	icon, color := "disc", "#1d4ed8"
	require.NoError(t, s.SetProfileAvatar(ctx, 1, nil, &icon, &color))
	p, err := s.GetListeningProfileByID(ctx, 1)
	require.NoError(t, err)
	require.NotNil(t, p.AvatarColor)
	require.Equal(t, "#1d4ed8", *p.AvatarColor)
	require.Equal(t, "disc", *p.AvatarIcon)
	list, err := s.ListListeningProfiles(ctx)
	require.NoError(t, err)
	require.Equal(t, "#1d4ed8", *list[0].AvatarColor)

	require.NoError(t, s.SetProfileAvatar(ctx, 1, nil, &icon, nil))
	p, err = s.GetListeningProfileByID(ctx, 1)
	require.NoError(t, err)
	require.Nil(t, p.AvatarColor)

	for _, bad := range []string{"red", "#FFFFFF", "#00000", "#0000000", "000000", "#12345g", ""} {
		_, err := s.db.Exec(`UPDATE listening_profiles SET avatar_color=? WHERE id=1`, bad)
		require.Error(t, err, "%q must violate the column CHECK", bad)
	}
}
