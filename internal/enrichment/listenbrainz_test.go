package enrichment

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/stretchr/testify/require"
)

const guestID = "d10c78f5-5ad3-4ba2-bd43-f4ac12c62f59"

// TestListenBrainzLimits exercises rate, cooldown, cache scope and request size.
func TestListenBrainzLimits(t *testing.T) {
	var calls atomic.Int32
	times := make(chan time.Time, 3)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if int(n) <= cap(times) {
			times <- time.Now()
		}
		asserted := r.URL.Query()
		if len(asserted) != 5 || asserted.Get("inc") != "artist" || asserted.Get("release_name") != "Album" {
			t.Error("unexpected request parameters")
		}
		if n == 3 {
			w.Header().Set("X-RateLimit-Reset-In", "30")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = fmt.Fprintf(w, `{"recording_mbid":%q,"recording_name":"Title","artist_credit_name":"Artist","artist_mbids":[%q],"release_name":"Album"}`, recordingID, artistID)
	}))
	defer server.Close()
	client := NewListenBrainz("TuneTrail/test", server.Client().Transport)
	client.endpoint = server.URL
	request := ListenBrainzRequest{ProfileID: 1, Generation: 1, Recording: true, Artists: []string{"Artist"},
		Snapshot: ingest.Snapshot{Title: "Title", Artist: "Artist", Release: "Album"}}
	first := client.Lookup(t.Context(), "fixture-token", request)
	require.Equal(t, Found, first.Status)
	require.Len(t, first.Declarations, 2)
	require.Equal(t, first, client.Lookup(t.Context(), "fixture-token", request))
	require.EqualValues(t, 1, calls.Load())
	request.ProfileID++
	require.Equal(t, Found, client.Lookup(t.Context(), "other-token", request).Status)
	firstRequest, secondRequest := <-times, <-times
	require.GreaterOrEqual(t, secondRequest.Sub(firstRequest), 900*time.Millisecond)
	request.Generation++
	client.interval = time.Millisecond // Keep the real first rate reservation and shorten subsequent ones.
	require.Equal(t, Failed, client.Lookup(t.Context(), "rotated-token", request).Status)
	require.Equal(t, Failed, client.Lookup(t.Context(), "rotated-token", request).Status)
	require.EqualValues(t, 3, calls.Load())
	request.Snapshot.Title = strings.Repeat("é", 250)
	require.Equal(t, Absent, client.Lookup(t.Context(), "rotated-token", request).Status)
	require.Equal(t, Disabled, client.Lookup(t.Context(), "", request).Status)
	require.EqualValues(t, 3, calls.Load())
}

// TestListenBrainzNamedCredits refuses ambiguous names without discarding other credits.
func TestListenBrainzNamedCredits(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprintf(w, `{"recording_mbid":%q,"recording_name":"Title","artist_credit_name":"Artist & Guest",
			"artist_mbids":[%q,%q],"release_name":"Album","metadata":{"artist":{"artists":[
			{"name":"Guest","artist_mbid":%q},{"name":"Artist","artist_mbid":%q},{"name":"artist","artist_mbid":%q}]}}}`,
			recordingID, artistID, guestID, guestID, artistID, artistID)
	}))
	defer server.Close()
	client := NewListenBrainz("TuneTrail/test", server.Client().Transport)
	client.endpoint = server.URL
	request := ListenBrainzRequest{ProfileID: 1, Recording: true, Artists: []string{"Artist", "Guest"},
		Snapshot: ingest.Snapshot{Title: "Title", Artist: "Artist & Guest", Release: "Album"}}
	result := client.Lookup(t.Context(), "fixture-token", request)
	require.Len(t, result.Declarations, 2)
	require.Equal(t, recordingID, result.Declarations[0].Value)
	require.Equal(t, "Guest", result.Declarations[1].Name)
	require.Equal(t, guestID, result.Declarations[1].Value)
	request.Snapshot.Recording = guestID
	require.Equal(t, Ambiguous, client.Lookup(t.Context(), "fixture-token", request).Status)
	request.Snapshot.Recording = ""
	request.Snapshot.ArtistNames = []string{"Artist", "Guest"}
	request.Snapshot.ArtistMBIDs = []string{"", artistID}
	require.Equal(t, Ambiguous, client.Lookup(t.Context(), "fixture-token", request).Status)
}

// TestListenBrainzFailures discards redirects, malformed and oversized responses, and timeouts.
func TestListenBrainzFailures(t *testing.T) {
	var redirected atomic.Bool
	destination := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		redirected.Store(true)
	}))
	defer destination.Close()
	for name, handler := range map[string]http.HandlerFunc{
		"redirect": func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, destination.URL, 307) },
		"invalid":  func(w http.ResponseWriter, _ *http.Request) { _, _ = fmt.Fprint(w, `{"artist":`) },
		"large":    func(w http.ResponseWriter, _ *http.Request) { _, _ = fmt.Fprint(w, strings.Repeat(" ", 65536)+"{}") },
		"timeout":  func(_ http.ResponseWriter, r *http.Request) { <-r.Context().Done() },
	} {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(handler)
			defer server.Close()
			client := NewListenBrainz("TuneTrail/test", server.Client().Transport)
			client.endpoint = server.URL
			client.client.Timeout = 100 * time.Millisecond
			request := ListenBrainzRequest{ProfileID: 1, Recording: true,
				Snapshot: ingest.Snapshot{Title: "Title", Artist: "Artist", Release: "Album"}}
			require.Equal(t, Failed, client.Lookup(t.Context(), "fixture-token", request).Status)
		})
	}
	require.False(t, redirected.Load())
}
