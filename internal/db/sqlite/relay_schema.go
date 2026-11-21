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
	"time"

	migrations "github.com/ebrgr/TuneTrail/db/migrations_sqlite"
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
	host := strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
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
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	if err != nil || portErr != nil || n == 0 || host == "" || strings.ContainsAny(host, " /?#@\t\r\n") {
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
	blocked                  bool
}

// relaySeeds resolves endpoints before obtaining the writer reservation.
func relaySeeds(ctx context.Context, db *sql.DB, resolver relayResolver) ([]relaySeed, error) {
	rows, err := db.QueryContext(ctx, "SELECT profile_id,url FROM profile_relays")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var seeds []relaySeed
	for rows.Next() {
		var seed relaySeed
		if err := rows.Scan(&seed.profile, &seed.raw); err != nil {
			return nil, err
		}
		seeds = append(seeds, seed)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// A single deadline bounds the total DNS wait, even with many configured profiles.
	dnsCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	for i := range seeds {
		seeds[i].resolve(dnsCtx, resolver)
	}
	return seeds, nil
}

// resolve records only the private IPs observed at upgrade and rejects mixed forbidden answers.
func (seed *relaySeed) resolve(ctx context.Context, resolver relayResolver) {
	seed.canonical, seed.hostPort, _ = canonicalRelay(seed.raw) // An invalid URL remains blocked.
	seed.blocked = true
	if seed.hostPort == "" {
		return
	}
	host, _, err := net.SplitHostPort(seed.hostPort)
	if err != nil {
		return
	}
	ip, parseErr := netip.ParseAddr(host)
	addrs := []netip.Addr{ip}
	if parseErr != nil {
		addrs, err = resolver.LookupNetIP(ctx, "ip", host)
		if err != nil || len(addrs) == 0 {
			return
		}
	}
	for _, ip := range addrs {
		ip = ip.Unmap().WithZone("")
		if relay.IsForbiddenAddress(ip) {
			return
		}
		if relay.IsReservedAddress(ip) {
			seed.private = append(seed.private, netip.PrefixFrom(ip, ip.BitLen()))
		}
	}
	seed.http = strings.HasPrefix(seed.canonical, "http://")
	seed.blocked = false
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
		if _, err := tx.ExecContext(ctx, "UPDATE relay_destinations SET pause_class=NULL WHERE profile_id=?", seed.profile); err != nil {
			return err
		}
		if seed.http {
			if _, err := tx.ExecContext(ctx, "INSERT INTO relay_allowlist VALUES(?,?,'',1,0,'seeded')", seed.profile, seed.hostPort); err != nil {
				return err
			}
		}
		for _, prefix := range seed.private {
			if _, err := tx.ExecContext(ctx, "INSERT INTO relay_allowlist VALUES(?,?,?,0,1,'seeded') ON CONFLICT DO NOTHING", seed.profile, seed.hostPort, prefix.String()); err != nil {
				return err
			}
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
	return tx.Commit()
}

// migrateDatabase applies historical Goose migrations and the atomic relay upgrade.
func migrateDatabase(db *sql.DB) error {
	if err := goose.UpTo(db, ".", 11); err != nil {
		return err
	}
	return upgradeRelaySchema(context.Background(), db, net.DefaultResolver)
}

// checkRelayWriterPolicy keeps intermediate binaries from serving an activated database.
func checkRelayWriterPolicy(db *sql.DB) error {
	var policy string
	if err := db.QueryRow("SELECT policy FROM relay_delivery_gate WHERE id=1").Scan(&policy); err != nil {
		return fmt.Errorf("read receipt delivery policy: %w", err)
	}
	if policy != "legacy" {
		return errors.New("database requires the durable relay writer")
	}
	return nil
}
