package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"slices"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/logger"
	"github.com/ebrgr/TuneTrail/internal/relay"
	"github.com/google/uuid"
)

// planRelayGrant derives the smallest permissions supported by the existing allowlist.
func planRelayGrant(ctx context.Context, resolver relayResolver, raw string, version int64, policy relay.NetworkPolicy) (relay.Grant, error) {
	endpoint, key, err := canonicalRelay(raw)
	if err != nil {
		return relay.Grant{}, db.ErrInvalidInput
	}
	info := relay.Describe(endpoint, "", "")
	host, _, _ := net.SplitHostPort(key)
	literal, err := netip.ParseAddr(host)
	addresses := []netip.Addr{literal}
	named := err != nil
	if named {
		addresses, err = resolver.LookupNetIP(ctx, "ip", host)
		if err != nil || len(addresses) == 0 {
			return relay.Grant{}, db.ErrConflict
		}
	}
	for i, ip := range addresses {
		addresses[i] = ip.Unmap()
	}
	slices.SortFunc(addresses, netip.Addr.Compare)
	addresses = slices.Compact(addresses)
	grant := relay.Grant{Authority: key, Scheme: info.Scheme, Named: named}
	for _, ip := range addresses {
		if !ip.IsValid() || ip.Zone() != "" || relay.IsForbiddenAddress(ip) {
			return relay.Grant{}, db.ErrForbidden
		}
		grant.Addresses = append(grant.Addresses, ip.String())
		privateMissing := !policy.Allows("https", key, ip)
		if privateMissing {
			prefix := netip.PrefixFrom(ip, ip.BitLen())
			if !relay.ValidExceptionPrefix(prefix) || !relay.IsReservedPrefix(prefix) {
				return relay.Grant{}, db.ErrForbidden
			}
			grant.Private = append(grant.Private, prefix.String())
			grant.Local = grant.Local || ip.IsLoopback() || ip.IsLinkLocalUnicast()
		}
		// Test cleartext with the missing address permission supplied locally.
		httpPolicy := policy
		if privateMissing {
			httpPolicy.Permissions = slices.Concat(policy.Permissions, relay.Permissions{
				{HostPort: key, Private: true, Prefix: netip.PrefixFrom(ip, ip.BitLen())},
			})
		}
		grant.HTTP = grant.HTTP || !httpPolicy.Allows(info.Scheme, key, ip)
	}
	sum := sha256.Sum256([]byte(fmt.Sprintf("%q %d %q %t %q", raw, version, grant.Addresses, grant.HTTP, grant.Private)))
	grant.Fence = hex.EncodeToString(sum[:])
	return grant, nil
}

// RelayGrant checks a saved destination for a confirmation without contacting its HTTP service.
func (s *Sqlite) RelayGrant(ctx context.Context, session uuid.UUID, profile int32) (relay.Grant, error) {
	if err := checkIngestRoot(ctx, s.db, session); err != nil {
		return relay.Grant{}, err
	}
	draft, err := s.RelayAwaitingApproval(ctx, profile)
	if err != nil {
		return relay.Grant{}, err
	}
	raw, version, err := s.relayGrantEndpoint(ctx, profile)
	if err != nil {
		return relay.Grant{}, err
	}
	if draft {
		// Named drafts are resolved only after a root confirms network approval.
		_, authority, err := canonicalRelay(raw)
		if err != nil {
			return relay.Grant{}, db.ErrInvalidInput
		}
		host, _, _ := net.SplitHostPort(authority)
		if _, err := netip.ParseAddr(host); err != nil {
			sum := sha256.Sum256([]byte(fmt.Sprintf("named:%d:%d:%q", profile, version, raw)))
			return relay.Grant{Profile: profile, Version: version, Draft: true, Named: true,
				Authority: authority, Scheme: relay.Describe(raw, "", "").Scheme,
				Fence: hex.EncodeToString(sum[:])}, nil
		}
	}
	policy, err := s.RelayNetworkPolicyForProfile(ctx, profile)
	if err != nil {
		return relay.Grant{}, err
	}
	dns, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	grant, err := planRelayGrant(dns, net.DefaultResolver, raw, version, policy)
	grant.Draft = draft
	grant.Profile, grant.Version = profile, version
	sum := sha256.Sum256([]byte(fmt.Sprintf("%d:%s", profile, grant.Fence)))
	grant.Fence = hex.EncodeToString(sum[:])
	return grant, err
}

// relayGrantEndpoint reads the endpoint and configuration fence without selecting its token.
func (s *Sqlite) relayGrantEndpoint(ctx context.Context, profile int32) (string, int64, error) {
	var raw string
	var version int64
	err := s.db.QueryRowContext(ctx, `SELECT r.url,d.config_version FROM profile_relays r
		JOIN relay_destinations d USING(profile_id) WHERE r.profile_id=?`, profile).Scan(&raw, &version)
	if errors.Is(err, sql.ErrNoRows) {
		err = db.ErrNotFound
	}
	return raw, version, err
}

// ConfirmRelayGrant resolves again and fences the entire grant under the writer reservation.
func (s *Sqlite) ConfirmRelayGrant(ctx context.Context, session uuid.UUID, profile int32, fence string) error {
	grant, err := s.RelayGrant(ctx, session, profile)
	if err != nil {
		return err
	}
	if grant.Fence != fence || (!grant.Available() && !grant.Draft) {
		return db.ErrConflict
	}
	if grant.Draft && grant.Named {
		raw, version, err := s.relayGrantEndpoint(ctx, profile)
		if err != nil {
			return err
		}
		if version != grant.Version {
			return db.ErrConflict
		}
		policy, err := s.RelayNetworkPolicyForProfile(ctx, profile)
		if err != nil {
			return err
		}
		dns, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		resolved, err := planRelayGrant(dns, net.DefaultResolver, raw, version, policy)
		if err != nil {
			return err
		}
		// Never add an exception for a DNS answer the approver has not seen.
		if resolved.Available() {
			return db.ErrForbidden
		}
	}
	err = s.withIngestWriter(ctx, func(tx *sql.Conn) error {
		if err := checkIngestRoot(ctx, tx, session); err != nil {
			return err
		}
		version, err := relayConfigurationVersion(ctx, tx, profile)
		if err != nil {
			return err
		}
		if version != grant.Version {
			return db.ErrConflict
		}
		if grant.HTTP {
			if _, err := tx.ExecContext(ctx, `INSERT INTO relay_allowlist VALUES(?,?,'',1,0,'manual') ON CONFLICT DO NOTHING`,
				profile, grant.Authority); err != nil {
				return err
			}
		}
		for _, prefix := range grant.Private {
			if _, err := tx.ExecContext(ctx, `INSERT INTO relay_allowlist VALUES(?,?,?,0,1,'manual') ON CONFLICT DO NOTHING`,
				profile, grant.Authority, prefix); err != nil {
				return err
			}
		}
		query := `UPDATE relay_destinations SET config_version=config_version+1,
			pause_class=NULL,paused_until=NULL,backoff_count=0 WHERE profile_id=?`
		if grant.Draft {
			query = `UPDATE relay_destinations SET config_version=config_version+1,awaiting_approval=0,
				pause_class=NULL,paused_until=NULL,backoff_count=0 WHERE profile_id=?`
		}
		_, err = tx.ExecContext(ctx, query, profile)
		return err
	})
	if err == nil {
		logger.FromContext(ctx).Info().Int32("profile_id", profile).Str("destination_id", fmt.Sprintf("relay:%d", profile)).
			Str("host_port", grant.Authority).Bool("allow_http", grant.HTTP).Bool("allow_private", len(grant.Private) > 0).
			Msg("Relay exception confirmed")
	}
	return err
}

// RelayAwaitingApproval reads the draft state without resolving or contacting the destination.
func (s *Sqlite) RelayAwaitingApproval(ctx context.Context, profile int32) (bool, error) {
	var draft bool
	err := s.db.QueryRowContext(ctx, `SELECT coalesce((SELECT awaiting_approval FROM relay_destinations
		JOIN profile_relays USING(profile_id) WHERE profile_id=?),0)`, profile).Scan(&draft)
	return draft, err
}

// RelayTestApproved checks the exact approved configuration before an explicit probe.
func (s *Sqlite) RelayTestApproved(ctx context.Context, profile int32, url, token string) (bool, error) {
	var approved bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM profile_relays r
		JOIN relay_destinations d USING(profile_id) WHERE r.profile_id=? AND r.url=? AND r.token=? AND d.awaiting_approval=0)`,
		profile, url, token).Scan(&approved)
	return approved, err
}
