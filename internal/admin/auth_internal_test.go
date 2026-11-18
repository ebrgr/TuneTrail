package admin

import (
	"context"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

type rejectedLoginStore struct{ user *models.User }

// GetUserByUsername supplies missing and pending identities without querying a database.
func (s rejectedLoginStore) GetUserByUsername(context.Context, string) (*models.User, error) {
	return s.user, nil
}

// SaveAuthenticatedSession must never run for either rejected identity.
func (s rejectedLoginStore) SaveAuthenticatedSession(context.Context, int32, []byte, time.Time, bool) (*models.Session, error) {
	panic("unexpected session insert")
}

// TestRejectedLoginRunsBcrypt159 observes comparison directly, independent of elapsed time.
func TestRejectedLoginRunsBcrypt159(t *testing.T) {
	cost, err := bcrypt.Cost(invalidLoginHash)
	require.NoError(t, err)
	require.Equal(t, bcrypt.DefaultCost, cost)
	for _, tc := range []struct {
		name string
		user *models.User
	}{
		{"missing", nil},
		{"pending", &models.User{Status: "pending"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			session, err := loginWithCompare(context.Background(), rejectedLoginStore{tc.user}, "account", "test-password", false, func(hash, password []byte) error {
				calls++
				require.Equal(t, invalidLoginHash, hash)
				require.Equal(t, []byte("test-password"), password)
				return bcrypt.CompareHashAndPassword(hash, password)
			})
			require.ErrorIs(t, err, ErrUnauthorized)
			require.Nil(t, session)
			require.Equal(t, 1, calls)
			// Even a successful comparison against the dummy hash cannot authenticate.
			session, err = loginWithCompare(context.Background(), rejectedLoginStore{tc.user}, "account", "test-password", false, func([]byte, []byte) error { return nil })
			require.ErrorIs(t, err, ErrUnauthorized)
			require.Nil(t, session)
		})
	}
}
