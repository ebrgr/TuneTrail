package relay

import (
	"bytes"
	"errors"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/logger"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

// TestPolicyDiagnosticSanitizesAndRatesByReason keeps resolved addresses and secrets out of logs.
func TestPolicyDiagnosticSanitizesAndRatesByReason(t *testing.T) {
	const endpoint = "http://user:fictional-secret@relay.example:8080/private-path?token=fictional-query"
	err := dialPolicy("relay.example:8080", "10.44.22.17:8080", Permissions(nil), true)
	require.ErrorIs(t, err, errDialPolicy)
	d := transportDiagnostic(endpoint, PolicyBlocked, err)
	require.Equal(t, "private_network_not_allowed", d.Code)
	require.Equal(t, "rfc1918", d.Category)
	var output bytes.Buffer
	l := zerolog.New(&output)
	ctx := logger.NewContext(&l)
	now := time.Unix(1900000000, 0)
	LogDiagnostic(ctx, 712345, d, now)
	LogDiagnostic(ctx, 712345, d, now)
	LogDiagnostic(ctx, 712345, Describe(endpoint, "http_not_allowed", ""), now)
	require.Equal(t, 2, strings.Count(output.String(), "\n"))
	require.Contains(t, output.String(), "relay.example")
	for _, secret := range []string{"10.44.22.17", "fictional-secret", "private-path", "fictional-query"} {
		require.NotContains(t, output.String(), secret)
	}
	blocked := dialPolicy("relay.example:8080", "169.254.169.254:8080", allowFunc(func(string, netip.Addr) bool { return true }), true)
	var denial blockedAddress
	require.True(t, errors.As(blocked, &denial))
	require.Equal(t, "metadata_endpoint", denial.code)
}

// TestGlobalPrivatePolicyKeepsLocalExceptionsRequired checks both IP families.
func TestGlobalPrivatePolicyKeepsLocalExceptionsRequired(t *testing.T) {
	policy := NetworkPolicy{AllowHTTP: true, AllowPrivate: true}
	for _, raw := range []string{"192.168.1.1", "10.1.2.3", "172.16.1.1", "172.31.1.1", "100.64.1.1"} {
		require.True(t, policy.Allows("http", "relay.test:80", netip.MustParseAddr(raw)), raw)
	}
	for _, raw := range []string{"127.1.2.3", "::1", "169.254.8.3", "fe80::1"} {
		ip := netip.MustParseAddr(raw)
		require.False(t, policy.Allows("http", "relay.test:80", ip), raw)
		explicit := policy
		explicit.Permissions = Permissions{{HostPort: "relay.test:80", Private: true, Prefix: netip.PrefixFrom(ip, ip.BitLen())}}
		require.True(t, explicit.Allows("http", "relay.test:80", ip), raw)
	}
	for _, raw := range []string{"169.254.169.254", "fd00:ec2::254"} {
		ip := netip.MustParseAddr(raw)
		explicit := policy
		explicit.Permissions = Permissions{{HostPort: "relay.test:80", Private: true, Prefix: netip.PrefixFrom(ip, ip.BitLen())}}
		require.False(t, explicit.Allows("http", "relay.test:80", ip), raw)
	}
}
