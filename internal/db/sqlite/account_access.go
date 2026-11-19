package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/google/uuid"
)

// UpdateAccountAccess acquires the SQLite write lock before resolving the actor
// or counting active admins. No process mutex or stale role snapshot is trusted.
func (s *Sqlite) UpdateAccountAccess(ctx context.Context, session uuid.UUID, target int32, role models.UserRole, status string) error {
	if status != "active" && status != "pending" {
		return db.ErrInvalidInput
	}
	return s.updateAccountAccess(ctx, session, target, role, status, false)
}

// UpdateProfileAccountRole resolves the stable profile under the write lock and preserves the account status.
func (s *Sqlite) UpdateProfileAccountRole(ctx context.Context, session uuid.UUID, profileID int32, role models.UserRole) error {
	return s.updateAccountAccess(ctx, session, profileID, role, "", true)
}

func (s *Sqlite) updateAccountAccess(ctx context.Context, session uuid.UUID, target int32, role models.UserRole, status string, byProfile bool) error {
	if target <= 0 || (role != models.UserRoleAdmin && role != models.UserRoleUser) {
		return db.ErrInvalidInput
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := lockRootAccount(ctx, tx, session); err != nil {
		return err
	}
	if byProfile {
		// Account IDs can be reused after deletion. Resolve from the stable profile ID.
		err := tx.QueryRowContext(ctx, `SELECT user_id FROM listening_profiles WHERE id=?`, target).Scan(&target)
		if errors.Is(err, sql.ErrNoRows) {
			return db.ErrNotFound
		}
		if err != nil {
			return err
		}
	}
	var oldRole, oldStatus string
	if err = tx.QueryRowContext(ctx, `SELECT role,status FROM users WHERE id=?`, target).Scan(&oldRole, &oldStatus); err != nil {
		return err
	}
	// A role-only request preserves the status read under the write lock.
	if status == "" {
		status = oldStatus
	}
	if oldRole == "admin" && oldStatus == "active" && (role != models.UserRoleAdmin || status != "active") {
		var count int
		if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE role='admin' AND status='active'`).Scan(&count); err != nil {
			return err
		}
		if count <= 1 {
			return db.ErrLastAdmin
		}
	}
	// Pending accounts have no password; reactivation requires the existing password/token path.
	if oldStatus == "pending" && status == "active" {
		return db.ErrInvalidInput
	}
	if _, err = tx.ExecContext(ctx, `UPDATE users SET role=?,status=?,password=CASE WHEN ?='pending' THEN X'' ELSE password END WHERE id=?`, string(role), status, status, target); err != nil {
		return err
	}
	if status != "active" {
		if _, err = tx.ExecContext(ctx, `DELETE FROM sessions WHERE user_id=?`, target); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// lockRootAccount acquires the write lock before checking the current issuer.
func lockRootAccount(ctx context.Context, tx *sql.Tx, session uuid.UUID) (int32, error) {
	var actor int32
	err := tx.QueryRowContext(ctx, `UPDATE users SET username=username
		WHERE role='admin' AND status='active' AND id=(SELECT user_id FROM sessions WHERE id=? AND expires_at>unixepoch())
		RETURNING id`, session.String()).Scan(&actor)
	if errors.Is(err, sql.ErrNoRows) {
		err = db.ErrConflict
	}
	return actor, err
}
