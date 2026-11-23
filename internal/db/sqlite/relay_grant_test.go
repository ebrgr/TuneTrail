package sqlite

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/ebrgr/TuneTrail/internal/relay"
	"github.com/ebrgr/TuneTrail/internal/testfixtures"
	"github.com/stretchr/testify/require"
)

// TestRelayGrantPlan exercises only the permissions missing from the current policy.
func TestRelayGrantPlan(t *testing.T) {
	dns := lookupFixture{
		"multi_scrobbler": {netip.MustParseAddr("192.168.1.50")},
		"mixed.test":      {netip.MustParseAddr("192.168.1.50"), netip.MustParseAddr("169.254.169.254")},
	}
	for _, tc := range []struct {
		name, endpoint     string
		policy             relay.NetworkPolicy
		http, named, local bool
		private            []string
		forbidden          bool
	}{
		{name: "HTTP only", endpoint: "http://203.0.114.2", http: true},
		{name: "private only", endpoint: "https://192.168.1.50", private: []string{"192.168.1.50/32"}},
		{name: "both", endpoint: "http://multi_scrobbler:9078/x", http: true, named: true, private: []string{"192.168.1.50/32"}},
		{name: "public HTTPS", endpoint: "https://203.0.114.2"},
		{name: "loopback", endpoint: "http://127.0.0.1", policy: relay.NetworkPolicy{AllowPrivate: true},
			http: true, local: true, private: []string{"127.0.0.1/32"}},
		{name: "link-local", endpoint: "https://[fe80::1]", policy: relay.NetworkPolicy{AllowPrivate: true},
			local: true, private: []string{"fe80::1/128"}},
		{name: "metadata", endpoint: "http://169.254.169.254", forbidden: true},
		{name: "mixed answer", endpoint: "https://mixed.test", forbidden: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			grant, err := planRelayGrant(t.Context(), dns, tc.endpoint, 1, tc.policy)
			if tc.forbidden {
				require.ErrorIs(t, err, db.ErrForbidden)
				require.False(t, grant.Available())
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.http, grant.HTTP)
			require.Equal(t, tc.named, grant.Named)
			require.Equal(t, tc.local, grant.Local)
			require.Equal(t, tc.private, grant.Private)
		})
	}
}

// TestRelayGrantRootFence verifies stale confirmation and ordinary-session rejection.
func TestRelayGrantRootFence(t *testing.T) {
	s, _, _ := migrationDB(t, currentSchemaVersion)
	ctx := t.Context()
	testfixtures.Profile(t, s, 1, "one", "One", 0)
	testfixtures.Profile(t, s, 2, "two", "Two", 0)
	require.NoError(t, s.SetProfileRelay(ctx, 1, "http://127.0.0.1:9078", "fictional"))
	execSQL(t, s, "UPDATE users SET role='admin',status='active',password=X'01' WHERE id=1")
	execSQL(t, s, "UPDATE users SET role='user',status='active',password=X'01' WHERE id=2")
	root, err := s.SaveSession(ctx, 1, time.Now().Add(time.Hour), false)
	require.NoError(t, err)
	member, err := s.SaveSession(ctx, 2, time.Now().Add(time.Hour), false)
	require.NoError(t, err)
	_, err = s.RelayPolicyProfiles(ctx, member.ID, 1)
	require.ErrorIs(t, err, db.ErrForbidden)
	_, err = s.RelayGrant(ctx, member.ID, 1)
	require.ErrorIs(t, err, db.ErrForbidden)
	preview, err := s.RelayGrant(ctx, root.ID, 1)
	require.NoError(t, err)
	require.True(t, preview.Available())
	require.ErrorIs(t, s.ConfirmRelayGrant(ctx, member.ID, 1, preview.Fence), db.ErrForbidden)
	require.NoError(t, s.SetProfileRelay(ctx, 1, "http://127.0.0.1:9078/changed", "fictional"))
	require.ErrorIs(t, s.ConfirmRelayGrant(ctx, root.ID, 1, preview.Fence), db.ErrConflict)
	require.Zero(t, scalar(t, s, "SELECT count(*) FROM relay_allowlist"))
	preview, err = s.RelayGrant(ctx, root.ID, 1)
	require.NoError(t, err)
	require.NoError(t, s.ConfirmRelayGrant(ctx, root.ID, 1, preview.Fence))
	require.Equal(t, 2, scalar(t, s, "SELECT count(*) FROM relay_allowlist WHERE host_port='127.0.0.1:9078'"))
}

// TestRelayDraftApproval fences both queued and unresolved pre-approval listens.
func TestRelayDraftApproval(t *testing.T) {
	s, _, _ := migrationDB(t, currentSchemaVersion)
	ctx := t.Context()
	testfixtures.Profile(t, s, 1, "draft", "Draft", 0)
	execSQL(t, s, "UPDATE users SET role='admin',status='active',password=X'01' WHERE id=1")
	root, err := s.SaveSession(ctx, 1, time.Now().Add(time.Hour), false)
	require.NoError(t, err)
	require.NoError(t, s.ActivateRelayDelivery(ctx))
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	require.NoError(t, s.SetProfileRelay(ctx, 1, "http://169.254.169.254", "fictional"))
	_, err = s.RelayGrant(ctx, root.ID, 1)
	require.ErrorIs(t, err, db.ErrForbidden)
	require.NoError(t, s.SetProfileRelay(ctx, 1, server.URL, "fictional"))
	draft, err := s.RelayAwaitingApproval(ctx, 1)
	require.NoError(t, err)
	require.True(t, draft)
	require.ErrorIs(t, s.ResumeRelay(ctx, root.ID, 1), db.ErrConflict)

	admit := func(stamp int64) {
		t.Helper()
		_, err := s.AdmitOccurrence(ctx, ingest.Batch{
			ProfileID: 1, Source: ingest.ShadowSourceListenBrainz, Operation: "single", ReceivedAt: time.Now(),
			Items: []ingest.Item{{ListenedAt: time.Unix(stamp, 0), Snapshot: ingest.Snapshot{Artist: "Fixture", Title: "Track"}}},
		})
		require.NoError(t, err)
	}
	resolve := func() {
		t.Helper()
		found, err := s.ResolveNextIngest(ctx, nil)
		require.NoError(t, err)
		require.True(t, found)
	}
	admit(1780000000)
	resolve()
	admit(1780000060)
	require.Zero(t, scalar(t, s, "SELECT count(*) FROM outbox"))
	require.Zero(t, requests.Load())
	grant, err := s.RelayGrant(ctx, root.ID, 1)
	require.NoError(t, err)
	require.True(t, grant.Draft)
	require.True(t, grant.HTTP)
	require.Len(t, grant.Private, 1)
	require.Zero(t, scalar(t, s, "SELECT count(*) FROM relay_allowlist"))
	require.NoError(t, s.ConfirmRelayGrant(ctx, root.ID, 1, grant.Fence))
	draft, err = s.RelayAwaitingApproval(ctx, 1)
	require.NoError(t, err)
	require.False(t, draft)
	require.Equal(t, 2, scalar(t, s, "SELECT count(*) FROM relay_allowlist"))
	resolve()
	require.Zero(t, scalar(t, s, "SELECT count(*) FROM outbox"))
	admit(1780000120)
	resolve()
	require.Equal(t, 1, scalar(t, s, "SELECT count(*) FROM outbox"))
	require.Equal(t, 3, scalar(t, s, "SELECT count(*) FROM listens"))
	require.Zero(t, requests.Load())
	batch, err := s.ClaimRelayBatch(ctx, time.Now().Add(time.Second))
	require.NoError(t, err)
	require.Len(t, batch.Items, 1)
}
