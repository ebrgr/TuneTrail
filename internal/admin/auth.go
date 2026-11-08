// Package admin holds administrative rules shared by JSON and HTML adapters.
package admin

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"
	"unicode/utf8"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var ErrUnauthorized = errors.New("administrator session required")
var ErrForbidden = errors.New("administrator required")
var ErrUnprocessable = errors.New("check form fields")

// Identity is resolved from an authenticated session, never from a profile or API key.
type Identity struct {
	User      *models.User
	SessionID uuid.UUID
}
type SessionReader interface {
	GetUserBySession(context.Context, uuid.UUID) (*models.User, error)
}
type LoginStore interface {
	GetUserByUsername(context.Context, string) (*models.User, error)
	SaveAuthenticatedSession(context.Context, int32, []byte, time.Time, bool) (*models.Session, error)
}
type AccountStore interface {
	UpdateUser(context.Context, db.UpdateUserOpts) error
}

// Authorize keeps privilege checks available to non-HTTP callers of admin services.
func Authorize(user *models.User) error {
	if user == nil {
		return ErrUnauthorized
	}
	if user.Role != models.UserRoleAdmin {
		return ErrForbidden
	}
	return nil
}

// Authenticate reads a fixed-expiry session without extending its lifetime.
func Authenticate(ctx context.Context, store SessionReader, id uuid.UUID) (Identity, error) {
	if id == uuid.Nil {
		return Identity{}, ErrUnauthorized
	}
	user, err := store.GetUserBySession(ctx, id)
	if err != nil {
		return Identity{}, fmt.Errorf("authenticate: %w", err)
	}
	if err := Authorize(user); err != nil {
		return Identity{}, err
	}
	return Identity{User: user, SessionID: id}, nil
}

// Login checks bcrypt outside SQLite locks, then atomically binds the new session to that hash.
func Login(ctx context.Context, store LoginStore, username, password string, remember bool) (*models.Session, error) {
	if username == "" || password == "" {
		return nil, db.ErrInvalidInput
	}
	user, err := store.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("login: %w", err)
	}
	if user == nil || user.Role != models.UserRoleAdmin {
		return nil, ErrUnauthorized
	}
	if bcrypt.CompareHashAndPassword(user.Password, []byte(password)) != nil {
		return nil, ErrUnauthorized
	}
	lifetime := 24 * time.Hour
	if remember {
		lifetime = 30 * 24 * time.Hour
	}
	session, err := store.SaveAuthenticatedSession(ctx, user.ID, user.Password, time.Now().Add(lifetime).Truncate(time.Second), remember)
	if errors.Is(err, db.ErrConflict) {
		return nil, ErrUnauthorized
	}
	if err != nil {
		return nil, fmt.Errorf("login session: %w", err)
	}
	return session, nil
}

var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_.-]{1,32}$`)

// ChangeAccount validates explicit fields; the store atomically revokes all sessions on a password change.
func ChangeAccount(ctx context.Context, store AccountStore, user *models.User, username, password *string) (bool, error) {
	if err := Authorize(user); err != nil {
		return false, err
	}
	if username == nil && password == nil {
		return false, db.ErrInvalidInput
	}
	if username != nil && !usernamePattern.MatchString(*username) {
		return false, db.ErrInvalidInput
	}
	if password != nil && (utf8.RuneCountInString(*password) < 8 || len(*password) > 72) {
		return false, db.ErrInvalidInput
	}
	opts := db.UpdateUserOpts{ID: user.ID}
	if username != nil {
		opts.Username = *username
	}
	if password != nil {
		opts.Password = *password
	}
	if err := store.UpdateUser(ctx, opts); err != nil {
		return false, fmt.Errorf("update account: %w", err)
	}
	return password != nil, nil
}

// Logout is idempotent and reports storage failures before the HTTP adapter clears its cookie.
func Logout(ctx context.Context, store interface {
	DeleteSession(context.Context, uuid.UUID) error
}, id uuid.UUID) error {
	if id == uuid.Nil {
		return nil
	}
	if err := store.DeleteSession(ctx, id); err != nil {
		return fmt.Errorf("logout: %w", err)
	}
	return nil
}
