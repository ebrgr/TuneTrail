package enrichment

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/stretchr/testify/require"
)

const recordingID = "e10c78f5-5ad3-4ba2-bd43-f4ac12c62f59"
const artistID = "b10c78f5-5ad3-4ba2-bd43-f4ac12c62f59"

// TestSubsonicLookup checks matching, profile cache boundaries and received identifiers.
func TestSubsonicLookup(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		require.Equal(t, http.MethodPost, r.Method)
		require.NoError(t, r.ParseForm())
		require.Equal(t, "secret", r.Form.Get("t"))
		require.Empty(t, r.URL.RawQuery)
		if strings.HasSuffix(r.URL.Path, "/getArtist") {
			_, _ = w.Write([]byte(`{"subsonic-response":{"status":"ok","artist":{"name":"Artist","musicBrainzId":"` + artistID + `"}}}`))
			return
		}
		_, _ = w.Write([]byte(`{"subsonic-response":{"status":"ok","searchResult3":{"song":[{"title":"Title","artist":"Artist","artistId":"native","album":"Album","duration":200,"mediaType":"song","musicBrainzId":"` + recordingID + `"}]}}}`))
	}))
	defer server.Close()
	c, err := NewSubsonic(server.URL, "u=user&t=secret&s=salt", "test", time.Second, time.Nanosecond)
	require.NoError(t, err)
	req := Request{ProfileID: 1, Snapshot: ingest.Snapshot{Title: "Title", Artist: "Artist", Release: "Album", DurationMS: 200000}}
	result := c.Lookup(t.Context(), req)
	require.Equal(t, Found, result.Status)
	require.Len(t, result.Declarations, 2)
	require.Equal(t, recordingID, result.Declarations[0].Value)
	require.Equal(t, artistID, result.Declarations[1].Value)
	require.EqualValues(t, 2, calls.Load())
	c.Lookup(t.Context(), req)
	require.EqualValues(t, 2, calls.Load())
	req.ProfileID = 2
	c.Lookup(t.Context(), req)
	require.EqualValues(t, 4, calls.Load())
	req.Snapshot.Recording = recordingID
	req.Snapshot.ArtistMBIDs = []string{artistID}
	require.Equal(t, Absent, c.Lookup(t.Context(), req).Status)
	require.EqualValues(t, 4, calls.Load())
}

// TestSubsonicConservativeMatch rejects missing, divergent, ambiguous and truncated results.
func TestSubsonicConservativeMatch(t *testing.T) {
	base := subsonicSong{Title: "Title", Artist: "Artist", Album: "Album", Duration: 200, MediaType: "song", MBID: recordingID}
	for _, tc := range []struct {
		name     string
		songs    []subsonicSong
		expected Status
	}{
		{"exact", []subsonicSong{base}, Found},
		{"duplicate", []subsonicSong{base, base}, Ambiguous},
		{"truncated", make([]subsonicSong, 100), Ambiguous},
		{"different version", []subsonicSong{{Title: "Title (Live)", Artist: "Artist", Album: "Album", Duration: 200}}, Absent},
		{"duration", []subsonicSong{{Title: "Title", Artist: "Artist", Album: "Album", Duration: 190}}, Absent},
		{"missing album", []subsonicSong{{Title: "Title", Artist: "Artist", Duration: 200}}, Absent},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				response := subsonicResponse{}
				response.Response.Status = "ok"
				response.Response.Search.Songs = tc.songs
				require.NoError(t, json.NewEncoder(w).Encode(response))
			}))
			defer server.Close()
			c, err := NewSubsonic(server.URL, "u=x&t=y", "test", time.Second, time.Nanosecond)
			require.NoError(t, err)
			result := c.Lookup(t.Context(), Request{ProfileID: 1, Snapshot: ingest.Snapshot{Title: "Title", Artist: "Artist", Release: "Album", DurationMS: 200000}})
			require.Equal(t, tc.expected, result.Status)
		})
	}
}

// TestSubsonicCancellation confirms upstream failures stay terminal lookup results.
func TestSubsonicCancellation(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-release }))
	defer server.Close()
	defer close(release)
	c, err := NewSubsonic(server.URL, "u=x&t=secret", "test", 20*time.Millisecond, time.Nanosecond)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	require.Equal(t, Failed, c.Lookup(ctx, Request{ProfileID: 1, Snapshot: ingest.Snapshot{Title: "Title", Artist: "Artist", Release: "Album"}}).Status)
}
