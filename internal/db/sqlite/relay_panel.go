package sqlite

import (
	"context"
	"strings"

	"github.com/ebrgr/TuneTrail/internal/relay"
	"github.com/google/uuid"
)

// RelayPolicyProfiles reads the root's relay panel without selecting any credentials.
func (s *Sqlite) RelayPolicyProfiles(ctx context.Context, session uuid.UUID) ([]relay.ProfileView, error) {
	if err := checkIngestRoot(ctx, s.db, session); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT p.id,p.display_name,coalesce(r.url,''),coalesce(d.pause_class,''),
		d.pause_class IS NOT NULL AND (d.paused_until IS NULL OR d.paused_until>unixepoch()),
		coalesce(a.host_port,''),coalesce(a.cidr,''),coalesce(a.origin,''),coalesce(a.allow_http,0),coalesce(a.allow_private,0)
		FROM listening_profiles p LEFT JOIN profile_relays r ON r.profile_id=p.id
		LEFT JOIN relay_destinations d ON d.profile_id=p.id LEFT JOIN relay_allowlist a ON a.profile_id=p.id
		ORDER BY p.id,a.host_port,a.cidr,a.origin`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var profiles []relay.ProfileView
	for rows.Next() {
		var p relay.ProfileView
		var e relay.Exception
		var endpoint, reason string
		if err := rows.Scan(&p.ID, &p.Name, &endpoint, &reason, &p.Paused, &e.HostPort, &e.CIDR, &e.Origin, &e.HTTP, &e.Private); err != nil {
			return nil, err
		}
		if len(profiles) == 0 || profiles[len(profiles)-1].ID != p.ID {
			parts := strings.SplitN(reason, ":", 3)
			code, category := parts[0], ""
			if len(parts) > 1 {
				code = parts[1]
			}
			if len(parts) > 2 {
				category = parts[2]
			}
			p.Configured = endpoint != ""
			p.Diagnostic = relay.Describe(endpoint, code, category)
			profiles = append(profiles, p)
		}
		if e.Origin != "" {
			profiles[len(profiles)-1].Exceptions = append(profiles[len(profiles)-1].Exceptions, e)
		}
	}
	return profiles, rows.Err()
}
