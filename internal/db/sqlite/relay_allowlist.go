package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"net/netip"

	"github.com/ebrgr/TuneTrail/internal/relay"
)

// RelayPermission separates cleartext permission from access to reserved addresses.
type RelayPermission struct {
	HostPort      string
	Prefix        netip.Prefix
	HTTP, Private bool
}

// SetRelayPermission saves a validated operator exception for one profile.
func (s *Sqlite) SetRelayPermission(ctx context.Context, profile int32, permission RelayPermission) error {
	host, err := normalizeRelayHostPort(permission.HostPort)
	if err != nil {
		return err
	}
	cidr := ""
	if permission.Prefix.IsValid() {
		prefix := permission.Prefix.Masked()
		if !permission.Private || !relay.ValidExceptionPrefix(prefix) || !relay.IsReservedPrefix(prefix) {
			return errors.New("invalid relay exception prefix")
		}
		cidr = prefix.String()
	}
	if (!permission.HTTP && !permission.Private) || (permission.Private && cidr == "") {
		return errors.New("invalid relay permissions")
	}
	return s.withIngestWriter(ctx, func(tx *sql.Conn) error {
		result, err := tx.ExecContext(ctx, `INSERT INTO relay_allowlist VALUES(?,?,?,?,?,'manual')
			ON CONFLICT(profile_id,host_port,cidr,origin) DO UPDATE SET allow_http=excluded.allow_http,allow_private=excluded.allow_private
			WHERE allow_http<>excluded.allow_http OR allow_private<>excluded.allow_private`,
			profile, host, cidr, permission.HTTP, permission.Private)
		if err != nil {
			return err
		}
		n, err := result.RowsAffected()
		if err != nil || n == 0 {
			return err
		}
		// An operator policy change clears a policy block and fences old transport results.
		_, err = tx.ExecContext(ctx, `UPDATE relay_destinations SET config_version=config_version+1,
			paused_until=CASE WHEN pause_class='policy_blocked' THEN NULL ELSE paused_until END,
			pause_class=nullif(pause_class,'policy_blocked') WHERE profile_id=?`, profile)
		return err
	})
}

// RelayPermissions reads a profile-scoped permission snapshot without exposing credentials.
func (s *Sqlite) RelayPermissions(ctx context.Context, profile int32) ([]RelayPermission, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT host_port,cidr,allow_http,allow_private FROM relay_allowlist WHERE profile_id=?", profile)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var permissions []RelayPermission
	for rows.Next() {
		var p RelayPermission
		var cidr string
		if err := rows.Scan(&p.HostPort, &cidr, &p.HTTP, &p.Private); err != nil {
			return nil, err
		}
		if cidr != "" {
			p.Prefix, err = netip.ParsePrefix(cidr)
			if err != nil || !p.Private || !relay.ValidExceptionPrefix(p.Prefix) || !relay.IsReservedPrefix(p.Prefix) {
				return nil, errors.New("invalid stored relay exception")
			}
		}
		permissions = append(permissions, p)
	}
	return permissions, rows.Err()
}
