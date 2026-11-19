package admin

import (
	"context"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/google/uuid"
)

// AccountAccessStore revalidates the session and last-admin invariant in its write transaction.
type AccountAccessStore interface {
	SessionReader
	UpdateAccountAccess(context.Context, uuid.UUID, int32, models.UserRole, string) error
}

// ChangeAccountAccess is the service boundary for promotion, demotion and deactivation.
// Account activation and recovery tokens and their HTTP flows belong to ISSUE-162.
func ChangeAccountAccess(ctx context.Context, store AccountAccessStore, session uuid.UUID, target int32, role models.UserRole, status string) error {
	identity, err := Authenticate(ctx, store, session)
	if err != nil {
		return err
	}
	if err := AuthorizeAction(identity.User, ManageAccounts, nil); err != nil {
		return err
	}
	return store.UpdateAccountAccess(ctx, session, target, role, status)
}
