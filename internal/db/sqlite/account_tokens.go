package sqlite

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	activationTokenTTL = 24 * time.Hour
	resetTokenTTL      = 30 * time.Minute
)

// ProfileAccount loads only the fields the account management view uses.
func (s *Sqlite) ProfileAccount(ctx context.Context, profileID int32) (db.AccountDetails, error) {
	var a db.AccountDetails
	err := s.db.QueryRowContext(ctx, `SELECT u.username,u.role,u.status FROM users u
		JOIN listening_profiles p ON p.user_id=u.id WHERE p.id=?`, profileID).Scan(&a.Username, &a.Role, &a.Status)
	if errors.Is(err, sql.ErrNoRows) {
		err = db.ErrNotFound
	}
	return a, err
}

// ReplaceAccountToken revalidates the issuer and picks the purpose in the write transaction.
func (s *Sqlite) ReplaceAccountToken(ctx context.Context, session uuid.UUID, profileID int32, digest [32]byte) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	actor, err := lockRootAccount(ctx, tx, session)
	if err != nil {
		return err
	}
	var target int32
	var status string
	err = tx.QueryRowContext(ctx, `SELECT u.id,u.status FROM users u
		JOIN listening_profiles p ON p.user_id=u.id WHERE p.id=?`, profileID).Scan(&target, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return db.ErrNotFound
	}
	if err != nil {
		return err
	}
	if target == actor {
		return db.ErrOwnAccountToken
	}
	purpose, ttl := "reset", resetTokenTTL
	if status == "pending" {
		purpose, ttl = "activation", activationTokenTTL
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO account_tokens(user_id,token_hash,purpose,expires_at)
		VALUES(?,?,?,unixepoch()+?) ON CONFLICT(user_id) DO UPDATE SET
		token_hash=excluded.token_hash,purpose=excluded.purpose,expires_at=excluded.expires_at`,
		target, digest[:], purpose, int64(ttl/time.Second))
	if err != nil {
		return err
	}
	return tx.Commit()
}

// RedeemAccountToken compares a fixed-size digest and consumes the credential with the password change.
func (s *Sqlite) RedeemAccountToken(ctx context.Context, username string, digest [32]byte, password string) error {
	pw, err := validateAndNormalizePassword(password)
	if err != nil {
		return db.ErrInvalidInput
	}
	// Use the same bcrypt work for absent accounts and all invalid tokens, outside the write lock.
	hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		return db.ErrInvalidInput
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var id int32
	var stored [32]byte
	var saved []byte
	err = tx.QueryRowContext(ctx, `UPDATE account_tokens SET expires_at=expires_at
		WHERE user_id=(SELECT id FROM users WHERE username=? AND status=CASE account_tokens.purpose
			WHEN 'activation' THEN 'pending' ELSE 'active' END)
		AND expires_at>unixepoch() RETURNING user_id,token_hash`, strings.ToLower(username)).Scan(&id, &saved)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	copy(stored[:], saved)
	if match := subtle.ConstantTimeCompare(stored[:], digest[:]); match != 1 || id == 0 {
		return db.ErrAccountToken
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM account_tokens WHERE user_id=?`, id); err != nil {
		return err
	}
	if err := setAccountPassword(ctx, tx, id, hash); err != nil {
		return err
	}
	return tx.Commit()
}
