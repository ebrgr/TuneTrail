package enrichment

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/stretchr/testify/require"
)

type mbCountingBudget struct {
	calls  int
	client *http.Client
}

// Do counts use of the registry-provided budget in tests.
func (b *mbCountingBudget) Do(ctx context.Context, job func(context.Context, *http.Client) ([]byte, error)) ([]byte, error) {
	b.calls++
	return job(ctx, b.client)
}

const recordingA = "11111111-1111-4111-8111-111111111111"
const recordingB = "22222222-2222-4222-8222-222222222222"
const mbArtistID = "33333333-3333-4333-8333-333333333333"

// candidate builds a MusicBrainz search fixture.
func candidate(id string, length any, album, artist string, score int) map[string]any {
	return map[string]any{"id": id, "score": score, "length": length, "releases": []any{map[string]any{"title": album}}, "artist-credit": []any{map[string]any{"name": artist, "artist": map[string]any{"id": mbArtistID, "name": artist}}}}
}

// TestMusicBrainzSearch verifies the acceptance rule on simulated recordings.
func TestMusicBrainzSearch(t *testing.T) {
	cases := []struct {
		name       string
		recordings []any
		status     Status
	}{
		{"studio versus live", []any{candidate(recordingA, 200000, "Album", "Artist", 100), candidate(recordingB, 240000, "Album", "Artist", 100)}, Found},
		{"other artist", []any{candidate(recordingA, 200000, "Album", "Other", 100)}, Absent},
		{"duration outside tolerance", []any{candidate(recordingA, 205001, "Album", "Artist", 100)}, Absent},
		{"unknown duration", []any{candidate(recordingA, nil, "Album", "Artist", 100)}, Absent},
		{"wrong album", []any{candidate(recordingA, 200000, "Other", "Artist", 100)}, Absent},
		{"score", []any{candidate(recordingA, 200000, "Album", "Artist", 94)}, Absent},
		{"ambiguous within tolerance", []any{candidate(recordingA, 200000, "Album", "Artist", 100), candidate(recordingB, 204000, "Album", "Artist", 100)}, Ambiguous},
		{"no candidate", nil, Absent},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "application/json", r.Header.Get("Accept"))
				require.Equal(t, "test-agent", r.Header.Get("User-Agent"))
				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"recordings": tc.recordings}))
			}))
			defer server.Close()
			budget := &mbCountingBudget{client: server.Client()}
			registry := NewRegistry()
			registry.SetBudget(BudgetMusicBrainz, budget)
			shared, _ := registry.Budget(BudgetMusicBrainz)
			provider := NewMusicBrainz(server.URL, "test-agent", shared)
			got := provider.Lookup(t.Context(), Request{Snapshot: ingest.Snapshot{Title: "Title", Artist: "Artist", Release: "Album", DurationMS: 200000}})
			require.Equal(t, tc.status, got.Status)
			if tc.status == Ambiguous {
				require.Empty(t, got.Declarations)
			}
			require.Equal(t, 1, budget.calls)
			provider.Lookup(t.Context(), Request{Snapshot: ingest.Snapshot{Title: "Title", Artist: "Artist", Release: "Album", DurationMS: 200000}})
			require.Equal(t, 1, budget.calls)
		})
	}
}

// TestMusicBrainzFailures keeps transport and response failures distinct from absence.
func TestMusicBrainzFailures(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{{"503", 503, ""}, {"429", 429, ""}, {"json", 200, "{"}, {"oversize", 200, strings.Repeat(" ", (8<<20)+1)}} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(tc.status); fmt.Fprint(w, tc.body) }))
			defer server.Close()
			b := &mbCountingBudget{client: server.Client()}
			p := NewMusicBrainz(server.URL, "test", b)
			require.Equal(t, Failed, p.Lookup(t.Context(), Request{Snapshot: ingest.Snapshot{Title: "T", Artist: "A", Release: "R"}}).Status)
			require.Equal(t, 1, b.calls)
		})
	}
}

// TestMusicBrainzPriorAndEscape checks early exit and Lucene escaping.
func TestMusicBrainzPriorAndEscape(t *testing.T) {
	b := &mbCountingBudget{}
	p := NewMusicBrainz("http://invalid", "test", b)
	s := ingest.Snapshot{Title: "T", Artist: "A", Release: "R", ArtistMBIDs: []string{mbArtistID}}
	r := p.Lookup(t.Context(), Request{Snapshot: s, Prior: []Declaration{{Entity: "recording", Field: "musicbrainz_id", Value: recordingA}}})
	require.Equal(t, Absent, r.Status)
	require.Zero(t, b.calls)
	require.Equal(t, `\"\\\+\-\&\&\|\|\!\(\)\{\}\[\]\^\~\*\?\:\/`, luceneEscape(`"\+-&&||!(){}[]^~*?:/`))
}

// TestMusicBrainzISRC verifies the disconnected lookup and its uniqueness rule.
func TestMusicBrainzISRC(t *testing.T) {
	for _, tc := range []struct {
		ids  []any
		want Status
	}{{nil, Absent}, {[]any{map[string]any{"id": recordingA}}, Found}, {[]any{map[string]any{"id": recordingA}, map[string]any{"id": recordingB}}, Ambiguous}} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, "/ws/2/isrc/USABC1234567", r.URL.Path)
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"recordings": tc.ids}))
		}))
		b := &mbCountingBudget{client: server.Client()}
		require.Equal(t, tc.want, NewMusicBrainz(server.URL, "test", b).lookupISRC(t.Context(), "USABC1234567").Status)
		server.Close()
	}
}

// TestMusicBrainzQueryAndCredits checks the actual request and field-level declarations.
func TestMusicBrainzQueryAndCredits(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, `recording:"A\"B\+" AND artist:"Artist" AND release:"Album"`, r.URL.Query().Get("query"))
		require.Equal(t, "100", r.URL.Query().Get("limit"))
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"recordings": []any{candidate(recordingA, nil, "Album", "Artist", 100)}}))
	}))
	defer server.Close()
	b := &mbCountingBudget{client: server.Client()}
	result := NewMusicBrainz(server.URL, "test", b).Lookup(t.Context(), Request{Snapshot: ingest.Snapshot{Title: `A"B+`, Artist: "Artist", Release: "Album"}})
	require.Equal(t, Found, result.Status)
	require.Len(t, result.Declarations, 2)
	require.Equal(t, "recording", result.Declarations[0].Entity)
	require.Equal(t, "artist", result.Declarations[1].Entity)
	require.Equal(t, musicBrainzSource, result.Declarations[0].Source)
	require.False(t, result.Declarations[0].ObservedAt.IsZero())
	require.Equal(t, 1, b.calls)
}

// TestMusicBrainzConservativeEvidence rejects conflicting IDs and incomplete pages.
func TestMusicBrainzConservativeEvidence(t *testing.T) {
	for _, tc := range []struct {
		name     string
		count    int
		prior    []Declaration
		snapshot ingest.Snapshot
		status   Status
	}{
		{name: "artist conflict", prior: []Declaration{{Entity: "artist", Name: "Artist", Field: "musicbrainz_id", Value: recordingB}}, status: Absent},
		{name: "known recording conflict", prior: []Declaration{{Entity: "recording", Field: "musicbrainz_id", Value: recordingB}}, status: Absent},
		{name: "incomplete page", count: 101, status: Ambiguous},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"count": tc.count, "recordings": []any{candidate(recordingA, 200000, "Album", "Artist", 100)}}))
			}))
			defer server.Close()
			b := &mbCountingBudget{client: server.Client()}
			s := ingest.Snapshot{Title: "Title", Artist: "Artist", Release: "Album", DurationMS: 200000}
			result := NewMusicBrainz(server.URL, "test", b).Lookup(t.Context(), Request{Snapshot: s, Prior: tc.prior})
			require.Equal(t, tc.status, result.Status)
			require.Empty(t, result.Declarations)
		})
	}
	require.NotEqual(t, normalizedMetadata("ab c"), normalizedMetadata("a bc"))
	require.Equal(t, normalizedMetadata("THE  Album!"), normalizedMetadata("the album"))
}

// TestMusicBrainzRedirectDoesNotBypassBudget rejects automatic follow-up requests.
func TestMusicBrainzRedirectDoesNotBypassBudget(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			http.Redirect(w, r, "/other", http.StatusFound)
			return
		}
		t.Error("redirect followed outside budget")
	}))
	defer server.Close()
	b := &mbCountingBudget{client: server.Client()}
	got := NewMusicBrainz(server.URL, "test", b).Lookup(t.Context(), Request{Snapshot: ingest.Snapshot{Title: "T", Artist: "A", Release: "R"}})
	require.Equal(t, Failed, got.Status)
	require.Equal(t, 1, b.calls)
	require.Equal(t, 1, calls)
}

// TestMusicBrainzCreditNameRequiresTheObservedCredit rejects a matching canonical name with a different credit.
func TestMusicBrainzCreditNameRequiresTheObservedCredit(t *testing.T) {
	different := candidate(recordingA, 200000, "Album", "Different credit", 100)
	different["artist-credit"] = []any{map[string]any{"name": "Different credit", "artist": map[string]any{"id": mbArtistID, "name": "Artist"}}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"recordings": []any{different}}))
	}))
	defer server.Close()
	budget := &mbCountingBudget{client: server.Client()}
	got := NewMusicBrainz(server.URL, "test", budget).Lookup(t.Context(), Request{Snapshot: ingest.Snapshot{Title: "Title", Artist: "Artist", Release: "Album", DurationMS: 200000}})
	require.Equal(t, Absent, got.Status)
}
