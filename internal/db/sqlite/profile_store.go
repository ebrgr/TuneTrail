package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/google/uuid"
	sqlite3 "modernc.org/sqlite"
)

const profileColumns = "id, slug, display_name, avatar_image_id, avatar_icon, created_at, user_id, require_login"

// scanProfile includes canonical ownership without exposing it in public serialization.
func scanProfile(row interface{ Scan(...any) error }) (*models.ListeningProfile, error) {
	p := new(models.ListeningProfile)
	err := row.Scan(&p.ID, &p.Slug, &p.DisplayName, &p.AvatarImageID, &p.AvatarIcon, &p.CreatedAt, &p.UserID, &p.RequireLogin)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, db.ErrNotFound
	}
	return p, err
}
func (s *Sqlite) ListListeningProfiles(ctx context.Context) ([]models.ListeningProfile, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+profileColumns+` FROM listening_profiles ORDER BY display_name,id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	profiles := make([]models.ListeningProfile, 0)
	for rows.Next() {
		p, err := scanProfile(rows)
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, *p)
	}
	return profiles, rows.Err()
}
func (s *Sqlite) GetListeningProfile(ctx context.Context, slug string) (*models.ListeningProfile, error) {
	return scanProfile(s.db.QueryRowContext(ctx, `SELECT `+profileColumns+` FROM listening_profiles WHERE slug=?`, slug))
}
func (s *Sqlite) GetListeningProfileByID(ctx context.Context, id int32) (*models.ListeningProfile, error) {
	return scanProfile(s.db.QueryRowContext(ctx, `SELECT `+profileColumns+` FROM listening_profiles WHERE id=?`, id))
}

// GetProfileByAPIKey resolves the ingestion destination, never an authenticated actor.
func (s *Sqlite) GetProfileByAPIKey(ctx context.Context, key string) (*models.ListeningProfile, error) {
	return scanProfile(s.db.QueryRowContext(ctx, `SELECT p.id,p.slug,p.display_name,p.avatar_image_id,p.avatar_icon,p.created_at,p.user_id,p.require_login FROM listening_profiles p JOIN api_keys k ON k.profile_id=p.id WHERE k.key=?`, key))
}

var profileSlugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// UpdateListeningProfile distinguishes duplicate identity from infrastructure failures.
func (s *Sqlite) UpdateListeningProfile(ctx context.Context, id int32, displayName, slug string, requireLogin *bool) error {
	if !validProfileIdentity(displayName, slug) {
		return db.ErrInvalidInput
	}
	// Leave visibility untouched when older callers update only the identity.
	result, err := s.db.ExecContext(ctx, `UPDATE listening_profiles SET display_name=?,slug=?,require_login=COALESCE(?,require_login) WHERE id=?`, displayName, slug, requireLogin, id)
	if err != nil {
		return profileConstraint(err)
	}
	n, err := result.RowsAffected()
	if err == nil && n == 0 {
		return db.ErrNotFound
	}
	return err
}
func (s *Sqlite) GetApiKeysByProfileID(ctx context.Context, id int32) ([]models.ApiKey, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,key,COALESCE(user_id,0),profile_id,label,created_at FROM api_keys WHERE profile_id=? ORDER BY id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	keys := make([]models.ApiKey, 0)
	for rows.Next() {
		var k models.ApiKey
		var created int64
		if err := rows.Scan(&k.ID, &k.Key, &k.UserID, &k.ProfileID, &k.Label, &created); err != nil {
			return nil, err
		}
		k.CreatedAt = time.Unix(created, 0).UTC()
		keys = append(keys, k)
	}
	return keys, rows.Err()
}
func (s *Sqlite) DeleteProfileApiKey(ctx context.Context, id, profileID int32) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM api_keys WHERE id=? AND profile_id=?`, id, profileID)
	return affectedProfileRow(result, err)
}
func (s *Sqlite) UpdateProfileApiKeyLabel(ctx context.Context, id, profileID int32, label string) error {
	result, err := s.db.ExecContext(ctx, `UPDATE api_keys SET label=? WHERE id=? AND profile_id=?`, label, id, profileID)
	return affectedProfileRow(result, err)
}
func affectedProfileRow(result sql.Result, err error) error {
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err == nil && n == 0 {
		return db.ErrNotFound
	}
	return err
}
func (s *Sqlite) GetProfileRelay(ctx context.Context, id int32) (string, string, error) {
	var url, token string
	err := s.db.QueryRowContext(ctx, `SELECT url,token FROM profile_relays WHERE profile_id=?`, id).Scan(&url, &token)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", nil
	}
	return url, token, err
}

// Record the migration decision once. Revoking a relay or deleting its profile
// must never cause the old credentials to be reassigned on a later restart.
func (s *Sqlite) InitializeLegacyRelay(ctx context.Context, url, token string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM legacy_relay_binding`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	var profileID sql.NullInt64
	if err := tx.QueryRowContext(ctx, `SELECT CASE WHEN (SELECT COUNT(*) FROM listening_profiles)=1 THEN
   (SELECT m.id FROM listening_profiles m JOIN users u ON u.id=m.user_id WHERE u.role='admin' LIMIT 1) END`).Scan(&profileID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO legacy_relay_binding(id,profile_id) VALUES (1,?)`, profileID); err != nil {
		return err
	}
	if profileID.Valid {
		if _, err := tx.ExecContext(ctx, `INSERT INTO profile_relays(profile_id,url,token) VALUES (?,?,?) ON CONFLICT(profile_id) DO NOTHING`, profileID.Int64, url, token); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	if !profileID.Valid {
		return fmt.Errorf("legacy relay not associated: select and configure a profile explicitly")
	}
	return nil
}
func (s *Sqlite) SetProfileRelay(ctx context.Context, id int32, url, token string) error {
	if url == "" && token == "" {
		_, err := s.db.ExecContext(ctx, `DELETE FROM profile_relays WHERE profile_id=?`, id)
		return err
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO profile_relays(profile_id,url,token) VALUES (?,?,?) ON CONFLICT(profile_id) DO UPDATE SET url=excluded.url,token=excluded.token`, id, url, token)
	// Expose an incompatible relay configuration as an administrative conflict.
	if isOccurrenceGuard(err, "Relay conflicts") {
		return fmt.Errorf("SetProfileRelay: %w: %w", db.ErrConflict, err)
	}
	return err
}

func validProfileIdentity(name, slug string) bool {
	return strings.TrimSpace(name) != "" && len(name) <= 200 && slug != "assets" && len(slug) <= 64 && profileSlugPattern.MatchString(slug)
}

// CreateListeningProfile writes a pending account, its public profile and initial key atomically.
func (s *Sqlite) CreateListeningProfile(ctx context.Context, name, slug, icon, token string) (*models.ListeningProfile, error) {
	return s.createListeningProfile(ctx, uuid.Nil, name, slug, icon, token)
}

// CreateListeningProfileAsRoot verifies the live root session in the creation transaction.
func (s *Sqlite) CreateListeningProfileAsRoot(ctx context.Context, session uuid.UUID, name, slug, icon, token string) (*models.ListeningProfile, error) {
	if session == uuid.Nil {
		return nil, db.ErrConflict
	}
	return s.createListeningProfile(ctx, session, name, slug, icon, token)
}

func (s *Sqlite) createListeningProfile(ctx context.Context, session uuid.UUID, name, slug, icon, token string) (*models.ListeningProfile, error) {
	if !validProfileIdentity(name, slug) {
		return nil, db.ErrInvalidInput
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	// Keep root session validation in the same write transaction as the new account.
	if session != uuid.Nil {
		if _, err := lockRootAccount(ctx, tx, session); err != nil {
			return nil, err
		}
	}
	userID, err := insertPendingAccount(ctx, tx, slug)
	if err != nil {
		return nil, err
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO listening_profiles(user_id,slug,display_name,avatar_icon,created_at) VALUES (?,?,?,NULLIF(?,''),unixepoch())`, userID, slug, strings.TrimSpace(name), icon)
	if err != nil {
		return nil, profileConstraint(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO api_keys(key,profile_id,label,created_at) VALUES (?,?,'Initial key',unixepoch())`, token, id)
	if err != nil {
		return nil, err
	}
	profile, err := scanProfile(tx.QueryRowContext(ctx, `SELECT `+profileColumns+` FROM listening_profiles WHERE id=?`, id))
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return profile, nil
}
func (s *Sqlite) SetProfileAvatar(ctx context.Context, id int32, image, icon *string) error {
	result, err := s.db.ExecContext(ctx, `UPDATE listening_profiles SET avatar_image_id=?,avatar_icon=? WHERE id=?`, image, icon, id)
	return affectedProfileRow(result, err)
}
func (s *Sqlite) ProfileListenCount(ctx context.Context, id int32) (int64, error) {
	var count int64
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM listens WHERE profile_id=?`, id).Scan(&count)
	return count, err
}

// DeleteListeningProfile removes the confirmed pair and sessions, preserving the last admin.
// Profile cascades also remove all receipt history and integrations.
func (s *Sqlite) DeleteListeningProfile(ctx context.Context, id int32, name string, count int64) error {
	return s.deleteListeningProfile(ctx, uuid.Nil, id, "", name, count)
}

// DeleteListeningProfileAsRoot verifies the live root session and confirmed profile in the delete transaction.
func (s *Sqlite) DeleteListeningProfileAsRoot(ctx context.Context, session uuid.UUID, id int32, slug, name string, count int64) error {
	if session == uuid.Nil || slug == "" {
		return db.ErrConflict
	}
	return s.deleteListeningProfile(ctx, session, id, slug, name, count)
}

func (s *Sqlite) deleteListeningProfile(ctx context.Context, session uuid.UUID, id int32, slug, name string, count int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Acquire the write lock before observing administrator counts.
	if session != uuid.Nil {
		if _, err := lockRootAccount(ctx, tx, session); err != nil {
			return err
		}
	}
	var userID int64
	err = tx.QueryRowContext(ctx, `DELETE FROM listening_profiles WHERE id=? AND display_name=? AND (?='' OR slug=?) AND (SELECT COUNT(*) FROM listens WHERE profile_id=?)=? RETURNING user_id`, id, name, slug, slug, id, count).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return db.ErrNotFound
	}
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `DELETE FROM users WHERE id=? AND (role<>'admin' OR status<>'active' OR (SELECT COUNT(*) FROM users WHERE role='admin' AND status='active')>1)`, userID)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return db.ErrLastAdmin
	}
	return tx.Commit()
}

// profileConstraint maps SQLite uniqueness failures without hiding operational errors.
func profileConstraint(err error) error {
	var e *sqlite3.Error
	if errors.As(err, &e) && (e.Code() == 2067 || e.Code() == 1555) {
		return fmt.Errorf("profile identity: %w", db.ErrConflict)
	}
	return err
}
