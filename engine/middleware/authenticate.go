package middleware

import (
	"context"
	"github.com/ebrgr/TuneTrail/internal/adminweb"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/models"
	"net/http"
)

type MiddlwareContextKey string

const UserContextKey MiddlwareContextKey = "user"

// validateSession uses the shared fixed-expiry service and never authenticates ingestion keys.
func validateSession(ctx context.Context, store db.UserStore, r *http.Request) (*models.User, error) {
	identity, err := adminweb.SessionIdentity(r.WithContext(ctx), store)
	if err != nil {
		return nil, err
	}
	return identity.User, nil
}

// GetUserFromContext reads the actor set by RequireAdmin, not a listening profile.
func GetUserFromContext(ctx context.Context) *models.User {
	user, _ := ctx.Value(UserContextKey).(*models.User)
	return user
}
