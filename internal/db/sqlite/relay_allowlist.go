package sqlite

import (
	"context"
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
		if !relay.ValidExceptionPrefix(prefix) {
			return errors.New("invalid relay exception prefix")
		}
		cidr = prefix.String()
	}
	if (!permission.HTTP && !permission.Private) || (permission.Private && cidr == "") {
		return errors.New("invalid relay permissions")
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO relay_allowlist VALUES(?,?,?,?,?,'manual')
		ON CONFLICT(profile_id,host_port,cidr,origin) DO UPDATE SET allow_http=excluded.allow_http,allow_private=excluded.allow_private`,
		profile, host, cidr, permission.HTTP, permission.Private)
	return err
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
			if err != nil || !relay.ValidExceptionPrefix(p.Prefix) {
				return nil, errors.New("invalid stored relay exception")
			}
		}
		permissions = append(permissions, p)
	}
	return permissions, rows.Err()
}
