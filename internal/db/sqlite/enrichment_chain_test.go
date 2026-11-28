package sqlite

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
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
	require.Empty(t, chainRuns(t, s, first), "successful runs are removed when the job closes")
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
	require.Empty(t, chainRuns(t, s, second))
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
	require.Empty(t, chainRuns(t, s, id))
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
		{"a later source resolves it", enrichment.Found, "found", 1, []string{"unclear/identity=ambiguous"}},
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
	require.Empty(t, chainRuns(t, s, id))
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
	require.Equal(t, []string{"source-b/identity=failed"}, chainRuns(t, s, id))
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
	require.Empty(t, chainRuns(t, s, id))
}

// TestMusicBrainzIdentityChain persists the middle provider's result before ListenBrainz decides.
func TestMusicBrainzIdentityChain(t *testing.T) {
	s := convertedOccurrenceFixture(t)
	approveListenBrainz175(t, s, 1, enrichment.ListenBrainzBase, "profile-one")
	var order []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "musicbrainz")
		fmt.Fprintf(w, `{"recordings":[{"id":%q,"score":100,"length":200000,"artist-credit":[{"name":"Chain artist","artist":{"name":"Chain artist","id":%q}}],"releases":[{"title":"Chain album"}]}]}`, fixtureRecording, "33333333-3333-4333-8333-333333333333")
	}))
	defer server.Close()
	registry := enrichment.NewRegistry()
	budget := chainMusicBrainzBudget{client: server.Client()}
	registry.SetBudget(enrichment.BudgetMusicBrainz, budget)
	shared, ok := registry.Budget(enrichment.BudgetMusicBrainz)
	require.True(t, ok)
	require.NoError(t, registry.Register(enrichment.SubsonicDescriptor, chainProvider{"subsonic", func(context.Context, enrichment.Request) enrichment.Result {
		order = append(order, "navidrome")
		return enrichment.Result{Status: enrichment.Absent}
	}}))
	require.NoError(t, registry.Register(enrichment.MusicBrainzDescriptor, enrichment.NewMusicBrainz(server.URL, "test", shared)))
	fallback := listenBrainz175Fixture(t, func(http.ResponseWriter, *http.Request) {
		t.Error("ListenBrainz network requested after MusicBrainz resolved all fields")
	})
	require.NoError(t, registry.Register(enrichment.ListenBrainzDescriptor, enrichment.NewListenBrainzProvider(fallback)))
	require.NoError(t, s.ConfigureEnrichmentProviders(t.Context(), registry))
	id := chainReceipt(t, s, "Chain", 1700003300)
	found, err := s.ResolveNextEnrichmentChain(t.Context(), registry)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, []string{"navidrome", "musicbrainz", "navidrome"}, order)
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM metadata_values WHERE source='musicbrainz-search' AND recording_id IS NOT NULL`))
	require.Empty(t, chainRuns(t, s, id))
}

// chainMusicBrainzBudget routes the fixture through its local server client.
type chainMusicBrainzBudget struct{ client *http.Client }

// Do invokes the job with the fixture client.
func (b chainMusicBrainzBudget) Do(ctx context.Context, job func(context.Context, *http.Client) ([]byte, error)) ([]byte, error) {
	return job(ctx, b.client)
}

// TestNavidromeISRCFeedsMusicBrainz verifies identity uses the stored ISRC in the same pass.
func TestNavidromeISRCFeedsMusicBrainz(t *testing.T) {
	s := convertedOccurrenceFixture(t)
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		fmt.Fprintf(w, `{"recordings":[{"id":%q}]}`, fixtureRecording)
	}))
	defer server.Close()
	registry := enrichment.NewRegistry()
	require.NoError(t, registry.Register(enrichment.SubsonicDescriptor, chainProvider{"synthetic-instance", func(_ context.Context, r enrichment.Request) enrichment.Result {
		if r.Chain == enrichment.ChainMetadata {
			return enrichment.Result{Status: enrichment.Absent}
		}
		return enrichment.Result{Status: enrichment.Found, Declarations: []enrichment.Declaration{{Entity: "recording", Field: "isrc", Value: "USABC0500001", Source: "navidrome-subsonic", ObservedAt: time.Now()}}}
	}}))
	require.NoError(t, registry.Register(enrichment.MusicBrainzDescriptor, enrichment.NewMusicBrainz(server.URL, "test", chainMusicBrainzBudget{server.Client()})))
	require.NoError(t, s.ConfigureEnrichmentProviders(t.Context(), registry))
	chainReceipt(t, s, "ISRC", 1700003400)
	found, err := s.ResolveNextEnrichmentChain(t.Context(), registry)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, []string{"/ws/2/isrc/USABC0500001"}, paths)
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM descriptive_metadata WHERE kind='isrc'`))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM metadata_values WHERE source='musicbrainz-search' AND recording_id IS NOT NULL`))
}

// TestEnrichmentClosingLeasePreservesAnotherWorkersRuns fences successful-run cleanup.
func TestEnrichmentClosingLeasePreservesAnotherWorkersRuns(t *testing.T) {
	s := convertedOccurrenceFixture(t)
	registry := enrichment.NewRegistry()
	require.NoError(t, registry.Register(chainDescriptor("worker", 100, enrichment.ChainIdentity), chainProvider{"fixture", func(context.Context, enrichment.Request) enrichment.Result {
		return enrichment.Result{Status: enrichment.Absent}
	}}))
	require.NoError(t, s.ConfigureEnrichmentProviders(t.Context(), registry))
	id := chainReceipt(t, s, "Lease", 1700003500)
	c, err := s.claimEnrichment(t.Context())
	require.NoError(t, err)
	require.Equal(t, id, c.receipt)
	execSQL(t, s, `INSERT INTO enrichment_provider_runs(receipt_id,provider,chain,state,instance,finished_at) VALUES(?,'worker','identity','found','fixture',1)`, id)
	execSQL(t, s, `UPDATE enrichment_jobs SET lease='new-worker' WHERE receipt_id=?`, id)
	require.NoError(t, s.closeEnrichment(t.Context(), c, []enrichmentAnswer{{status: enrichment.Found}}))
	require.Equal(t, []string{"worker/identity=found"}, chainRuns(t, s, id))
	require.Equal(t, "working", jobState(t, s, id))
}

// TestEnrichmentExhaustedRecoveryRemovesSuccessKeepsFailure covers the terminal claim path.
func TestEnrichmentExhaustedRecoveryRemovesSuccessKeepsFailure(t *testing.T) {
	s := convertedOccurrenceFixture(t)
	registry := enrichment.NewRegistry()
	for _, id := range []string{"one", "two"} {
		require.NoError(t, registry.Register(chainDescriptor(id, 100, enrichment.ChainIdentity), chainProvider{id, func(context.Context, enrichment.Request) enrichment.Result {
			return enrichment.Result{Status: enrichment.Absent}
		}}))
	}
	require.NoError(t, s.ConfigureEnrichmentProviders(t.Context(), registry))
	id := chainReceipt(t, s, "Exhausted", 1700003600)
	execSQL(t, s, `UPDATE enrichment_jobs SET state='working',lease='expired',lease_until=0,attempts=2 WHERE receipt_id=?`, id)
	execSQL(t, s, `INSERT INTO enrichment_provider_runs VALUES(?,'one','identity','found','fixture',1),(?,'two','identity','failed','fixture',1)`, id, id)
	c, err := s.claimEnrichment(t.Context())
	require.NoError(t, err)
	require.Empty(t, c.lease)
	require.Equal(t, "failed", jobState(t, s, id))
	require.Equal(t, []string{"two/identity=failed"}, chainRuns(t, s, id))
}

// TestEnrichmentExpiredLeaseCannotRemoveSuccess keeps a run for the next worker to recover.
func TestEnrichmentExpiredLeaseCannotRemoveSuccess(t *testing.T) {
	s := convertedOccurrenceFixture(t)
	registry := enrichment.NewRegistry()
	require.NoError(t, registry.Register(chainDescriptor("worker", 100, enrichment.ChainIdentity), chainProvider{"fixture", func(context.Context, enrichment.Request) enrichment.Result {
		return enrichment.Result{Status: enrichment.Absent}
	}}))
	require.NoError(t, s.ConfigureEnrichmentProviders(t.Context(), registry))
	id := chainReceipt(t, s, "Expired", 1700003700)
	c, err := s.claimEnrichment(t.Context())
	require.NoError(t, err)
	execSQL(t, s, `INSERT INTO enrichment_provider_runs VALUES(?,'worker','identity','found','fixture',1)`, id)
	execSQL(t, s, `UPDATE enrichment_jobs SET lease_until=0 WHERE receipt_id=?`, id)
	require.NoError(t, s.closeEnrichment(t.Context(), c, []enrichmentAnswer{{status: enrichment.Found}}))
	require.Equal(t, []string{"worker/identity=found"}, chainRuns(t, s, id))
	require.Equal(t, "working", jobState(t, s, id))
}

// TestEnrichmentDisabledCleanup removes only completed success runs on jobs it closes.
func TestEnrichmentDisabledCleanup(t *testing.T) {
	s := convertedOccurrenceFixture(t)
	registry := enrichment.NewRegistry()
	require.NoError(t, registry.Register(chainDescriptor("worker", 100, enrichment.ChainIdentity), chainProvider{"fixture", func(context.Context, enrichment.Request) enrichment.Result {
		return enrichment.Result{Status: enrichment.Absent}
	}}))
	require.NoError(t, s.ConfigureEnrichmentProviders(t.Context(), registry))
	id := chainReceipt(t, s, "Disabled", 1700003800)
	_, err := s.claimEnrichment(t.Context())
	require.NoError(t, err)
	execSQL(t, s, `INSERT INTO enrichment_provider_runs VALUES(?,'worker','identity','found','fixture',1)`, id)
	require.NoError(t, s.ConfigureEnrichmentProviders(t.Context(), enrichment.NewRegistry()))
	require.Equal(t, "disabled", jobState(t, s, id))
	require.Empty(t, chainRuns(t, s, id))
}

// TestDescriptiveRepeatedAndMissingValues retains the first observation without duplicating it.
func TestDescriptiveRepeatedAndMissingValues(t *testing.T) {
	s := convertedOccurrenceFixture(t)
	value := "Rock"
	registry := enrichment.NewRegistry()
	require.NoError(t, registry.Register(chainDescriptor("source", 100, enrichment.ChainIdentity), chainProvider{"fixture", func(context.Context, enrichment.Request) enrichment.Result {
		if value == "" {
			return enrichment.Result{Status: enrichment.Absent}
		}
		return enrichment.Result{Status: enrichment.Found, Declarations: []enrichment.Declaration{{Entity: "recording", Field: "genre", Value: value, Source: "source", ObservedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}}}
	}}))
	require.NoError(t, s.ConfigureEnrichmentProviders(t.Context(), registry))
	for i := 0; i < 3; i++ {
		id := admitOccurrenceFixture(t, s, ingest.Snapshot{Title: "Repeat", Artist: "Repeat artist", Release: "Repeat album"}, 1700004000+int64(i), fmt.Sprintf("repeat-%d", i))
		resolveOccurrenceFixture(t, s, &mbz.MbzErrorCaller{})
		if i == 2 {
			value = ""
		}
		_, err := s.ResolveNextEnrichmentChain(t.Context(), registry)
		require.NoError(t, err)
		require.Empty(t, chainRuns(t, s, id))
	}
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM descriptive_metadata WHERE kind='genre' AND value='Rock'`))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM descriptive_metadata WHERE observed_at='2026-01-01T00:00:00Z'`))
}

// TestSubsonicChainSharesLookup collects identity and metadata with one set of requests.
func TestSubsonicChainSharesLookup(t *testing.T) {
	s := convertedOccurrenceFixture(t)
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		switch r.URL.Path {
		case "/rest/search3":
			fmt.Fprint(w, `{"subsonic-response":{"status":"ok","searchResult3":{"song":[{"id":"song","title":"One","artist":"One artist","album":"One album"}]}}}`)
		case "/rest/getSong":
			fmt.Fprint(w, `{"subsonic-response":{"status":"ok","song":{"id":"song","albumId":"album","title":"One","artist":"One artist","album":"One album","year":2005,"genres":[{"name":"Rock"}],"isrc":["USABC0500001"],"bpm":120}}}`)
		case "/rest/getAlbum":
			fmt.Fprint(w, `{"subsonic-response":{"status":"ok","album":{"id":"album","name":"One album","releaseDate":{"year":2005,"month":4},"originalReleaseDate":{"year":2004}}}}`)
		default:
			t.Errorf("unexpected request: %s", r.URL.Path)
		}
	}))
	defer server.Close()
	provider, err := enrichment.NewSubsonic(server.URL, "", "test", time.Second, time.Nanosecond)
	require.NoError(t, err)
	registry := enrichment.NewRegistry()
	require.NoError(t, registry.Register(enrichment.SubsonicDescriptor, provider))
	require.NoError(t, s.ConfigureEnrichmentProviders(t.Context(), registry))
	chainReceipt(t, s, "One", 1700004100)
	_, err = s.ResolveNextEnrichmentChain(t.Context(), registry)
	require.NoError(t, err)
	require.Equal(t, 3, calls)
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM descriptive_metadata WHERE kind='recording_year' AND value='2005'`))
	require.Equal(t, 6, scalar(t, s, `SELECT count(*) FROM descriptive_metadata m JOIN metadata_sources s ON s.id=m.source_id WHERE s.source='navidrome-subsonic' AND length(s.instance)=64 AND ((m.occurrence_id IS NOT NULL AND m.kind IN ('genre','recording_year','isrc','bpm')) OR (m.release_id IS NOT NULL AND m.kind IN ('release_date','first_release_date')))`))
}

// TestReviewerSubsonicSkipsCompleteMetadata avoids requests for an already enriched occurrence.
func TestReviewerSubsonicSkipsCompleteMetadata(t *testing.T) {
	s := convertedOccurrenceFixture(t)
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		switch r.URL.Path {
		case "/rest/search3":
			fmt.Fprint(w, `{"subsonic-response":{"status":"ok","searchResult3":{"song":[{"id":"song","title":"Complete","artist":"Artist","album":"Album"}]}}}`)
		case "/rest/getSong":
			fmt.Fprint(w, `{"subsonic-response":{"status":"ok","song":{"id":"song","albumId":"album","title":"Complete","artist":"Artist","album":"Album","year":2005}}}`)
		case "/rest/getAlbum":
			fmt.Fprint(w, `{"subsonic-response":{"status":"ok","album":{"id":"album","name":"Album"}}}`)
		}
	}))
	defer srv.Close()
	provider, err := enrichment.NewSubsonic(srv.URL, "", "test", time.Second, time.Nanosecond)
	require.NoError(t, err)
	registry := enrichment.NewRegistry()
	require.NoError(t, registry.Register(enrichment.SubsonicDescriptor, provider))
	require.NoError(t, s.ConfigureEnrichmentProviders(t.Context(), registry))
	snap := ingest.Snapshot{Title: "Complete", Artist: "Artist", Release: "Album", Recording: fixtureRecording, ArtistMBIDs: []string{"33333333-3333-4333-8333-333333333333"}}
	for i := 0; i < 2; i++ {
		admitOccurrenceFixture(t, s, snap, 1700005000+int64(i), fmt.Sprintf("complete-%d", i))
		resolveOccurrenceFixture(t, s, &mbz.MbzErrorCaller{})
		_, err = s.ResolveNextEnrichmentChain(t.Context(), registry)
		require.NoError(t, err)
		provider, err = enrichment.NewSubsonic(srv.URL, "", "test", time.Second, time.Nanosecond)
		require.NoError(t, err)
		registry = enrichment.NewRegistry()
		require.NoError(t, registry.Register(enrichment.SubsonicDescriptor, provider))
	}
	require.Equal(t, 3, calls)
	// An independent instance must not reuse another library's metadata.
	provider, err = enrichment.NewSubsonic(srv.URL, "fixture=other", "test", time.Second, time.Nanosecond)
	require.NoError(t, err)
	registry = enrichment.NewRegistry()
	require.NoError(t, registry.Register(enrichment.SubsonicDescriptor, provider))
	admitOccurrenceFixture(t, s, snap, 1700005010, "complete-other")
	resolveOccurrenceFixture(t, s, &mbz.MbzErrorCaller{})
	_, err = s.ResolveNextEnrichmentChain(t.Context(), registry)
	require.NoError(t, err)
	require.Equal(t, 6, calls)
}

// TestReviewerPartialDetailFailureRetriesOnNextListen preserves useful fields without blocking retry.
func TestReviewerPartialDetailFailureRetriesOnNextListen(t *testing.T) {
	s := convertedOccurrenceFixture(t)
	albumCalls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/search3":
			fmt.Fprint(w, `{"subsonic-response":{"status":"ok","searchResult3":{"song":[{"id":"song","title":"Retry","artist":"Artist","album":"Album"}]}}}`)
		case "/rest/getSong":
			fmt.Fprint(w, `{"subsonic-response":{"status":"ok","song":{"id":"song","albumId":"album","title":"Retry","artist":"Artist","album":"Album","year":2005}}}`)
		case "/rest/getAlbum":
			albumCalls++
			if albumCalls == 1 {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			fmt.Fprint(w, `{"subsonic-response":{"status":"ok","album":{"id":"album","name":"Album","releaseDate":{"year":2005}}}}`)
		}
	}))
	defer srv.Close()
	provider, err := enrichment.NewSubsonic(srv.URL, "", "test", time.Second, time.Nanosecond)
	require.NoError(t, err)
	registry := enrichment.NewRegistry()
	require.NoError(t, registry.Register(enrichment.SubsonicDescriptor, provider))
	require.NoError(t, s.ConfigureEnrichmentProviders(t.Context(), registry))
	snap := ingest.Snapshot{Title: "Retry", Artist: "Artist", Release: "Album", Recording: fixtureRecording, ArtistMBIDs: []string{"33333333-3333-4333-8333-333333333333"}}
	for i := 0; i < 2; i++ {
		admitOccurrenceFixture(t, s, snap, 1700005100+int64(i), fmt.Sprintf("retry-%d", i))
		resolveOccurrenceFixture(t, s, &mbz.MbzErrorCaller{})
		_, err = s.ResolveNextEnrichmentChain(t.Context(), registry)
		require.NoError(t, err)
		provider, err = enrichment.NewSubsonic(srv.URL, "", "test", time.Second, time.Nanosecond)
		require.NoError(t, err)
		registry = enrichment.NewRegistry()
		require.NoError(t, registry.Register(enrichment.SubsonicDescriptor, provider))
	}
	require.Equal(t, 2, albumCalls)
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM metadata_retries`))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM descriptive_metadata WHERE kind='release_date'`))
}
