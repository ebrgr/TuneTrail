package db

import (
	"context"
	"errors"

	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/google/uuid"
)

// ErrAccountToken covers all rejected redemption attempts.
var ErrAccountToken = errors.New("invalid account token")

// ErrOwnAccountToken identifies a forbidden self-issuance.
var ErrOwnAccountToken = errors.New("cannot issue a token for your own account")

// AccountDetails contains only the fields permitted in root account management.
type AccountDetails struct {
	Username string
	Role     models.UserRole
	Status   string
}

type AccountTokenStore interface {
	ProfileAccount(context.Context, int32) (AccountDetails, error)
	ReplaceAccountToken(ctx context.Context, session uuid.UUID, profileID int32, digest [32]byte) error
	RedeemAccountToken(ctx context.Context, username string, digest [32]byte, password string) error
}
