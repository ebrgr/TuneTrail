package sqlite

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/enrichment"
	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/ebrgr/TuneTrail/internal/mbz"
	"github.com/stretchr/testify/require"
)

// chainProvider is a fixture source whose behavior each test scripts.
type chainProvider struct {
	instance string
	lookup   func(context.Context, enrichment.Request) enrichment.Result
}

func (p chainProvider) Instance() string { return p.instance }
func (p chainProvider) Lookup(ctx context.Context, r enrichment.Request) enrichment.Result {
	return p.lookup(ctx, r)
}

func chainDescriptor(id string, priority int, chains ...enrichment.Chain) enrichment.Descriptor {
	return enrichment.Descriptor{ID: id, Source: id, Chains: chains, Priority: priority, Eligibility: enrichment.EligibleConfigured}
}

// recordingDeclaration answers with the fixture recording ID under the provider's own source.
func recordingDeclaration(source string) enrichment.Declaration {
	return enrichment.Declaration{Entity: "recording", Field: "musicbrainz_id", Value: fixtureRecording, Source: source, ObservedAt: time.Now()}
}

// chainReceipt admits and resolves one receipt so it can be enriched.
func chainReceipt(t *testing.T, s *Sqlite, title string, stamp int64) string {
	t.Helper()
	id := admitOccurrenceFixture(t, s, ingest.Snapshot{Title: title, Artist: title + " artist", Release: title + " album"}, stamp, title)
	resolveOccurrenceFixture(t, s, &mbz.MbzErrorCaller{})
	return id
}

// chainRuns lists "provider/chain=state" for a receipt in a stable order.
func chainRuns(t *testing.T, s *Sqlite, receipt string) []string {
	t.Helper()
	rows, err := s.db.Query(`SELECT provider||'/'||chain||'='||state FROM enrichment_provider_runs WHERE receipt_id=? ORDER BY provider,chain`, receipt)
	require.NoError(t, err)
	defer rows.Close()
	var out []string
	for rows.Next() {
		var run string
		require.NoError(t, rows.Scan(&run))
		out = append(out, run)
	}
	require.NoError(t, rows.Err())
	return out
}

func jobState(t *testing.T, s *Sqlite, receipt string) string {
	t.Helper()
	var state string
	require.NoError(t, s.db.QueryRow(`SELECT state FROM enrichment_jobs WHERE receipt_id=?`, receipt).Scan(&state))
	return state
}

// TestEnrichmentChainRunsProvidersInStoredOrder shows that the order is data: changing the stored
// priority changes which source is asked first, with no change to the code.
func TestEnrichmentChainRunsProvidersInStoredOrder(t *testing.T) {
	s := convertedOccurrenceFixture(t)
	var calls []string
	registry := enrichment.NewRegistry()
	require.NoError(t, registry.Register(chainDescriptor("source-a", 100, enrichment.ChainIdentity), chainProvider{"a-instance",
		func(context.Context, enrichment.Request) enrichment.Result {
			calls = append(calls, "a")
			return enrichment.Result{Status: enrichment.Absent}
		}}))
	require.NoError(t, registry.Register(chainDescriptor("source-b", 200, enrichment.ChainIdentity), chainProvider{"b-instance",
		func(context.Context, enrichment.Request) enrichment.Result {
			calls = append(calls, "b")
			return enrichment.Result{Status: enrichment.Found, Declarations: []enrichment.Declaration{recordingDeclaration("source-b")}}
		}}))
	require.NoError(t, s.ConfigureEnrichmentProviders(t.Context(), registry))

	first := chainReceipt(t, s, "First", 1700001000)
	found, err := s.ResolveNextEnrichmentChain(t.Context(), registry)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, []string{"a", "b"}, calls)
	require.Equal(t, []string{"source-b/identity=found"}, chainRuns(t, s, first), "only the answer that changed something is stored")
	require.Equal(t, "found", jobState(t, s, first))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM metadata_values WHERE source='source-b' AND instance='b-instance' AND value=?`, fixtureRecording))
	// The job keeps the first recorded instance, as the ISSUE-174 single-provider job did.
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM enrichment_jobs WHERE receipt_id=? AND instance='a-instance'`, first))

	calls = nil
	require.NoError(t, s.SetEnrichmentProvider(t.Context(), "source-b", true, 50))
	second := chainReceipt(t, s, "Second", 1700001100)
	found, err = s.ResolveNextEnrichmentChain(t.Context(), registry)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, []string{"b", "a"}, calls, "the lower stored priority runs first")
	require.Equal(t, []string{"source-b/identity=found"}, chainRuns(t, s, second))
}

// TestEnrichmentChainSkipsDisabledAndUnavailableProviders covers the owner's switch and the process state.
func TestEnrichmentChainSkipsDisabledAndUnavailableProviders(t *testing.T) {
	s := convertedOccurrenceFixture(t)
	var calls []string
	script := func(name string) chainProvider {
		return chainProvider{name + "-instance", func(context.Context, enrichment.Request) enrichment.Result {
			calls = append(calls, name)
			return enrichment.Result{Status: enrichment.Absent}
		}}
	}
	registry := enrichment.NewRegistry()
	require.NoError(t, registry.Register(chainDescriptor("on", 100, enrichment.ChainIdentity), script("on")))
	require.NoError(t, registry.Register(chainDescriptor("off", 110, enrichment.ChainIdentity), script("off")))
	require.NoError(t, registry.Register(chainDescriptor("missing", 120, enrichment.ChainIdentity), nil))
	require.NoError(t, s.ConfigureEnrichmentProviders(t.Context(), registry))
	require.NoError(t, s.SetEnrichmentProvider(t.Context(), "off", false, 110))

	id := chainReceipt(t, s, "Skipped", 1700001200)
	found, err := s.ResolveNextEnrichmentChain(t.Context(), registry)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, []string{"on"}, calls)
	require.Empty(t, chainRuns(t, s, id), "an answer with nothing to add leaves no row")
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM enrichment_provider_runs`))
	require.Equal(t, "absent", jobState(t, s, id))
}

// TestEnrichmentChainPassesPriorDeclarationsToLaterProviders is how the chain stops per field: a
// later source sees what an earlier one stored and leaves that field alone.
func TestEnrichmentChainPassesPriorDeclarationsToLaterProviders(t *testing.T) {
	s := convertedOccurrenceFixture(t)
	var seen []enrichment.Declaration
	registry := enrichment.NewRegistry()
	require.NoError(t, registry.Register(chainDescriptor("source-a", 100, enrichment.ChainIdentity), chainProvider{"a-instance",
		func(_ context.Context, r enrichment.Request) enrichment.Result {
			require.Empty(t, r.Prior, "nothing was stored before the first source")
			return enrichment.Result{Status: enrichment.Found, Declarations: []enrichment.Declaration{recordingDeclaration("source-a")}}
		}}))
	var laterCalledWithNetwork bool
	require.NoError(t, registry.Register(chainDescriptor("source-b", 200, enrichment.ChainIdentity), chainProvider{"b-instance",
		func(_ context.Context, r enrichment.Request) enrichment.Result {
			seen = r.Prior
			for _, prior := range r.Prior {
				if prior.Entity == "recording" {
					// The field is resolved, so this source answers without asking anyone.
					return enrichment.Result{Status: enrichment.Absent}
				}
			}
			laterCalledWithNetwork = true
			return enrichment.Result{Status: enrichment.Found, Declarations: []enrichment.Declaration{recordingDeclaration("source-b")}}
		}}))
	require.NoError(t, s.ConfigureEnrichmentProviders(t.Context(), registry))
	id := chainReceipt(t, s, "Prior", 1700001300)
	found, err := s.ResolveNextEnrichmentChain(t.Context(), registry)
	require.NoError(t, err)
	require.True(t, found)

	require.Len(t, seen, 1)
	require.Equal(t, "recording", seen[0].Entity)
	require.Equal(t, "source-a", seen[0].Source)
	require.Equal(t, fixtureRecording, seen[0].Value)
	require.False(t, laterCalledWithNetwork)
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM metadata_values WHERE recording_id IS NOT NULL`))
	require.Equal(t, []string{"source-a/identity=found"}, chainRuns(t, s, id))
}

// TestEnrichmentChainAmbiguousAnswerContinuesAndNeverWrites keeps an unclear match out of the catalog.
func TestEnrichmentChainAmbiguousAnswerContinuesAndNeverWrites(t *testing.T) {
	for _, scenario := range []struct {
		name         string
		second       enrichment.Status
		wantJob      string
		wantMetadata int
		wantRuns     []string
	}{
		{"a later source resolves it", enrichment.Found, "found", 1, []string{"clear/identity=found", "unclear/identity=ambiguous"}},
		{"nobody resolves it", enrichment.Absent, "ambiguous", 0, []string{"unclear/identity=ambiguous"}},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			s := convertedOccurrenceFixture(t)
			var calls []string
			registry := enrichment.NewRegistry()
			require.NoError(t, registry.Register(chainDescriptor("unclear", 100, enrichment.ChainIdentity), chainProvider{"u-instance",
				func(context.Context, enrichment.Request) enrichment.Result {
					calls = append(calls, "unclear")
					// Even a stray declaration from an ambiguous answer must not be stored as fact.
					return enrichment.Result{Status: enrichment.Ambiguous}
				}}))
			require.NoError(t, registry.Register(chainDescriptor("clear", 200, enrichment.ChainIdentity), chainProvider{"c-instance",
				func(context.Context, enrichment.Request) enrichment.Result {
					calls = append(calls, "clear")
					if scenario.second != enrichment.Found {
						return enrichment.Result{Status: scenario.second}
					}
					return enrichment.Result{Status: enrichment.Found, Declarations: []enrichment.Declaration{recordingDeclaration("clear")}}
				}}))
			require.NoError(t, s.ConfigureEnrichmentProviders(t.Context(), registry))
			id := chainReceipt(t, s, "Ambiguous", 1700001400)
			found, err := s.ResolveNextEnrichmentChain(t.Context(), registry)
			require.NoError(t, err)
			require.True(t, found)
			require.Equal(t, []string{"unclear", "clear"}, calls, "an ambiguous answer is not accepted and the chain goes on")
			require.Equal(t, scenario.wantJob, jobState(t, s, id))
			require.Equal(t, scenario.wantMetadata, scalar(t, s, `SELECT count(*) FROM metadata_values`))
			require.Equal(t, scenario.wantRuns, chainRuns(t, s, id))
		})
	}
}

// TestEnrichmentChainMetadataChainFollowsIdentity pins the two chains: identity always runs first
// and each provider is told which question it answers.
func TestEnrichmentChainMetadataChainFollowsIdentity(t *testing.T) {
	s := convertedOccurrenceFixture(t)
	var calls []string
	script := func(name string, status enrichment.Status, declarations ...enrichment.Declaration) chainProvider {
		return chainProvider{name + "-instance", func(_ context.Context, r enrichment.Request) enrichment.Result {
			calls = append(calls, name+":"+string(r.Chain))
			return enrichment.Result{Status: status, Declarations: declarations}
		}}
	}
	registry := enrichment.NewRegistry()
	// The metadata source has the lowest priority number, yet it runs after the identity source.
	require.NoError(t, registry.Register(chainDescriptor("genres", 10, enrichment.ChainMetadata),
		script("genres", enrichment.Found, enrichment.Declaration{Entity: "recording", Field: "genre", Value: "rock", Source: "genres", ObservedAt: time.Now()})))
	require.NoError(t, registry.Register(chainDescriptor("ids", 100, enrichment.ChainIdentity), script("ids", enrichment.Absent)))
	require.NoError(t, registry.Register(chainDescriptor("both", 200, enrichment.ChainIdentity, enrichment.ChainMetadata), script("both", enrichment.Absent)))
	require.NoError(t, s.ConfigureEnrichmentProviders(t.Context(), registry))
	id := chainReceipt(t, s, "Chains", 1700001500)
	found, err := s.ResolveNextEnrichmentChain(t.Context(), registry)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, []string{"ids:identity", "both:identity", "genres:metadata", "both:metadata"}, calls)
	require.Equal(t, []string{"genres/metadata=found"}, chainRuns(t, s, id))
	require.Equal(t, "found", jobState(t, s, id))
	// Only MusicBrainz IDs are stored by this issue; genres arrive with their providers (ISSUE-217 and 218).
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM metadata_values`))
}

// TestEnrichmentChainRecoveryKeepsFinalAnswers proves a recovered job does not repeat the calls
// whose answers were already final, and that the attempts stay capped.
func TestEnrichmentChainRecoveryKeepsFinalAnswers(t *testing.T) {
	s := convertedOccurrenceFixture(t)
	var calls []string
	interrupt := true
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	registry := enrichment.NewRegistry()
	require.NoError(t, registry.Register(chainDescriptor("source-a", 100, enrichment.ChainIdentity), chainProvider{"a-instance",
		func(context.Context, enrichment.Request) enrichment.Result {
			calls = append(calls, "a")
			return enrichment.Result{Status: enrichment.Found, Declarations: []enrichment.Declaration{recordingDeclaration("source-a")}}
		}}))
	require.NoError(t, registry.Register(chainDescriptor("source-b", 200, enrichment.ChainIdentity), chainProvider{"b-instance",
		func(context.Context, enrichment.Request) enrichment.Result {
			calls = append(calls, "b")
			if interrupt {
				cancel() // the process stops in the middle of the chain
			}
			return enrichment.Result{Status: enrichment.Failed}
		}}))
	require.NoError(t, s.ConfigureEnrichmentProviders(t.Context(), registry))
	id := chainReceipt(t, s, "Recovery", 1700001600)

	_, err := s.ResolveNextEnrichmentChain(ctx, registry)
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, []string{"a", "b"}, calls)
	require.Equal(t, "working", jobState(t, s, id), "the lease is left finite for recovery")
	require.Equal(t, []string{"source-a/identity=found"}, chainRuns(t, s, id), "the finished step was stored")

	interrupt = false
	execSQL(t, s, `UPDATE enrichment_jobs SET lease_until=0 WHERE receipt_id=?`, id)
	calls = nil
	found, err := s.ResolveNextEnrichmentChain(t.Context(), registry)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, []string{"b"}, calls, "the final answer of the first source is not asked again")
	require.Equal(t, "found", jobState(t, s, id))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM enrichment_jobs WHERE receipt_id=? AND attempts=2`, id))
	require.Equal(t, []string{"source-a/identity=found", "source-b/identity=failed"}, chainRuns(t, s, id))
}

// TestEnrichmentChainStoresOnlyAnswersThatChangeSomething separates a failed call, which is stored
// and retried, from "nothing to add", which is not stored so listens that already carry their IDs
// cost no extra bytes.
func TestEnrichmentChainStoresOnlyAnswersThatChangeSomething(t *testing.T) {
	s := convertedOccurrenceFixture(t)
	status := enrichment.Failed
	var calls int
	registry := enrichment.NewRegistry()
	require.NoError(t, registry.Register(chainDescriptor("flaky", 100, enrichment.ChainIdentity), chainProvider{"f-instance",
		func(context.Context, enrichment.Request) enrichment.Result {
			calls++
			return enrichment.Result{Status: status}
		}}))
	require.NoError(t, s.ConfigureEnrichmentProviders(t.Context(), registry))
	id := chainReceipt(t, s, "Flaky", 1700001700)
	again := func() {
		execSQL(t, s, `UPDATE enrichment_jobs SET state='pending',lease=NULL,lease_until=NULL,attempts=0 WHERE receipt_id=?`, id)
		_, err := s.ResolveNextEnrichmentChain(t.Context(), registry)
		require.NoError(t, err)
	}
	_, err := s.ResolveNextEnrichmentChain(t.Context(), registry)
	require.NoError(t, err)
	require.Equal(t, "failed", jobState(t, s, id))
	require.Equal(t, []string{"flaky/identity=failed"}, chainRuns(t, s, id), "a failure is stored")

	// The failure is asked again, and the later answer replaces it.
	status = enrichment.Absent
	again()
	require.Equal(t, 2, calls, "the failed run was asked again")
	require.Equal(t, "absent", jobState(t, s, id))
	require.Empty(t, chainRuns(t, s, id), "the stale failure is dropped and nothing is stored for 'nothing to add'")

	// Nothing stored means the source is simply asked again if the job is ever recovered.
	again()
	require.Equal(t, 3, calls)
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM enrichment_provider_runs`))
}

// TestEnrichmentProviderRegistryKeepsOwnerChoices checks the stored rows across restarts.
func TestEnrichmentProviderRegistryKeepsOwnerChoices(t *testing.T) {
	s := convertedOccurrenceFixture(t)
	registry := enrichment.NewRegistry()
	require.NoError(t, registry.Register(chainDescriptor("extra", 150, enrichment.ChainIdentity), nil))
	require.NoError(t, s.ConfigureEnrichmentProviders(t.Context(), registry))
	rows, err := s.EnrichmentProviders(t.Context())
	require.NoError(t, err)
	var ids []string
	for _, row := range rows {
		ids = append(ids, fmt.Sprintf("%s:%d:%t:%t", row.ID, row.Priority, row.Enabled, row.Available))
	}
	// A new source appears with its default order and no migration; the old ones are not available here.
	require.Equal(t, []string{"navidrome-subsonic:100:true:false", "extra:150:true:false", "listenbrainz:200:true:false"}, ids)

	require.NoError(t, s.SetEnrichmentProvider(t.Context(), "extra", false, 400))
	require.ErrorContains(t, s.SetEnrichmentProvider(t.Context(), "nobody", true, 1), "unknown enrichment provider")
	require.Error(t, s.SetEnrichmentProvider(t.Context(), "extra", true, -1))

	// A restart reconciles availability and chains but keeps the owner's enabled flag and priority.
	changed := enrichment.NewRegistry()
	require.NoError(t, changed.Register(chainDescriptor("extra", 150, enrichment.ChainIdentity, enrichment.ChainMetadata),
		chainProvider{"x", func(context.Context, enrichment.Request) enrichment.Result {
			return enrichment.Result{Status: enrichment.Absent}
		}}))
	require.NoError(t, s.ConfigureEnrichmentProviders(t.Context(), changed))
	rows, err = s.EnrichmentProviders(t.Context())
	require.NoError(t, err)
	var extra EnrichmentProvider
	for _, row := range rows {
		if row.ID == "extra" {
			extra = row
		}
		if row.ID == "navidrome-subsonic" || row.ID == "listenbrainz" {
			require.False(t, row.Available, "a source this build does not provide is unavailable, though its row is kept")
		}
	}
	require.False(t, extra.Enabled)
	require.Equal(t, 400, extra.Priority)
	require.True(t, extra.Available)
	require.True(t, extra.Identity)
	require.True(t, extra.Metadata)
	require.NoError(t, checkDatabase(s.db))
}

// TestEnrichmentQueueFollowsTheListenBrainzSwitch shows the owner can stop relay-bound lookups.
func TestEnrichmentQueueFollowsTheListenBrainzSwitch(t *testing.T) {
	s := convertedOccurrenceFixture(t)
	require.NoError(t, s.ConfigureEnrichment(t.Context(), false))
	approveListenBrainz175(t, s, 1, enrichment.ListenBrainzBase, "profile-one")

	require.NoError(t, s.SetEnrichmentProvider(t.Context(), "listenbrainz", false, 200))
	off := chainReceipt(t, s, "Switched off", 1700001800)
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM enrichment_jobs WHERE receipt_id=?`, off))

	require.NoError(t, s.SetEnrichmentProvider(t.Context(), "listenbrainz", true, 200))
	on := chainReceipt(t, s, "Switched on", 1700001900)
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM enrichment_jobs WHERE receipt_id=?`, on))
}

// TestEnrichmentChainProviderCannotWriteUnderAnotherSource keeps a source from speaking for another.
func TestEnrichmentChainProviderCannotWriteUnderAnotherSource(t *testing.T) {
	s := convertedOccurrenceFixture(t)
	registry := enrichment.NewRegistry()
	require.NoError(t, registry.Register(chainDescriptor("honest", 100, enrichment.ChainIdentity), chainProvider{"h-instance",
		func(context.Context, enrichment.Request) enrichment.Result {
			// The declaration claims another provider's source, so it must not be stored.
			return enrichment.Result{Status: enrichment.Found, Declarations: []enrichment.Declaration{recordingDeclaration("navidrome-subsonic")}}
		}}))
	require.NoError(t, s.ConfigureEnrichmentProviders(t.Context(), registry))
	id := chainReceipt(t, s, "Spoof", 1700002000)
	found, err := s.ResolveNextEnrichmentChain(t.Context(), registry)
	require.NoError(t, err)
	require.True(t, found)
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM metadata_values`))
	require.Equal(t, []string{"honest/identity=found"}, chainRuns(t, s, id))
}
