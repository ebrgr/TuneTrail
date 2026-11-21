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
	var raw string
	var version int64
	err := s.db.QueryRowContext(ctx, `SELECT r.url,d.config_version FROM profile_relays r
		JOIN relay_destinations d USING(profile_id) WHERE r.profile_id=?`, profile).Scan(&raw, &version)
	if errors.Is(err, sql.ErrNoRows) {
		return relay.Grant{}, db.ErrNotFound
	}
	if err != nil {
		return relay.Grant{}, err
	}
	policy, err := s.RelayNetworkPolicyForProfile(ctx, profile)
	if err != nil {
		return relay.Grant{}, err
	}
	dns, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	grant, err := planRelayGrant(dns, net.DefaultResolver, raw, version, policy)
	grant.Profile, grant.Version = profile, version
	sum := sha256.Sum256([]byte(fmt.Sprintf("%d:%s", profile, grant.Fence)))
	grant.Fence = hex.EncodeToString(sum[:])
	return grant, err
}

// ConfirmRelayGrant resolves again and fences the entire grant under the writer reservation.
func (s *Sqlite) ConfirmRelayGrant(ctx context.Context, session uuid.UUID, profile int32, fence string) error {
	grant, err := s.RelayGrant(ctx, session, profile)
	if err != nil {
		return err
	}
	if grant.Fence != fence || !grant.Available() {
		return db.ErrConflict
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
		_, err = tx.ExecContext(ctx, `UPDATE relay_destinations SET config_version=config_version+1,
			pause_class=NULL,paused_until=NULL,backoff_count=0 WHERE profile_id=?`, profile)
		return err
	})
	if err == nil {
		logger.FromContext(ctx).Info().Int32("profile_id", profile).Str("destination_id", fmt.Sprintf("relay:%d", profile)).
			Str("host_port", grant.Authority).Bool("allow_http", grant.HTTP).Bool("allow_private", len(grant.Private) > 0).
			Msg("Relay exception confirmed")
	}
	return err
}
