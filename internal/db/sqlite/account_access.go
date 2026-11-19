package sqlite

import (
	"context"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/google/uuid"
)

// UpdateAccountAccess acquires the SQLite write lock before resolving the actor
// or counting active admins. No process mutex or stale role snapshot is trusted.
func (s *Sqlite) UpdateAccountAccess(ctx context.Context, session uuid.UUID, target int32, role models.UserRole, status string) error {
	if target <= 0 || (role != models.UserRoleAdmin && role != models.UserRoleUser) || (status != "active" && status != "pending") {
		return db.ErrInvalidInput
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE users SET username=username WHERE role='admin' AND status='active' AND id=(SELECT user_id FROM sessions WHERE id=? AND expires_at>unixepoch())`, session.String())
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return db.ErrConflict
	}
	var oldRole, oldStatus string
	if err = tx.QueryRowContext(ctx, `SELECT role,status FROM users WHERE id=?`, target).Scan(&oldRole, &oldStatus); err != nil {
		return err
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
