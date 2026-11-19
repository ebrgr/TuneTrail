package admin

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/google/uuid"
)

type ProfileAccountStore interface {
	SessionReader
	UpdateProfileAccountRole(context.Context, uuid.UUID, int32, models.UserRole) error
	db.AccountTokenStore
	GetListeningProfileByID(context.Context, int32) (*models.ListeningProfile, error)
}

// AuthorizeAccountTarget resolves the live owner before applying the account capability.
func AuthorizeAccountTarget(ctx context.Context, store ProfileAccountStore, user *models.User, profileID int32, action Action) (*models.ListeningProfile, error) {
	if err := AuthorizeAction(user, ManageAccounts, nil); err != nil {
		return nil, err
	}
	profile, err := store.GetListeningProfileByID(ctx, profileID)
	if err != nil {
		return nil, err
	}
	return profile, AuthorizeAction(user, action, profile)
}

// IssueProfileAccountToken returns the only plaintext copy of the credential.
func IssueProfileAccountToken(ctx context.Context, store ProfileAccountStore, session uuid.UUID, profileID int32) (string, error) {
	identity, err := Authenticate(ctx, store, session)
	if err != nil {
		return "", err
	}
	if _, err := AuthorizeAccountTarget(ctx, store, identity.User, profileID, IssueAccountToken); err != nil {
		return "", err
	}
	var secret [32]byte
	if _, err := rand.Read(secret[:]); err != nil {
		return "", err
	}
	digest := sha256.Sum256(secret[:])
	if err := store.ReplaceAccountToken(ctx, session, profileID, digest); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(secret[:]), nil
}

// ChangeProfileAccountRole updates only the role of the server-resolved account.
func ChangeProfileAccountRole(ctx context.Context, store ProfileAccountStore, session uuid.UUID, profileID int32, role models.UserRole) error {
	identity, err := Authenticate(ctx, store, session)
	if err != nil {
		return err
	}
	if _, err := AuthorizeAccountTarget(ctx, store, identity.User, profileID, ChangeAccountRole); err != nil {
		return err
	}
	return store.UpdateProfileAccountRole(ctx, session, profileID, role)
}

// RedeemProfileAccountToken validates input and delegates single-use consumption.
func RedeemProfileAccountToken(ctx context.Context, store db.AccountTokenStore, username, token, password string) error {
	if !validAccountPassword(password) {
		return db.ErrInvalidInput
	}
	secret, err := base64.RawURLEncoding.Strict().DecodeString(token)
	if err != nil || len(secret) != 32 || !usernamePattern.MatchString(username) {
		return db.ErrAccountToken
	}
	return store.RedeemAccountToken(ctx, username, sha256.Sum256(secret), password)
}
