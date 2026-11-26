package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"slices"

	"github.com/ebrgr/TuneTrail/internal/enrichment"
)

// EnrichmentProvider is one stored registry row: the owner's choices (Enabled, Priority) next to
// what the running process reports (Available) and what the code declares (chains, eligibility).
type EnrichmentProvider struct {
	ID          string
	Enabled     bool
	Available   bool
	Priority    int
	Identity    bool
	Metadata    bool
	Eligibility enrichment.Eligibility
}

// enrichmentProviderState pairs a descriptor with the process's availability for it.
type enrichmentProviderState struct {
	descriptor enrichment.Descriptor
	available  bool
}

// ConfigureEnrichment keeps the ISSUE-174 call: whether a Subsonic server is configured. The
// ListenBrainz provider is always attached, and each receipt still needs an approved relay.
func (s *Sqlite) ConfigureEnrichment(ctx context.Context, subsonicConfigured bool) error {
	return s.configureEnrichment(ctx, []enrichmentProviderState{
		{enrichment.SubsonicDescriptor, subsonicConfigured},
		{enrichment.ListenBrainzDescriptor, true},
	})
}

// ConfigureEnrichmentProviders reconciles the stored registry with the providers of this process.
// Missing rows are inserted with the default order, which is how a new source appears without a
// migration; the owner's enabled flag and priority are kept for rows that already exist.
func (s *Sqlite) ConfigureEnrichmentProviders(ctx context.Context, registry *enrichment.Registry) error {
	var states []enrichmentProviderState
	for _, descriptor := range registry.Descriptors() {
		states = append(states, enrichmentProviderState{descriptor, registry.Available(descriptor.ID)})
	}
	return s.configureEnrichment(ctx, states)
}

func (s *Sqlite) configureEnrichment(ctx context.Context, states []enrichmentProviderState) error {
	err := s.withIngestWriter(ctx, func(tx *sql.Conn) error {
		// Availability is a statement about this process, so it starts from none on every start.
		if _, err := tx.ExecContext(ctx, `UPDATE enrichment_providers SET available=0`); err != nil {
			return err
		}
		for _, state := range states {
			d := state.descriptor
			if _, err := tx.ExecContext(ctx, `INSERT INTO enrichment_providers(id,enabled,available,priority,identity,metadata,eligibility)
				VALUES(?,1,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET available=excluded.available,
				identity=excluded.identity,metadata=excluded.metadata,eligibility=excluded.eligibility`,
				d.ID, state.available, d.Priority, d.In(enrichment.ChainIdentity), d.In(enrichment.ChainMetadata), string(d.Eligibility)); err != nil {
				return err
			}
		}
		// Queued work with no source left to ask is closed instead of waiting forever. A receipt
		// whose profile has an approved relay still has the profile-relay source.
		_, err := tx.ExecContext(ctx, `UPDATE enrichment_jobs SET state='disabled',lease=NULL,lease_until=NULL
			WHERE state IN ('pending','working')
			AND NOT EXISTS (SELECT 1 FROM enrichment_providers WHERE enabled=1 AND available=1 AND eligibility='configured')
			AND NOT EXISTS (SELECT 1 FROM enrichment_providers WHERE enabled=1 AND available=1 AND eligibility='profile-relay'
				AND EXISTS (SELECT 1 `+listenBrainzRelay+` AND e.receipt_id=enrichment_jobs.receipt_id))`, enrichment.ListenBrainzBase)
		return err
	})
	// Admission only uses the enrichment tables after engine startup has configured them.
	if err == nil {
		s.enrichmentReady.Store(true)
	}
	return err
}

// EnrichmentProviders lists the registry in execution order.
func (s *Sqlite) EnrichmentProviders(ctx context.Context) ([]EnrichmentProvider, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,enabled,available,priority,identity,metadata,eligibility
		FROM enrichment_providers ORDER BY priority,id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []EnrichmentProvider
	for rows.Next() {
		var p EnrichmentProvider
		var eligibility string
		if err := rows.Scan(&p.ID, &p.Enabled, &p.Available, &p.Priority, &p.Identity, &p.Metadata, &eligibility); err != nil {
			return nil, err
		}
		p.Eligibility = enrichment.Eligibility(eligibility)
		out = append(out, p)
	}
	return out, rows.Err()
}

// SetEnrichmentProvider turns a registered source on or off and moves it in the order. Lower
// priorities run first. Work already queued follows the new choice at its next step.
func (s *Sqlite) SetEnrichmentProvider(ctx context.Context, id string, enabled bool, priority int) error {
	if priority < 0 {
		return fmt.Errorf("enrichment priority must not be negative")
	}
	return s.withIngestWriter(ctx, func(tx *sql.Conn) error {
		res, err := tx.ExecContext(ctx, `UPDATE enrichment_providers SET enabled=?,priority=? WHERE id=?`, enabled, priority, id)
		if err != nil {
			return err
		}
		if n, err := res.RowsAffected(); err != nil {
			return err
		} else if n == 0 {
			return fmt.Errorf("unknown enrichment provider %q", id)
		}
		return nil
	})
}

// enrichmentOrder lists the descriptors of one chain that are enabled, available in this process
// and present in the registry, in the stored order. Reading it per receipt lets a change of order
// or of the enabled flag apply without a restart.
func (s *Sqlite) enrichmentOrder(ctx context.Context, registry *enrichment.Registry, chain enrichment.Chain) ([]enrichment.Descriptor, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM enrichment_providers WHERE enabled=1 AND available=1
		AND ((?1='identity' AND identity=1) OR (?1='metadata' AND metadata=1)) ORDER BY priority,id`, string(chain))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var order []enrichment.Descriptor
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		if _, ok := registry.Provider(id); !ok {
			continue
		}
		if d, ok := registry.Descriptor(id); ok && d.In(chain) {
			order = append(order, d)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return slices.Clip(order), nil
}
