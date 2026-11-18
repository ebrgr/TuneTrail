package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/ebrgr/TuneTrail/internal/db"
	"golang.org/x/crypto/bcrypt"
)

// accountPassword preserves the existing validators, normalization and bcrypt cost.
func accountPassword(username, password string) ([]byte, error) {
	if err := validateUsername(username); err != nil {
		return nil, err
	}
	pw, err := validateAndNormalizePassword(password)
	if err != nil {
		return nil, err
	}
	return bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
}

// identifierCandidate keeps deterministic suffixes inside an ASCII identifier's size limit.
func identifierCandidate(base string, maxLength, attempt int) string {
	suffix := ""
	if attempt > 1 {
		suffix = fmt.Sprintf("-%d", attempt)
	}
	if len(base) > maxLength-len(suffix) {
		base = base[:maxLength-len(suffix)]
	}
	return base + suffix
}

// insertAccountProfile replaces the retired trigger and avoids occupied public slugs.
func insertAccountProfile(ctx context.Context, tx *sql.Tx, id int64, username string) error {
	var occupied bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM listening_profiles)`).Scan(&occupied); err != nil {
		return err
	}
	base := fmt.Sprintf("user-%d", id)
	if !occupied {
		base = "default"
	}
	for attempt := 1; attempt <= 9; attempt++ {
		slug := identifierCandidate(base, 64, attempt)
		result, err := tx.ExecContext(ctx, `INSERT INTO listening_profiles(user_id,slug,display_name,created_at)
		SELECT ?,?,?,unixepoch() WHERE NOT EXISTS(SELECT 1 FROM listening_profiles WHERE slug=?)`, id, slug, username, slug)
		if err != nil {
			return err
		}
		n, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if n == 1 {
			return nil
		}
	}
	return db.ErrConflict
}

// insertPendingAccount allocates a bounded, deterministic login without sharing passwords.
func insertPendingAccount(ctx context.Context, tx *sql.Tx, slug string) (int64, error) {
	// Slugs are lowercase ASCII; validate the bounded candidate with the login validator too.
	base := strings.ToLower(slug)
	for n := 1; n <= 9; n++ {
		username := identifierCandidate(base, maxUsernameLength, n)
		if err := validateUsername(username); err != nil {
			return 0, fmt.Errorf("pending account username: %w: %v", db.ErrInvalidInput, err)
		}
		result, err := tx.ExecContext(ctx, `INSERT INTO users(username,role,password,status) SELECT ?,'user',X'','pending' WHERE NOT EXISTS(SELECT 1 FROM users WHERE username=?)`, username, username)
		if err != nil {
			return 0, err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return 0, err
		}
		if affected == 1 {
			return result.LastInsertId()
		}
	}
	return 0, db.ErrConflict
}

// BootstrapAdministrator creates the first account, profile and key as one write transaction.
// Existing accounts, even without an administrator, never trigger automatic elevation.
func (s *Sqlite) BootstrapAdministrator(ctx context.Context, opts db.BootstrapAdminOpts) (bool, error) {
	hash, err := accountPassword(opts.Username, opts.Password)
	if err != nil {
		return false, fmt.Errorf("bootstrap password: %w", err)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	username := strings.ToLower(opts.Username)
	result, err := tx.ExecContext(ctx, `INSERT INTO users(username,role,password,status) SELECT ?,'admin',?,'active' WHERE NOT EXISTS(SELECT 1 FROM users)`, username, hash)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if n == 0 {
		return false, nil
	}
	id, err := result.LastInsertId()
	if err != nil {
		return false, err
	}
	if err = insertAccountProfile(ctx, tx, id, username); err != nil {
		return false, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO api_keys(key,user_id,profile_id,label,created_at) SELECT ?,?,id,'Default',unixepoch() FROM listening_profiles WHERE user_id=?`, opts.APIKey, id, id)
	if err != nil {
		return false, err
	}
	if err = tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}
