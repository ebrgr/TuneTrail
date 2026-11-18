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

// insertAccountProfile replaces the retired trigger inside the caller's transaction.
func insertAccountProfile(ctx context.Context, tx *sql.Tx, id int64, username string) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO listening_profiles(user_id,slug,display_name,created_at)
    VALUES(?,CASE WHEN NOT EXISTS(SELECT 1 FROM listening_profiles) THEN 'default' ELSE 'user-' || ? END,?,unixepoch())`, id, id, username)
	return err
}

// insertPendingAccount allocates a bounded, deterministic login without sharing passwords.
func insertPendingAccount(ctx context.Context, tx *sql.Tx, slug string) (int64, error) {
	base := slug
	if len(base) > 32 {
		base = base[:32]
	}
	for n := 1; n <= 9; n++ {
		username := base
		if n > 1 {
			if len(username) > 30 {
				username = username[:30]
			}
			username = fmt.Sprintf("%s-%d", username, n)
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
