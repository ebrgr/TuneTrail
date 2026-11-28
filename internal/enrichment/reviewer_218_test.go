package enrichment

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/stretchr/testify/require"
)

// TestReviewerSubsonicPartialIdentityRetries covers detail failures without losing identity.
func TestReviewerSubsonicPartialIdentityRetries(t *testing.T) {
	for _, failing := range []string{"song", "album", "mismatch"} {
		t.Run(failing, func(t *testing.T) {
			searches := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case strings.HasSuffix(r.URL.Path, "/search3"):
					searches++
					fmt.Fprintf(w, `{"subsonic-response":{"status":"ok","searchResult3":{"song":[{"id":"song","title":"Title","artist":"Artist","album":"Album","mediaType":"song","musicBrainzId":"%s"}]}}}`, recordingID)
				case strings.HasSuffix(r.URL.Path, "/getSong"):
					if failing == "song" {
						w.WriteHeader(500)
						return
					}
					fmt.Fprint(w, `{"subsonic-response":{"status":"ok","song":{"id":"song","albumId":"album","title":"Title","artist":"Artist","album":"Album","year":2005}}}`)
				case strings.HasSuffix(r.URL.Path, "/getAlbum"):
					if failing == "album" {
						w.WriteHeader(500)
						return
					}
					fmt.Fprint(w, `{"subsonic-response":{"status":"ok","album":{"id":"album","name":"Other"}}}`)
				}
			}))
			defer srv.Close()
			p, err := NewSubsonic(srv.URL, "", "test", time.Second, time.Nanosecond)
			require.NoError(t, err)
			req := Request{Chain: ChainIdentity, Snapshot: ingest.Snapshot{Title: "Title", Artist: "Artist", Release: "Album"}}
			for i := 0; i < 2; i++ {
				result := p.Lookup(t.Context(), req)
				require.Equal(t, Found, result.Status)
				require.NotEmpty(t, result.Declarations)
				require.Equal(t, recordingID, result.Declarations[0].Value)
				if failing != "song" {
					require.Len(t, result.Declarations, 2)
				}
			}
			require.Equal(t, 2, searches)
		})
	}
}

// TestReviewerISRCFailuresFallBack distinguishes 404 and transient failures.
func TestReviewerISRCFailuresFallBack(t *testing.T) {
	for _, code := range []int{404, 500} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.Contains(r.URL.Path, "/isrc/") {
					w.WriteHeader(code)
					return
				}
				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"recordings": []any{candidate(recordingA, nil, "Album", "Artist", 100)}}))
			}))
			defer srv.Close()
			p := NewMusicBrainz(srv.URL, "test", &mbCountingBudget{client: srv.Client()})
			result := p.Lookup(t.Context(), Request{Snapshot: ingest.Snapshot{Title: "Title", Artist: "Artist", Release: "Album"}, Prior: []Declaration{{Entity: "recording", Field: "isrc", Value: "USABC1234567"}}})
			require.Equal(t, Found, result.Status)
		})
	}
}

// TestReviewerMultipleISRCsJoinRecordingIDs checks distinct recordings and caching.
func TestReviewerMultipleISRCsJoinRecordingIDs(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"recordings": []any{map[string]any{"id": recordingA}}}))
	}))
	defer srv.Close()
	p := NewMusicBrainz(srv.URL, "test", &mbCountingBudget{client: srv.Client()})
	req := Request{Snapshot: ingest.Snapshot{Title: "Title", Artist: "Artist", Release: "Album"}, Prior: []Declaration{{Entity: "recording", Field: "isrc", Value: "USABC1234568"}, {Entity: "recording", Field: "isrc", Value: "USABC1234567"}}}
	require.Equal(t, Found, p.Lookup(t.Context(), req).Status)
	require.Equal(t, Found, p.Lookup(t.Context(), req).Status)
	require.Equal(t, 2, calls)
}

// TestReviewerISRCCredits verifies artist identifiers from ISRC results.
func TestReviewerISRCCredits(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "artist-credits", r.URL.Query().Get("inc"))
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"recordings": []any{candidate(recordingA, nil, "Album", "Artist", 100)}}))
	}))
	defer srv.Close()
	p := NewMusicBrainz(srv.URL, "test", &mbCountingBudget{client: srv.Client()})
	result := p.Lookup(t.Context(), Request{Snapshot: ingest.Snapshot{Title: "Title", Artist: "Artist", Release: "Album"}, Prior: []Declaration{{Entity: "recording", Field: "isrc", Value: "USABC1234567"}}})
	require.Equal(t, Found, result.Status)
	require.Len(t, result.Declarations, 2)
	require.Equal(t, mbArtistID, result.Declarations[1].Value)
}

// TestReviewerISRCPartialFailureDoesNotBecomeNegative ensures uncertain lookups are not cached.
func TestReviewerISRCPartialFailureDoesNotBecomeNegative(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if strings.Contains(r.URL.Path, "/isrc/") {
			if strings.HasSuffix(r.URL.Path, "8") {
				w.WriteHeader(500)
				return
			}
			fmt.Fprintf(w, `{"recordings":[{"id":"%s"}]}`, recordingA)
			return
		}
		fmt.Fprint(w, `{"recordings":[]}`)
	}))
	defer srv.Close()
	p := NewMusicBrainz(srv.URL, "test", &mbCountingBudget{client: srv.Client()})
	req := Request{Snapshot: ingest.Snapshot{Title: "Title", Artist: "Artist", Release: "Album"}, Prior: []Declaration{{Entity: "recording", Field: "isrc", Value: "USABC1234567"}, {Entity: "recording", Field: "isrc", Value: "USABC1234568"}}}
	require.Equal(t, Failed, p.Lookup(t.Context(), req).Status)
	require.Equal(t, Failed, p.Lookup(t.Context(), req).Status)
	require.Equal(t, 6, calls)
}

// TestReviewerMultipleISRCDistinctRecordings validates all codes in deterministic order.
func TestReviewerMultipleISRCDistinctRecordings(t *testing.T) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		id := recordingA
		if strings.HasSuffix(r.URL.Path, "8") {
			id = recordingB
		}
		fmt.Fprintf(w, `{"recordings":[{"id":"%s"}]}`, id)
	}))
	defer srv.Close()
	p := NewMusicBrainz(srv.URL, "test", &mbCountingBudget{client: srv.Client()})
	result := p.Lookup(t.Context(), Request{Snapshot: ingest.Snapshot{Title: "Title", Artist: "Artist", Release: "Album"}, Prior: []Declaration{{Entity: "recording", Field: "isrc", Value: "USABC1234568"}, {Entity: "recording", Field: "isrc", Value: "USABC1234567"}}})
	require.Equal(t, Ambiguous, result.Status)
	require.Equal(t, []string{"/ws/2/isrc/USABC1234567", "/ws/2/isrc/USABC1234568"}, paths)
}

// TestReviewerISRC404IsAbsent treats only the ISRC endpoint's 404 as absence.
func TestReviewerISRC404IsAbsent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotFound) }))
	defer srv.Close()
	p := NewMusicBrainz(srv.URL, "test", &mbCountingBudget{client: srv.Client()})
	require.Equal(t, Absent, p.lookupISRC(t.Context(), "USABC1234567").Status)
	require.Equal(t, Failed, p.Lookup(t.Context(), Request{Snapshot: ingest.Snapshot{Title: "Title", Artist: "Artist", Release: "Album"}}).Status)
}

// TestReviewerSubsonicIncompleteEvidenceStillQueries preserves identity and missing metadata work.
func TestReviewerSubsonicIncompleteEvidenceStillQueries(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		fmt.Fprint(w, `{"subsonic-response":{"status":"ok","searchResult3":{"song":[]}}}`)
	}))
	defer srv.Close()
	p, err := NewSubsonic(srv.URL, "", "test", time.Second, time.Nanosecond)
	require.NoError(t, err)
	known := ingest.Snapshot{Title: "Title", Artist: "Artist", Release: "Album", Recording: recordingA, ArtistMBIDs: []string{mbArtistID}}
	require.Equal(t, Absent, p.Lookup(t.Context(), Request{Chain: ChainIdentity, MetadataPresent: true, Snapshot: known}).Status)
	require.Zero(t, calls)
	require.Equal(t, Absent, p.Lookup(t.Context(), Request{Chain: ChainIdentity, MetadataPresent: false, Snapshot: known}).Status)
	require.Equal(t, 1, calls)
	known.Recording = ""
	require.Equal(t, Absent, p.Lookup(t.Context(), Request{Chain: ChainIdentity, MetadataPresent: true, Snapshot: known}).Status)
	require.Equal(t, 2, calls)
}

// TestReviewerISRCCreditsRejectKnownArtistConflict keeps the recording while withholding unsafe artist claims.
func TestReviewerISRCCreditsRejectKnownArtistConflict(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"recordings":[{"id":"%s","artist-credit":[{"name":"A","artist":{"id":"%s"}},{"name":"B","artist":{"id":"%s"}}]}]}`, recordingA, mbArtistID, recordingB)
	}))
	defer srv.Close()
	p := NewMusicBrainz(srv.URL, "test", &mbCountingBudget{client: srv.Client()})
	snap := ingest.Snapshot{Title: "Title", Artist: "A & B", Release: "Album", ArtistNames: []string{"A", "B"}, ArtistMbidMappings: []ingest.ArtistMapping{{Name: "B", MBID: recordingA}}}
	result := p.Lookup(t.Context(), Request{Snapshot: snap, Prior: []Declaration{{Entity: "recording", Field: "isrc", Value: "USABC1234567"}}})
	require.Equal(t, Found, result.Status)
	require.Len(t, result.Declarations, 1)
	require.Equal(t, "recording", result.Declarations[0].Entity)
}
