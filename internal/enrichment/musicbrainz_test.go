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

// recordingLetters has hex letters, so its upper and lower case spellings really differ.
const recordingLetters = "abcdefab-cdef-4abc-8def-abcdefabcdef"

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
	// The comparison is the one measured in ISSUE-215: case and spaces only, punctuation stays.
	require.Equal(t, normalizedMetadata("  THE   Album "), normalizedMetadata("the album"))
	require.NotEqual(t, normalizedMetadata("Help!"), normalizedMetadata("Help"))
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

// searchProvider serves the given pages in order and counts the use of the budget.
func searchProvider(t *testing.T, pages ...func(http.ResponseWriter)) (*MusicBrainz, *mbCountingBudget) {
	t.Helper()
	call := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := pages[min(call, len(pages)-1)]
		call++
		page(w)
	}))
	t.Cleanup(server.Close)
	budget := &mbCountingBudget{client: server.Client()}
	return NewMusicBrainz(server.URL, "test", budget), budget
}

func page(count int, recordings ...any) func(http.ResponseWriter) {
	return func(w http.ResponseWriter) {
		_ = json.NewEncoder(w).Encode(map[string]any{"count": count, "recordings": recordings})
	}
}

var searchSnapshot = ingest.Snapshot{Title: "Title", Artist: "Artist", Release: "Album", DurationMS: 200000}

// TestMusicBrainzSameRecordingInDifferentCaseIsNotAmbiguous dedupes by the canonical UUID.
func TestMusicBrainzSameRecordingInDifferentCaseIsNotAmbiguous(t *testing.T) {
	require.NotEqual(t, recordingLetters, strings.ToUpper(recordingLetters))
	provider, _ := searchProvider(t, page(0, candidate(strings.ToUpper(recordingLetters), 200000, "Album", "Artist", 100), candidate(recordingLetters, 200001, "Album", "Artist", 100)))
	got := provider.Lookup(t.Context(), Request{Snapshot: searchSnapshot})
	require.Equal(t, Found, got.Status)
	require.Equal(t, recordingLetters, got.Declarations[0].Value, "the declared ID is the lowercase canonical form")
}

// TestMusicBrainzFullPageWithoutCountIsAmbiguous treats a full page as possibly truncated even when
// the response carries no count, and accepts one recording below the page size.
func TestMusicBrainzFullPageWithoutCountIsAmbiguous(t *testing.T) {
	filler := func(n int) []any {
		out := make([]any, n)
		for i := range out {
			out[i] = candidate(recordingB, 90000, "Other", "Other", 10)
		}
		return out
	}
	for _, tc := range []struct {
		name   string
		fill   int
		status Status
	}{{"99 results", 98, Found}, {"100 results", 99, Ambiguous}} {
		t.Run(tc.name, func(t *testing.T) {
			recordings := append([]any{candidate(recordingA, 200000, "Album", "Artist", 100)}, filler(tc.fill)...)
			provider, _ := searchProvider(t, page(0, recordings...))
			got := provider.Lookup(t.Context(), Request{Snapshot: searchSnapshot})
			require.Equal(t, tc.status, got.Status)
			if tc.status == Ambiguous {
				require.Empty(t, got.Declarations)
			}
		})
	}
}

// TestMusicBrainzKnownRecordingMustMatchTheCandidate rejects a candidate that disagrees with the
// recording ID the listen already carries.
func TestMusicBrainzKnownRecordingMustMatchTheCandidate(t *testing.T) {
	provider, _ := searchProvider(t, page(0, candidate(recordingA, 200000, "Album", "Artist", 100)))
	s := searchSnapshot
	s.Recording = recordingB
	got := provider.Lookup(t.Context(), Request{Snapshot: s})
	require.Equal(t, Absent, got.Status)
	require.Empty(t, got.Declarations)
}

// TestMusicBrainzFailureIsNotCached repeats a failed search and caches only the answers it can trust.
func TestMusicBrainzFailureIsNotCached(t *testing.T) {
	provider, budget := searchProvider(t,
		func(w http.ResponseWriter) { w.WriteHeader(http.StatusServiceUnavailable) },
		page(0, candidate(recordingA, 200000, "Album", "Artist", 100)))
	require.Equal(t, Failed, provider.Lookup(t.Context(), Request{Snapshot: searchSnapshot}).Status)
	require.Equal(t, Found, provider.Lookup(t.Context(), Request{Snapshot: searchSnapshot}).Status)
	require.Equal(t, 2, budget.calls, "the failure was asked again")
	require.Equal(t, Found, provider.Lookup(t.Context(), Request{Snapshot: searchSnapshot}).Status)
	require.Equal(t, 2, budget.calls, "the found answer is cached")
}

// TestMusicBrainzPunctuationOnlyTitleIsNotSearched keeps a title the analyzer would drop from
// widening the query to the whole album.
func TestMusicBrainzPunctuationOnlyTitleIsNotSearched(t *testing.T) {
	provider, budget := searchProvider(t, page(0, candidate(recordingA, 200000, "Album", "Artist", 100)))
	for _, title := range []string{"?", "...", " ! "} {
		s := searchSnapshot
		s.Title = title
		require.Equal(t, Absent, provider.Lookup(t.Context(), Request{Snapshot: s}).Status, title)
	}
	require.Zero(t, budget.calls, "no request was made")
	require.True(t, hasWordCharacter("Ünïcode 1"))
	require.False(t, hasWordCharacter("?!…"))
}

// TestMusicBrainzAlbumPunctuationMatters follows the comparison measured in ISSUE-215.
func TestMusicBrainzAlbumPunctuationMatters(t *testing.T) {
	provider, _ := searchProvider(t, page(0, candidate(recordingA, 200000, "Help", "Artist", 100)))
	s := searchSnapshot
	s.Release = "Help!"
	require.Equal(t, Absent, provider.Lookup(t.Context(), Request{Snapshot: s}).Status)
}
