package images

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// TestSubsonicClientName guards ISSUE-111: the Subsonic server records requests
// as coming from "tunetrail" in the ping and in every search/artwork URL.
func TestSubsonicClientName(t *testing.T) {
	got := ""
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Query().Get("c")
		w.Write([]byte(`{"subsonic-response":{"status":"ok"}}`))
	}))
	defer srv.Close()
	if err := TestSubsonic(context.Background(), srv.URL, "u=test&p=value"); err != nil {
		t.Fatal(err)
	}
	if got != "tunetrail" {
		t.Fatalf("ping client %q, want tunetrail", got)
	}
	for _, format := range []string{subsonicAlbumSearchFmtStr, subsonicArtistSearchFmtStr, subsonicCoverArtFmtStr} {
		u, err := url.Parse("http://x" + strings.Replace(strings.Replace(format, "%s", "a=b", 1), "%s", "q", 1))
		if err != nil {
			t.Fatal(err)
		}
		if c := u.Query().Get("c"); c != "tunetrail" {
			t.Fatalf("%s: client %q, want tunetrail", format, c)
		}
	}
}
