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

// TestDiagnosticHostSafety rejects control characters, Unicode and scoped addresses from logs.
func TestDiagnosticHostSafety(t *testing.T) {
	for _, endpoint := range []string{"https://bad<>.test", "https://evil\u2028.test", "https://evil\u202e.test", "https://[fe80::1%25eth0]:443", "https://bücher.example"} {
		require.Equal(t, "invalid", Describe(endpoint, "url_invalid", "").Host, endpoint)
	}
	host, err := CanonicalHost("Bücher.example")
	require.NoError(t, err)
	require.Equal(t, "xn--bcher-kva.example", host)
	require.Equal(t, host, Describe("https://"+host, "", "").Host)
	require.Equal(t, "2001:db8::1", Describe("https://[2001:db8::1]", "", "").Host)
}

// TestPrivateDiagnosticsSharePublicClass prevents a non-root network scanning oracle.
func TestPrivateDiagnosticsSharePublicClass(t *testing.T) {
	for _, class := range []Class{Auth, Redirect, Transient, PolicyBlocked, TLS, Route, Server} {
		err := &ValidationError{Class: class, Diagnostic: Diagnostic{Code: "private_network_not_allowed"}}
		require.Equal(t, "network", err.PublicClass())
	}
	require.Equal(t, "policy", (&ValidationError{Class: Auth, PublicPeer: true}).PublicClass())
	require.Equal(t, "network", (&ValidationError{Class: Redirect, PublicPeer: true}).PublicClass())
}

// TestComposeRelayHostname keeps service aliases usable without relaxing diagnostic sanitization.
func TestComposeRelayHostname(t *testing.T) {
	key := "multi_scrobbler:9078"
	ip := netip.MustParseAddr("10.1.2.3")
	policy := NetworkPolicy{Permissions: Permissions{
		{HostPort: key, HTTP: true},
		{HostPort: key, Private: true, Prefix: netip.PrefixFrom(ip, 32)},
	}}
	require.True(t, policy.Allows("http", key, ip))
	require.True(t, dialAllowed("Multi_Scrobbler:9078", "10.1.2.3:9078", policy, true))
	require.Equal(t, key, Describe("http://Multi_Scrobbler:9078/x", "", "").HostPort())
	for _, host := range []string{"a\u202eb", "a\u2028b", "a\x00b", "a\x7fb", "a b", "a/b", "a<>b", `a"b`, "fe80::1%eth0"} {
		_, err := CanonicalHost(host)
		require.Error(t, err, "%q", host)
	}
}
