package admin_test

import (
	"context"
	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/db/sqlite"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

// accountFixture creates only an isolated database, never a configured installation.
func accountFixture(t *testing.T) (*sqlite.Sqlite, *models.User) {
	t.Helper()
	s, e := sqlite.NewInMemory()
	require.NoError(t, e)
	t.Cleanup(func() { s.Close(context.Background()) })
	u, e := s.SaveUser(context.Background(), db.SaveUserOpts{Username: "admin", Password: "original-password", Role: models.UserRoleAdmin})
	require.NoError(t, e)
	return s, u
}

// TestSessionLifetimeAndRevocation checks fixed lifetimes and account/profile independence.
func TestSessionLifetimeAndRevocation(t *testing.T) {
	s, u := accountFixture(t)
	ctx := context.Background()
	sessions := []*models.Session{}
	for _, remember := range []bool{false, true} {
		session, e := admin.Login(ctx, s, "admin", "original-password", remember)
		require.NoError(t, e)
		sessions = append(sessions, session)
		lifetime := 24 * time.Hour
		if remember {
			lifetime = 30 * 24 * time.Hour
		}
		require.WithinDuration(t, time.Now().Add(lifetime), session.ExpiresAt, 2*time.Second)
		require.Equal(t, remember, session.Persistent)
		for i := 0; i < 2; i++ {
			_, e = admin.Authenticate(ctx, s, session.ID)
			require.NoError(t, e)
		}
		var expiry int64
		require.NoError(t, s.QueryRow(`SELECT expires_at FROM sessions WHERE id=?`, session.ID.String()).Scan(&expiry))
		require.Equal(t, session.ExpiresAt.Unix(), expiry)
	}
	name := "renamed"
	changed, e := admin.ChangeAccount(ctx, s, u, &name, nil)
	require.NoError(t, e)
	require.False(t, changed)
	_, e = admin.Authenticate(ctx, s, sessions[0].ID)
	require.NoError(t, e)
	newpw := "replacement-password"
	changed, e = admin.ChangeAccount(ctx, s, u, nil, &newpw)
	require.NoError(t, e)
	require.True(t, changed)
	for _, session := range sessions {
		_, e = admin.Authenticate(ctx, s, session.ID)
		require.ErrorIs(t, e, admin.ErrUnauthorized)
	}
	_, e = admin.Login(ctx, s, "renamed", "original-password", false)
	require.ErrorIs(t, e, admin.ErrUnauthorized)
	session, e := admin.Login(ctx, s, "renamed", newpw, false)
	require.NoError(t, e)
	require.NoError(t, s.Exec(`UPDATE sessions SET expires_at=1 WHERE id=?`, session.ID.String()))
	_, e = admin.Authenticate(ctx, s, session.ID)
	require.ErrorIs(t, e, admin.ErrUnauthorized)
	_, e = admin.Authenticate(ctx, s, uuid.Nil)
	require.ErrorIs(t, e, admin.ErrUnauthorized)
}

type pausedLogin struct {
	*sqlite.Sqlite
	ready, resume chan struct{}
}

// SaveAuthenticatedSession pauses after bcrypt so a competing password write wins deterministically.
func (s pausedLogin) SaveAuthenticatedSession(ctx context.Context, id int32, hash []byte, expires time.Time, persistent bool) (*models.Session, error) {
	close(s.ready)
	<-s.resume
	return s.Sqlite.SaveAuthenticatedSession(ctx, id, hash, expires, persistent)
}

// TestPasswordChangeRejectsInFlightOldLogin prevents session resurrection after revocation.
func TestPasswordChangeRejectsInFlightOldLogin(t *testing.T) {
	s, u := accountFixture(t)
	store := pausedLogin{s, make(chan struct{}), make(chan struct{})}
	result := make(chan error, 1)
	go func() {
		_, err := admin.Login(context.Background(), store, "admin", "original-password", false)
		result <- err
	}()
	select {
	case <-store.ready:
	case <-time.After(3 * time.Second):
		t.Fatal("login did not reach session insert")
	}
	password := "replacement-password"
	_, err := admin.ChangeAccount(context.Background(), s, u, nil, &password)
	close(store.resume)
	require.NoError(t, err)
	require.ErrorIs(t, <-result, admin.ErrUnauthorized)
	var n int
	require.NoError(t, s.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&n))
	require.Zero(t, n)
}

// TestAccountPrivilegesAndInput rejects ingestion identities and explicit empty credentials.
func TestAccountPrivilegesAndInput(t *testing.T) {
	s, u := accountFixture(t)
	ctx := context.Background()
	empty := ""
	_, e := admin.ChangeAccount(ctx, s, u, &empty, nil)
	require.ErrorIs(t, e, db.ErrInvalidInput)
	_, e = admin.ChangeAccount(ctx, s, nil, nil, &empty)
	require.ErrorIs(t, e, admin.ErrUnauthorized)
	require.ErrorIs(t, admin.Authorize(&models.User{Role: models.UserRoleUser}), admin.ErrForbidden)
}

// TestPasswordRevocationRollback preserves both credentials and sessions if revocation fails.
func TestPasswordRevocationRollback(t *testing.T) {
	s, u := accountFixture(t)
	ctx := context.Background()
	session, err := admin.Login(ctx, s, "admin", "original-password", false)
	require.NoError(t, err)
	require.NoError(t, s.Exec(`CREATE TRIGGER fail_revoke BEFORE DELETE ON sessions BEGIN SELECT RAISE(ABORT,'injected'); END`))
	password := "replacement-password"
	_, err = admin.ChangeAccount(ctx, s, u, nil, &password)
	require.Error(t, err)
	_, err = admin.Authenticate(ctx, s, session.ID)
	require.NoError(t, err)
	_, err = admin.Login(ctx, s, "admin", "original-password", false)
	require.NoError(t, err)
	_, err = admin.Login(ctx, s, "admin", password, false)
	require.ErrorIs(t, err, admin.ErrUnauthorized)
}
