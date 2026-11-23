package sqlite

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	migrations "github.com/ebrgr/TuneTrail/db/migrations_sqlite"
	"github.com/ebrgr/TuneTrail/internal/cfg"
	"github.com/ebrgr/TuneTrail/internal/relay"
	"github.com/pressly/goose/v3"
	sqlite3 "modernc.org/sqlite"
)

const relayMigration = "012_relay_delivery.sql"

func init() {
	// SQL writes, including legacy bootstrap, compare endpoint identities consistently.
	sqlite3.MustRegisterDeterministicScalarFunction("tt_relay_canonical", 1,
		func(_ *sqlite3.FunctionContext, values []driver.Value) (driver.Value, error) {
			raw, _ := values[0].(string)
			canonical, _, err := canonicalRelay(raw)
			if err != nil {
				return raw, nil
			}
			return canonical, nil
		})
}

// canonicalRelay normalizes the endpoint while preserving the case of its path.
func canonicalRelay(raw string) (string, string, error) {
	raw = strings.TrimSpace(raw)
	scheme, rest, ok := strings.Cut(raw, "://")
	if !ok {
		return "", "", errors.New("invalid relay endpoint")
	}
	normalized, _, err := relay.Normalize(strings.ToLower(scheme)+"://"+rest, "placeholder")
	if err != nil {
		return "", "", err
	}
	u, err := url.Parse(normalized)
	if err != nil {
		return "", "", err
	}
	host, err := relay.CanonicalHost(u.Hostname())
	if err != nil {
		return "", "", err
	}
	port := u.Port()
	defaultPort := "443"
	if u.Scheme == "http" {
		defaultPort = "80"
	}
	if port == "" {
		port = defaultPort
	}
	key, err := normalizeRelayHostPort(net.JoinHostPort(host, port))
	if err != nil {
		return "", "", err
	}
	u.Host = key
	if strings.HasSuffix(key, ":"+defaultPort) {
		u.Host = host
		if strings.Contains(host, ":") {
			u.Host = "[" + host + "]"
		}
	}
	return u.String(), key, nil
}

// normalizeRelayHostPort accepts only a concrete host and numeric port.
func normalizeRelayHostPort(raw string) (string, error) {
	host, port, err := net.SplitHostPort(raw)
	n, portErr := strconv.ParseUint(port, 10, 16)
	host, hostErr := relay.CanonicalHost(host)
	if err != nil || portErr != nil || n == 0 || hostErr != nil {
		return "", errors.New("invalid relay host and port")
	}
	return net.JoinHostPort(host, strconv.FormatUint(n, 10)), nil
}

type relayResolver interface {
	LookupNetIP(context.Context, string, string) ([]netip.Addr, error)
}

type relaySeed struct {
	profile                  int32
	raw, canonical, hostPort string
	http                     bool
	private                  []netip.Prefix
	blocked, admin           bool
	reason, category         string
}

// relaySeeds resolves endpoints before obtaining the writer reservation.
func relaySeeds(ctx context.Context, db *sql.DB, resolver relayResolver) ([]relaySeed, error) {
	rows, err := db.QueryContext(ctx, "SELECT r.profile_id,r.url,u.role='admin' FROM profile_relays r JOIN listening_profiles p ON p.id=r.profile_id JOIN users u ON u.id=p.user_id ORDER BY r.profile_id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var seeds []relaySeed
	for rows.Next() {
		var seed relaySeed
		if err := rows.Scan(&seed.profile, &seed.raw, &seed.admin); err != nil {
			return nil, err
		}
		seeds = append(seeds, seed)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	resolveRelaySeeds(ctx, seeds, resolver)
	return seeds, nil
}

// resolveRelaySeeds deduplicates DNS work and bounds both concurrency and latency.
func resolveRelaySeeds(ctx context.Context, seeds []relaySeed, resolver relayResolver) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	type lookup struct {
		host string
		ips  []netip.Addr
	}
	var lookups []lookup
	hostIndex := make(map[string]int)
	seedIndex := make([]int, len(seeds))
	for i := range seeds {
		seedIndex[i] = -1
		seed := &seeds[i]
		seed.blocked = true
		seed.reason = "url_invalid"
		canonical, key, err := canonicalRelay(seed.raw)
		if err != nil {
			continue
		}
		host, _, err := net.SplitHostPort(key)
		if err != nil {
			continue
		}
		seed.canonical, seed.hostPort = canonical, key
		seed.http = strings.HasPrefix(canonical, "http://")
		seed.reason = "dns_resolution_failed"
		index, ok := hostIndex[host]
		if !ok {
			index = len(lookups)
			hostIndex[host] = index
			lookups = append(lookups, lookup{host: host})
		}
		seedIndex[i] = index
	}
	work := make(chan int, len(lookups))
	for i := range lookups {
		work <- i
	}
	close(work)
	var workers sync.WaitGroup
	for range min(4, len(lookups)) {
		workers.Go(func() {
			for i := range work {
				if ctx.Err() != nil {
					return
				}
				entry := &lookups[i]
				if ip, err := netip.ParseAddr(entry.host); err == nil {
					entry.ips = []netip.Addr{ip}
					continue
				}
				hostCtx, stop := context.WithTimeout(ctx, 2*time.Second)
				ips, err := resolver.LookupNetIP(hostCtx, "ip", entry.host)
				stop()
				if err == nil {
					entry.ips = ips
				}
			}
		})
	}
	workers.Wait()
	for i, index := range seedIndex {
		if index >= 0 {
			seeds[i].applyAddresses(lookups[index].ips)
		}
	}
}

// applyAddresses permits narrow admin exceptions and rejects an entire unsafe answer.
func (seed *relaySeed) applyAddresses(addresses []netip.Addr) {
	if len(addresses) == 0 {
		return
	}
	for _, ip := range addresses {
		ip = ip.Unmap().WithZone("")
		if relay.IsForbiddenAddress(ip) {
			seed.reason = "seed_skipped_resolves_forbidden"
			seed.category = "forbidden"
			return
		}
		if relay.IsReservedAddress(ip) {
			prefix := netip.PrefixFrom(ip, ip.BitLen())
			if !relay.ValidExceptionPrefix(prefix) {
				seed.reason = "seed_skipped_resolves_forbidden"
				seed.category = "reserved"
				return
			}
			if !seed.admin {
				seed.reason = "seed_skipped_non_admin"
				seed.category = "reserved"
				return
			}
			seed.private = append(seed.private, prefix)
		}
	}
	if seed.http && !seed.admin {
		seed.reason = "seed_skipped_non_admin"
		return
	}
	seed.blocked = false
	seed.reason = "seed_not_required"
	if seed.http || len(seed.private) > 0 {
		seed.reason = "seeded"
	}
}

// applyRelaySeed records a policy block or narrowly scoped operator exceptions.
func applyRelaySeed(ctx context.Context, tx *sql.Tx, seed relaySeed) error {
	var pause any
	var deadline any
	if seed.blocked {
		pause = "policy_blocked:" + seed.reason + ":" + seed.category
		if seed.reason == "dns_resolution_failed" {
			// DNS at startup may be temporarily unavailable. The dialer checks each retry.
			pause = "transient:dns_resolution_failed:"
			deadline = time.Now().Add(30 * time.Second).Unix()
		}
	}
	if _, err := tx.ExecContext(ctx, "UPDATE relay_destinations SET pause_class=?,paused_until=? WHERE profile_id=?", pause, deadline, seed.profile); err != nil {
		return err
	}
	if seed.blocked {
		return nil
	}
	if seed.http {
		if _, err := tx.ExecContext(ctx, "INSERT INTO relay_allowlist VALUES(?,?,'',1,0,'seeded') ON CONFLICT DO NOTHING", seed.profile, seed.hostPort); err != nil {
			return err
		}
	}
	for _, prefix := range seed.private {
		if _, err := tx.ExecContext(ctx, "INSERT INTO relay_allowlist VALUES(?,?,?,0,1,'seeded') ON CONFLICT DO NOTHING", seed.profile, seed.hostPort, prefix.String()); err != nil {
			return err
		}
	}
	return nil
}

// upgradeRelaySchema commits the final SQL, seeds and version marker atomically.
func upgradeRelaySchema(ctx context.Context, db *sql.DB, resolver relayResolver) error {
	version, err := schemaVersion(db)
	if err != nil || version >= 12 {
		return err
	}
	if version != 11 {
		return errors.New("relay upgrade requires schema v11")
	}
	seeds, err := relaySeeds(ctx, db, resolver)
	if err != nil {
		return err
	}
	ddl, err := migrations.Files.ReadFile(relayMigration)
	if err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := bootstrapPhase("relay-before-ddl"); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, strings.SplitN(string(ddl), "-- +goose Down", 2)[0]); err != nil {
		return fmt.Errorf("apply relay schema: %w", err)
	}
	if err := bootstrapPhase("relay-after-ddl"); err != nil {
		return err
	}
	for _, seed := range seeds {
		if seed.blocked {
			var unchanged bool
			err := tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM profile_relays WHERE profile_id=? AND url=?)", seed.profile, seed.raw).Scan(&unchanged)
			if err != nil {
				return err
			}
			if unchanged {
				if err := applyRelaySeed(ctx, tx, seed); err != nil {
					return err
				}
			}
			continue
		}
		updated, err := tx.ExecContext(ctx, "UPDATE profile_relays SET url=? WHERE profile_id=? AND url=?", seed.canonical, seed.profile, seed.raw)
		if err != nil {
			return err
		}
		n, err := updated.RowsAffected()
		if err != nil {
			return err
		}
		if n != 1 {
			continue
		}
		if err := applyRelaySeed(ctx, tx, seed); err != nil {
			return err
		}
	}
	if err := bootstrapPhase("relay-after-seed"); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO goose_db_version(version_id,is_applied) VALUES(12,1)"); err != nil {
		return err
	}
	if err := bootstrapPhase("relay-before-commit"); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	for _, seed := range seeds {
		relay.LogDiagnostic(ctx, seed.profile, relay.Describe(seed.raw, seed.reason, seed.category), time.Now())
	}
	return nil
}

// migrateDatabaseToVersion retains the relay seed transaction for historical fixtures.
func migrateDatabaseToVersion(db *sql.DB, targetVersion int) error {
	// Preserve the custom relay seed transaction before applying later SQL migrations.
	version, err := goose.EnsureDBVersion(db)
	if err != nil {
		return err
	}
	if version < 11 {
		if err := goose.UpTo(db, ".", 11); err != nil {
			return err
		}
	}
	if err := upgradeRelaySchema(context.Background(), db, net.DefaultResolver); err != nil {
		return err
	}
	return goose.UpTo(db, ".", int64(targetVersion))
}

// ActivateRelayDelivery switches the durable writer on before the service accepts requests.
func (s *Sqlite) ActivateRelayDelivery(ctx context.Context) error {
	httpAllowed, privateAllowed := cfg.RelayNetworkPolicy()
	return s.activateRelayDelivery(ctx, httpAllowed, privateAllowed)
}

// activateRelayDelivery rechecks permissions-related pauses when a startup policy enables them.
func (s *Sqlite) activateRelayDelivery(ctx context.Context, httpAllowed, privateAllowed bool) error {
	return s.withIngestWriter(ctx, func(tx *sql.Conn) error {
		if _, err := tx.ExecContext(ctx, "UPDATE relay_delivery_gate SET policy='durable' WHERE id=1 AND policy='legacy'"); err != nil {
			return err
		}
		// Recover databases initialized by an earlier unpublished build.
		if _, err := tx.ExecContext(ctx, "UPDATE relay_destinations SET pause_class='transient:dns_resolution_failed:',paused_until=unixepoch()+30 WHERE pause_class LIKE 'policy_blocked:dns_resolution_failed:%'"); err != nil {
			return err
		}
		if !httpAllowed && !privateAllowed {
			return nil
		}
		// Global private access does not authorize loopback, link-local, or unclassified seed ranges.
		_, err := tx.ExecContext(ctx, `UPDATE relay_destinations SET pause_class=NULL,paused_until=NULL,config_version=config_version+1
			WHERE pause_class='policy_blocked'
			OR (? AND (pause_class LIKE 'policy_blocked:http_not_allowed:%' OR pause_class='policy_blocked:seed_skipped_non_admin:'))
			OR (? AND pause_class LIKE 'policy_blocked:private_network_not_allowed:%'
			AND pause_class NOT IN ('policy_blocked:private_network_not_allowed:loopback','policy_blocked:private_network_not_allowed:link_local'))`, httpAllowed, privateAllowed)
		return err
	})
}
