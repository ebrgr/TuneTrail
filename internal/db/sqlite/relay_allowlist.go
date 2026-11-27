package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"net/netip"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/relay"
	"github.com/google/uuid"
)

// RelayPermission separates cleartext permission from access to reserved addresses.
type RelayPermission = relay.Permission

// SetRelayPermission saves a validated operator exception for one profile.
func (s *Sqlite) SetRelayPermission(ctx context.Context, profile int32, permission RelayPermission) error {
	return s.setRelayPermission(ctx, uuid.Nil, profile, permission, nil)
}

// SetRelayPermissionAsRoot checks a live instance administrator in the write transaction.
func (s *Sqlite) SetRelayPermissionAsRoot(ctx context.Context, session uuid.UUID, profile int32, permission relay.Permission) error {
	if session == uuid.Nil {
		return db.ErrForbidden
	}
	return s.setRelayPermission(ctx, session, profile, permission, nil)
}

// setRelayPermission persists one exception and fences previous policy snapshots.
func (s *Sqlite) setRelayPermission(ctx context.Context, session uuid.UUID, profile int32, permission relay.Permission, old *relay.Exception) error {
	host, err := normalizeRelayHostPort(permission.HostPort)
	if err != nil {
		return fmt.Errorf("relay host and port: %w", db.ErrInvalidInput)
	}
	cidr := ""
	if permission.Prefix.IsValid() {
		prefix := permission.Prefix.Masked()
		if !permission.Private || !relay.ValidExceptionPrefix(prefix) || !relay.IsReservedPrefix(prefix) {
			return fmt.Errorf("invalid relay exception prefix: %w", db.ErrInvalidInput)
		}
		cidr = prefix.String()
	}
	if (!permission.HTTP && !permission.Private) || (permission.Private && cidr == "") {
		return fmt.Errorf("invalid relay permissions: %w", db.ErrInvalidInput)
	}
	return s.withIngestWriter(ctx, func(tx *sql.Conn) error {
		if session != uuid.Nil {
			if err := checkIngestRoot(ctx, tx, session); err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO relay_destinations(profile_id) VALUES(?) ON CONFLICT DO NOTHING", profile); err != nil {
			return err
		}
		if old != nil {
			removed, err := tx.ExecContext(ctx, "DELETE FROM relay_allowlist WHERE profile_id=? AND host_port=? AND cidr=? AND origin=?", profile, old.HostPort, old.CIDR, old.Origin)
			if err != nil {
				return err
			}
			n, err := removed.RowsAffected()
			if err != nil {
				return err
			}
			if n != 1 {
				return db.ErrConflict
			}
		}
		result, err := tx.ExecContext(ctx, `INSERT INTO relay_allowlist VALUES(?,?,?,?,?,'manual')
			ON CONFLICT(profile_id,host_port,cidr,origin) DO UPDATE SET allow_http=excluded.allow_http,allow_private=excluded.allow_private
			WHERE allow_http<>excluded.allow_http OR allow_private<>excluded.allow_private`,
			profile, host, cidr, permission.HTTP, permission.Private)
		if err != nil {
			return err
		}
		n, err := result.RowsAffected()
		if err != nil || (n == 0 && old == nil) {
			return err
		}
		// An operator policy change clears a policy block and fences old transport results.
		_, err = tx.ExecContext(ctx, `UPDATE relay_destinations SET config_version=config_version+1,
			paused_until=CASE WHEN (pause_class IN ('policy_blocked','config_invalid') OR pause_class LIKE 'policy_blocked:%') THEN NULL ELSE paused_until END,
			pause_class=CASE WHEN (pause_class IN ('policy_blocked','config_invalid') OR pause_class LIKE 'policy_blocked:%') THEN NULL ELSE pause_class END WHERE profile_id=?`, profile)
		return err
	})
}

// RelayPermissions reads a profile-scoped permission snapshot without exposing credentials.
func (s *Sqlite) RelayPermissions(ctx context.Context, profile int32) (relay.Permissions, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT host_port,cidr,allow_http,allow_private FROM relay_allowlist WHERE profile_id=?", profile)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var permissions relay.Permissions
	for rows.Next() {
		var p RelayPermission
		var cidr string
		if err := rows.Scan(&p.HostPort, &cidr, &p.HTTP, &p.Private); err != nil {
			return nil, err
		}
		if canonical, err := normalizeRelayHostPort(p.HostPort); err != nil || canonical != p.HostPort {
			return nil, relay.ErrConfigInvalid
		}
		if cidr != "" {
			p.Prefix, err = netip.ParsePrefix(cidr)
			if err != nil || !p.Private || !relay.ValidExceptionPrefix(p.Prefix) || !relay.IsReservedPrefix(p.Prefix) {
				return nil, relay.ErrConfigInvalid
			}
		}
		permissions = append(permissions, p)
	}
	return permissions, rows.Err()
}

// RemoveRelayPermissionAsRoot revokes a single manual or seeded exception.
func (s *Sqlite) RemoveRelayPermissionAsRoot(ctx context.Context, session uuid.UUID, profile int32, hostPort string, prefix netip.Prefix) error {
	host, err := normalizeRelayHostPort(hostPort)
	if err != nil {
		return db.ErrInvalidInput
	}
	cidr := ""
	if prefix.IsValid() {
		cidr = prefix.Masked().String()
	}
	return s.withIngestWriter(ctx, func(tx *sql.Conn) error {
		if err := checkIngestRoot(ctx, tx, session); err != nil {
			return err
		}
		result, err := tx.ExecContext(ctx, "DELETE FROM relay_allowlist WHERE profile_id=? AND host_port=? AND cidr=?", profile, host, cidr)
		if err != nil {
			return err
		}
		n, err := result.RowsAffected()
		if err != nil || n == 0 {
			return err
		}
		// Do not let an old claim reuse a subsequently revoked policy snapshot.
		_, err = tx.ExecContext(ctx, `UPDATE relay_destinations SET config_version=config_version+1,
			paused_until=CASE WHEN pause_class='config_invalid' THEN NULL ELSE paused_until END,
			pause_class=nullif(pause_class,'config_invalid') WHERE profile_id=?`, profile)
		return err
	})
}

// EditRelayExceptionAsRoot replaces an exception atomically and records the edit as manual.
func (s *Sqlite) EditRelayExceptionAsRoot(ctx context.Context, session uuid.UUID, profile int32, old relay.Exception, permission relay.Permission) error {
	if session == uuid.Nil {
		return db.ErrForbidden
	}
	return s.setRelayPermission(ctx, session, profile, permission, &old)
}

// RemoveRelayExceptionAsRoot revokes precisely the row selected by the operator.
func (s *Sqlite) RemoveRelayExceptionAsRoot(ctx context.Context, session uuid.UUID, profile int32, old relay.Exception) error {
	return s.withIngestWriter(ctx, func(tx *sql.Conn) error {
		if err := checkIngestRoot(ctx, tx, session); err != nil {
			return err
		}
		result, err := tx.ExecContext(ctx, "DELETE FROM relay_allowlist WHERE profile_id=? AND host_port=? AND cidr=? AND origin=?", profile, old.HostPort, old.CIDR, old.Origin)
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
		_, err = tx.ExecContext(ctx, "UPDATE relay_destinations SET config_version=config_version+1 WHERE profile_id=?", profile)
		return err
	})
}
