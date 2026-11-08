package engine

import (
	"errors"
	"fmt"
	"io"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/db/sqlite"
	"github.com/ebrgr/TuneTrail/internal/utils"
)

func ResetPassword(getenv func(string) string, w io.Writer, version string) error {
	l, ctx, err := initLogger(getenv, version, w)
	if err != nil {
		return fmt.Errorf("ResetPassword: %w", err)
	}

	// Reset only a previously existing, safely selected database.
	store, err := sqlite.NewExisting()
	if err != nil {
		return fmt.Errorf("ResetPassword: %w", err)
	}
	defer store.Close(ctx)

	user, err := store.GetAdminUser(ctx)
	if err != nil {
		return fmt.Errorf("ResetPassword: failed to get admin user: %w", err)
	}
	if user == nil {
		return errors.New("ResetPassword: no admin user found")
	}

	newPassword, err := utils.GenerateRandomString(16)
	if err != nil {
		return fmt.Errorf("ResetPassword: failed to generate password: %w", err)
	}

	if err := store.UpdateUser(ctx, db.UpdateUserOpts{
		ID:       user.ID,
		Password: newPassword,
	}); err != nil {
		return fmt.Errorf("ResetPassword: failed to update password: %w", err)
	}

	l.Info().Msgf("Password for '%s' has been reset. New password: %s", user.Username, newPassword)
	return nil
}
