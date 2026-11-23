package sqlite

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"strings"
	"time"

	"github.com/ebrgr/TuneTrail/internal/cfg"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/relay"
	"github.com/google/uuid"
)

// RelayPolicyProfiles reads the root's relay panel without selecting any credentials.
func (s *Sqlite) RelayPolicyProfiles(ctx context.Context, session uuid.UUID, profile int32) ([]relay.ProfileView, error) {
	if err := checkIngestRoot(ctx, s.db, session); err != nil {
		return nil, err
	}
	version, err := ingestSchemaVersion(ctx, s.db)
	if err != nil {
		return nil, err
	}
	draftColumn := "0"
	if version >= 21 {
		draftColumn = "coalesce(d.awaiting_approval,0)"
	}
	rows, err := s.db.QueryContext(ctx, `SELECT p.id,p.display_name,coalesce(r.url,''),coalesce(d.pause_class,''),`+draftColumn+`,
		d.pause_class IS NOT NULL AND (d.paused_until IS NULL OR d.paused_until>unixepoch()),
		coalesce(a.host_port,''),coalesce(a.cidr,''),coalesce(a.origin,''),coalesce(a.allow_http,0),coalesce(a.allow_private,0)
		FROM listening_profiles p LEFT JOIN profile_relays r ON r.profile_id=p.id
		LEFT JOIN relay_destinations d ON d.profile_id=p.id LEFT JOIN relay_allowlist a ON a.profile_id=p.id
		WHERE (?=0 OR p.id=?) ORDER BY p.id,a.host_port,a.cidr,a.origin`, profile, profile)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var profiles []relay.ProfileView
	var endpoints []string
	for rows.Next() {
		var p relay.ProfileView
		var e relay.Exception
		var endpoint, reason string
		if err := rows.Scan(&p.ID, &p.Name, &endpoint, &reason, &p.Draft, &p.Paused, &e.HostPort, &e.CIDR, &e.Origin, &e.HTTP, &e.Private); err != nil {
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
			endpoints = append(endpoints, endpoint)
		}
		if e.Origin != "" {
			profiles[len(profiles)-1].Exceptions = append(profiles[len(profiles)-1].Exceptions, e)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	dns, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	httpAllowed, privateAllowed := cfg.RelayNetworkPolicy()
	for i, endpoint := range endpoints {
		p := &profiles[i]
		if !p.Configured {
			continue
		}
		if p.Draft {
			// Rendering a saved draft must not trigger DNS or contact the destination.
			if ip, err := netip.ParseAddr(p.Diagnostic.Host); err != nil {
				continue
			} else if relay.IsForbiddenAddress(ip) {
				p.Diagnostic.Code = "forbidden_range"
				continue
			}
		}
		switch p.Diagnostic.Code {
		case "metadata_endpoint", "seed_skipped_resolves_forbidden":
			continue
		case "private_network_not_allowed", "http_not_allowed", "forbidden_range", "seed_skipped_non_admin":
		default:
			if _, err := netip.ParseAddr(p.Diagnostic.Host); err != nil && p.Diagnostic.Scheme != "http" {
				continue
			}
		}
		policy := relay.NetworkPolicy{AllowHTTP: httpAllowed, AllowPrivate: privateAllowed}
		for _, exception := range p.Exceptions {
			prefix, _ := netip.ParsePrefix(exception.CIDR)
			policy.Permissions = append(policy.Permissions, relay.Permission{HostPort: exception.HostPort,
				Prefix: prefix, HTTP: exception.HTTP, Private: exception.Private})
		}
		grant, err := planRelayGrant(dns, net.DefaultResolver, endpoint, 0, policy)
		if errors.Is(err, db.ErrForbidden) {
			p.Diagnostic.Code = "forbidden_range"
			p.Paused = true
		}
		if err == nil && grant.Available() {
			p.Grant.HTTP, p.Grant.Private = grant.HTTP, grant.Private
			p.Paused = true
		}
	}
	return profiles, nil
}
