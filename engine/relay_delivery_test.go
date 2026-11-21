package engine

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/mbz"
	"github.com/ebrgr/TuneTrail/internal/relay"
	"github.com/ebrgr/TuneTrail/internal/relayoutbox"
	"github.com/stretchr/testify/require"
)

// TestDurableRelayAdmission exercises mixed decisions and an admitted receipt across the cut.
func TestDurableRelayAdmission(t *testing.T) {
	app := newProfileTestApp(t)
	ctx := t.Context()
	const path = "/apis/listenbrainz/1/submit-listens"
	submit := func(body, key string) {
		r := httptest.NewRequest(http.MethodPost, "http://koito.test"+path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Authorization", "Token first-key")
		r.Header.Set("Idempotency-Key", key)
		w := httptest.NewRecorder()
		app.router.ServeHTTP(w, r)
		require.Equal(t, 200, w.Code, w.Body.String())
		require.JSONEq(t, `{"status":"ok"}`, w.Body.String())
	}
	require.NoError(t, app.store.Exec("INSERT INTO profile_ingest_flags(profile_id,active) VALUES(1,1)"))
	const legacyItem = `{"listened_at":1700000000,"track_metadata":{"artist_name":"Fixture Artist","track_name":"Old Track","release_name":"Album"}}`
	const newItem = `{"listened_at":1700000010,"track_metadata":{"artist_name":"Fixture Artist","track_name":"New Track","release_name":"Album"}}`
	legacy := `{"listen_type":"single","payload":[` + legacyItem + `]}`
	submit(legacy, "old")
	require.NoError(t, app.store.ActivateRelayDelivery(ctx))
	calls := make(chan string, 8)
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(500)
			return
		}
		if r.URL.Path != "/prefix/submit-listens" || r.Header.Get("Authorization") != "Token fixture-relay" {
			w.WriteHeader(403)
			return
		}
		calls <- string(body)
		w.WriteHeader(200)
	}))
	defer target.Close()
	base := target.URL + "/prefix"
	host := strings.TrimPrefix(target.URL, "http://")
	require.NoError(t, app.store.SetProfileRelay(ctx, 1, base, "fixture-relay"))
	require.NoError(t, app.store.SetRelayPermission(ctx, 1, relay.Permission{HostPort: host, HTTP: true}))
	require.NoError(t, app.store.SetRelayPermission(ctx, 1, relay.Permission{HostPort: host, Private: true, Prefix: netip.MustParsePrefix("127.0.0.1/32")}))
	submit(legacy, "old")
	mixed := `{"listen_type":"import","payload":[` + legacyItem + "," + newItem + `]}`
	submit(mixed, "mixed")
	submit(mixed, "mixed")
	require.Empty(t, calls)
	for range 3 {
		found, err := app.store.ResolveNextIngest(ctx, &mbz.MbzErrorCaller{})
		require.NoError(t, err)
		require.True(t, found)
	}
	legacyRows, err := app.store.Count("SELECT count(*) FROM outbox WHERE state='discarded' AND json_extract(result,'$.reason')='pre_082_best_effort'")
	require.NoError(t, err)
	require.Equal(t, 1, legacyRows)
	claims, stopClaims := context.WithCancel(ctx)
	attempts, cancel := context.WithCancel(ctx)
	done := relayoutbox.Run(attempts, claims, app.store)
	defer func() { stopClaims(); cancel(); <-done }()
	select {
	case body := <-calls:
		require.Contains(t, body, "New Track")
		require.NotContains(t, body, "Old Track")
		require.NotContains(t, body, "fixture-relay")
	case <-time.After(5 * time.Second):
		t.Fatal("durable relay POST not received")
	}
	require.Eventually(t, func() bool {
		count, err := app.store.Count("SELECT count(*) FROM outbox WHERE state='sent'")
		return err == nil && count == 1
	}, time.Second, 10*time.Millisecond)
	require.Empty(t, calls)
}
