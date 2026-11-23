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

	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/ebrgr/TuneTrail/internal/mbz"
	"github.com/ebrgr/TuneTrail/internal/relay"
	"github.com/ebrgr/TuneTrail/internal/relayoutbox"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// startFixtureRelay permits a loopback receiver and runs the production outbox worker.
func startFixtureRelay(t *testing.T, app *profileTestApp, profile int32, url string) {
	t.Helper()
	ctx := t.Context()
	require.NoError(t, app.store.ActivateRelayDelivery(ctx))
	host := strings.TrimPrefix(url, "http://")
	require.NoError(t, app.store.SetRelayPermission(ctx, profile, relay.Permission{HostPort: host, HTTP: true}))
	require.NoError(t, app.store.SetRelayPermission(ctx, profile, relay.Permission{HostPort: host, Private: true, Prefix: netip.MustParsePrefix("127.0.0.1/32")}))
	claims, stopClaims := context.WithCancel(ctx)
	attempts, cancel := context.WithCancel(ctx)
	done := relayoutbox.Run(attempts, claims, app.store)
	t.Cleanup(func() { stopClaims(); cancel(); <-done })
}

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

	const legacyItem = `{"listened_at":1700000000,"track_metadata":{"artist_name":"Fixture Artist","track_name":"Old Track","release_name":"Album"}}`
	const newItem = `{"listened_at":1700000010,"track_metadata":{"artist_name":"Fixture Artist","track_name":"New Track","release_name":"Album"}}`
	legacy := `{"listen_type":"single","payload":[` + legacyItem + `]}`
	items, err := app.store.AdmitOccurrence(ctx, ingest.Batch{
		ProfileID: 1, Source: ingest.ShadowSourceListenBrainz, Operation: "single",
		ReceivedAt: time.Now(), ClientKey: ingest.Digest([]byte("old")),
		Items: []ingest.Item{{ListenedAt: time.Unix(1700000000, 0), Snapshot: ingest.Snapshot{
			Artist: "Fixture Artist", Title: "Old Track", Release: "Album", OriginalTime: "1700000000",
		}}},
	})
	require.NoError(t, err)
	require.Len(t, items, 1)
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
	// Resolve the older receipt after approval to verify the admission fence.
	approveFixtureRelay(t, app, 1)
	found, err := app.store.ResolveNextIngest(ctx, &mbz.MbzErrorCaller{})
	require.NoError(t, err)
	require.True(t, found)
	submit(legacy, "old")
	mixed := `{"listen_type":"import","payload":[` + legacyItem + "," + newItem + `]}`
	submit(mixed, "mixed")
	submit(mixed, "mixed")
	require.Empty(t, calls)
	for range 2 {
		found, err := app.store.ResolveNextIngest(ctx, &mbz.MbzErrorCaller{})
		require.NoError(t, err)
		require.True(t, found)
	}
	legacyRows, err := app.store.Count("SELECT count(*) FROM outbox WHERE state='discarded' AND json_extract(result,'$.reason')='pre_082_best_effort'")
	require.NoError(t, err)
	require.Zero(t, legacyRows)
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

// TestSlowRelayDoesNotBlockOtherProfile keeps another destination live during a stalled POST.
func TestSlowRelayDoesNotBlockOtherProfile(t *testing.T) {
	app := newProfileTestApp(t)
	ctx := t.Context()
	require.NoError(t, app.store.ActivateRelayDelivery(ctx))

	slowEntered := make(chan struct{})
	fastEntered := make(chan struct{})
	release := make(chan struct{})
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		close(slowEntered)
		select {
		case <-release:
		case <-r.Context().Done():
		}
		w.WriteHeader(200)
	}))
	defer slow.Close()
	fast := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		close(fastEntered)
		w.WriteHeader(200)
	}))
	defer fast.Close()
	defer close(release)
	for profile, target := range map[int32]string{1: slow.URL, 2: fast.URL} {
		host := strings.TrimPrefix(target, "http://")
		require.NoError(t, app.store.SetProfileRelay(ctx, profile, target, "fictional-token"))
		require.NoError(t, app.store.SetRelayPermission(ctx, profile, relay.Permission{HostPort: host, HTTP: true}))
		require.NoError(t, app.store.SetRelayPermission(ctx, profile, relay.Permission{HostPort: host, Private: true, Prefix: netip.MustParsePrefix("127.0.0.1/32")}))
		approveFixtureRelay(t, app, profile)
	}
	submit := func(key string) {
		w := app.request("POST", "/apis/listenbrainz/1/submit-listens", listenBody("single", "Fixture", 1700000000), key, false)
		require.Equal(t, 200, w.Code, w.Body.String())
		found, err := app.store.ResolveNextIngest(ctx, &mbz.MbzErrorCaller{})
		require.NoError(t, err)
		require.True(t, found)
	}
	submit("first-key")
	attempts, cancel := context.WithCancel(ctx)
	claims, stopClaims := context.WithCancel(attempts)
	done := relayoutbox.Run(attempts, claims, app.store)
	defer func() { stopClaims(); cancel(); <-done }()
	select {
	case <-slowEntered:
	case <-time.After(5 * time.Second):
		t.Fatal("slow destination did not receive a POST")
	}
	submit("second-key")
	select {
	case <-fastEntered:
	case <-time.After(5 * time.Second):
		t.Fatal("slow destination blocked the second profile")
	}
}

// approveFixtureRelay uses the root confirmation path for a saved local destination.
func approveFixtureRelay(t *testing.T, app *profileTestApp, profile int32) {
	t.Helper()
	session, err := uuid.Parse(app.cookie)
	require.NoError(t, err)
	grant, err := app.store.RelayGrant(t.Context(), session, profile)
	require.NoError(t, err)
	require.NoError(t, app.store.ConfirmRelayGrant(t.Context(), session, profile, grant.Fence))
}
