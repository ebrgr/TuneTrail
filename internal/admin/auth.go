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

var ErrUnauthorized = errors.New("authenticated session required")
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

// Action is an explicit capability; unknown actions are always denied.
type Action string

const (
	ReadListening     Action = "profile.read-listening"
	ManageProfile     Action = "profile.manage"
	ManageAccount     Action = "account.self"
	ManageAccounts    Action = "accounts.manage"
	IssueAccountToken Action = "accounts.issue-token"
	ChangeAccountRole Action = "accounts.change-role"
	ManageCatalog     Action = "catalog.manage"
	ManageServer      Action = "server.manage"
)

// AuthorizeAction is the shared HTML, JSON and service policy. Actors come from
// session authentication; resources must be loaded by the server, never decoded from input.
func AuthorizeAction(user *models.User, action Action, resource *models.ListeningProfile) error {
	// Protected listening uses the same owner/root decision as profile management.
	if action == ReadListening {
		if resource == nil || resource.ID <= 0 || resource.UserID <= 0 {
			return ErrForbidden
		}
		if !resource.RequireLogin {
			return nil
		}
		return AuthorizeAction(user, ManageProfile, resource)
	}
	if user == nil {
		return ErrUnauthorized
	}
	if user.Status != "active" || user.ID <= 0 {
		return ErrForbidden
	}
	switch action {
	case ManageProfile:
		if resource == nil || resource.ID <= 0 || resource.UserID <= 0 {
			return ErrForbidden
		}
		if user.Role == models.UserRoleAdmin || user.Role == models.UserRoleUser && resource.UserID == user.ID {
			return nil
		}
	case ManageAccount:
		if user.Role == models.UserRoleAdmin || user.Role == models.UserRoleUser {
			return nil
		}
	case IssueAccountToken, ChangeAccountRole:
		// Account operations use the server-resolved profile owner as the target.
		if user.Role == models.UserRoleAdmin && resource != nil && resource.ID > 0 && resource.UserID > 0 {
			if action == IssueAccountToken && resource.UserID == user.ID {
				return db.ErrOwnAccountToken
			}
			return nil
		}
	case ManageAccounts, ManageCatalog, ManageServer:
		if user.Role == models.UserRoleAdmin {
			return nil
		}
	}
	return ErrForbidden
}

// Authorize is root-only policy shorthand for adapters with an authenticated actor.
func Authorize(user *models.User) error { return AuthorizeAction(user, ManageServer, nil) }

// AuthorizeProfile loads the canonical owner before granting a personal capability.
func AuthorizeProfile(ctx context.Context, store interface {
	GetListeningProfileByID(context.Context, int32) (*models.ListeningProfile, error)
}, user *models.User, id int32) (*models.ListeningProfile, error) {
	if user == nil {
		return nil, ErrUnauthorized
	}
	p, err := store.GetListeningProfileByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := AuthorizeService(ctx, store, user, ManageProfile, p); err != nil {
		return nil, err
	}
	return p, nil
}

// AuthorizeService re-resolves the human session at the service boundary. An
// ingestion-attributed User or a stale role object cannot act as a human session.
func AuthorizeService(ctx context.Context, store any, user *models.User, action Action, resource *models.ListeningProfile) error {
	if user == nil || user.SessionID == uuid.Nil {
		return ErrUnauthorized
	}
	reader, ok := store.(SessionReader)
	if !ok {
		return ErrForbidden
	}
	identity, err := Authenticate(ctx, reader, user.SessionID)
	if err != nil {
		return err
	}
	if identity.User.ID != user.ID {
		return ErrForbidden
	}
	return AuthorizeAction(identity.User, action, resource)
}

// Authenticate resolves an active actor independently of role, without extending session lifetime.
func Authenticate(ctx context.Context, store SessionReader, id uuid.UUID) (Identity, error) {
	if id == uuid.Nil {
		return Identity{}, ErrUnauthorized
	}
	user, err := store.GetUserBySession(ctx, id)
	if err != nil {
		return Identity{}, fmt.Errorf("authenticate: %w", err)
	}
	if user == nil || user.Status != "active" {
		return Identity{}, ErrUnauthorized
	}
	actor := *user
	actor.SessionID = id
	return Identity{User: &actor, SessionID: id}, nil
}

// invalidLoginHash is generated once at the same cost used by stored credentials.
var invalidLoginHash = func() []byte {
	hash, err := bcrypt.GenerateFromPassword([]byte("tunetrail-invalid-login"), bcrypt.DefaultCost)
	if err != nil {
		panic(fmt.Errorf("initialize login hash: %w", err))
	}
	return hash
}()

// Login performs bcrypt even for missing or pending accounts and binds sessions to the active hash.
func Login(ctx context.Context, store LoginStore, username, password string, remember bool) (*models.Session, error) {
	return loginWithCompare(ctx, store, username, password, remember, bcrypt.CompareHashAndPassword)
}

// loginWithCompare keeps the bcrypt work observable in tests without mutable global hooks.
func loginWithCompare(ctx context.Context, store LoginStore, username, password string, remember bool, compare func([]byte, []byte) error) (*models.Session, error) {
	if username == "" || password == "" {
		return nil, db.ErrInvalidInput
	}
	user, err := store.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("login: %w", err)
	}
	active := user != nil && user.Status == "active"
	hash := invalidLoginHash
	if active {
		hash = user.Password
	}
	comparison := compare(hash, []byte(password))
	if !active || comparison != nil {
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

// validAccountPassword applies the existing password change limits.
func validAccountPassword(password string) bool {
	return utf8.RuneCountInString(password) >= 8 && len(password) <= 72
}

// ChangeAccount validates explicit fields; the store atomically revokes all sessions on a password change.
func ChangeAccount(ctx context.Context, store AccountStore, user *models.User, username, password *string) (bool, error) {
	if err := AuthorizeService(ctx, store, user, ManageAccount, nil); err != nil {
		return false, err
	}
	if username == nil && password == nil {
		return false, db.ErrInvalidInput
	}
	if username != nil && !usernamePattern.MatchString(*username) {
		return false, db.ErrInvalidInput
	}
	if password != nil && !validAccountPassword(*password) {
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
