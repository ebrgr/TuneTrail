package sqlite

import (
	"net/netip"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/relay"
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
	s, _ := relayStoreFixture(t)
	ctx := t.Context()
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
