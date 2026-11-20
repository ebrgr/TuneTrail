package relay

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type allowFunc func(string, netip.Addr) bool

func (f allowFunc) Allows(hostPort string, ip netip.Addr) bool { return f(hostPort, ip) }

const fixtureToken = "fictional-relay-token"

// TestDeliverHTTPClasses exercises the wire contract and all HTTP outcome classes.
func TestDeliverHTTPClasses(t *testing.T) {
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	cases := map[int]Class{
		200: Success, 204: Success, 302: Redirect, 400: BadRequest, 401: Auth,
		403: Auth, 404: Route, 405: Route, 408: Transient, 410: Route,
		413: BadRequest, 422: Route, 429: RateLimited, 503: Server,
	}
	for status, class := range cases {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			const payload = " {\n \"payload\": []\n}\n"
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/prefix/submit-listens" {
					t.Error("unexpected path or redirect followed")
					w.WriteHeader(http.StatusTeapot)
					return
				}
				body, err := io.ReadAll(r.Body)
				if err != nil || string(body) != payload || r.Method != http.MethodPost {
					t.Error("unexpected submission")
				}
				if r.Header.Get("Authorization") != "Token "+fixtureToken ||
					r.Header.Get("Accept") != "application/json" || r.Header.Get("Content-Type") != "application/json" {
					t.Error("unexpected request headers")
				}
				w.Header().Set("Location", "/other")
				w.Header().Set("Retry-After", now.Add(time.Minute).Format(http.TimeFormat))
				w.Header().Set("X-RateLimit-Reset-In", "8")
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.WriteHeader(status)
			}))
			defer server.Close()
			allow := allowFunc(func(hostPort string, _ netip.Addr) bool {
				return hostPort == strings.TrimPrefix(server.URL, "http://")
			})
			result := NewDeliverer(allow, func() time.Time { return now }).Deliver(
				t.Context(), server.URL+"/prefix/submit-listens", fixtureToken, []byte(payload))
			require.Equal(t, Result{Status: status, Class: class, ServerWait: time.Minute, RemainingZero: true}, result)
		})
	}
}

// TestDeliverSuccessWithUnusableBody checks that a truncated or huge body cannot undo success.
func TestDeliverSuccessWithUnusableBody(t *testing.T) {
	for _, size := range []int{1, 2 * MaxResponse} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Length", fmt.Sprint(2*MaxResponse))
				if _, err := io.WriteString(w, strings.Repeat("x", size)); err != nil && size == 1 {
					t.Error("failed to write fixture")
				}
			}))
			defer server.Close()
			allow := allowFunc(func(_ string, ip netip.Addr) bool { return ip.IsLoopback() })
			require.Equal(t, Result{Status: 200, Class: Success}, NewDeliverer(allow, nil).
				Deliver(t.Context(), server.URL, fixtureToken, nil))
		})
	}
}

// TestDeliverTransportFailures checks classifier output and secrecy.
func TestDeliverTransportFailures(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("request should never reach TLS test handler")
	}))
	defer server.Close()
	allow := allowFunc(func(_ string, ip netip.Addr) bool { return ip.IsLoopback() })
	delivery := NewDeliverer(allow, nil)
	require.Equal(t, TLS, delivery.Deliver(t.Context(), server.URL, fixtureToken, nil).Class)
	oldTLS := httptest.NewUnstartedServer(server.Config.Handler)
	oldTLS.TLS = &tls.Config{MinVersion: tls.VersionTLS10, MaxVersion: tls.VersionTLS11}
	oldTLS.StartTLS()
	defer oldTLS.Close()
	require.Equal(t, TLS, delivery.Deliver(t.Context(), oldTLS.URL, fixtureToken, nil).Class)
	require.Equal(t, PolicyBlocked, NewDeliverer(nil, nil).Deliver(t.Context(), server.URL, fixtureToken, nil).Class)
	require.Equal(t, PolicyBlocked, NewDeliverer(nil, nil).Deliver(t.Context(), "http://192.0.2.1", fixtureToken, nil).Class)

	closed := httptest.NewServer(nil)
	closed.Close()
	require.Equal(t, Transient, delivery.Deliver(t.Context(), closed.URL, fixtureToken, nil).Class)

	ctx, cancel := context.WithDeadline(t.Context(), time.Now().Add(-time.Second))
	defer cancel()
	require.Equal(t, Transient, delivery.Deliver(ctx, server.URL, fixtureToken, nil).Class)

	result := delivery.Deliver(t.Context(), "https://example.test/?secret="+fixtureToken, fixtureToken, []byte("private-body"))
	require.Equal(t, PolicyBlocked, result.Class)
	require.NotContains(t, fmt.Sprintf("%+v", result), fixtureToken)
	require.NotContains(t, fmt.Sprintf("%+v", result), "example.test")
	require.NotContains(t, fmt.Sprintf("%+v", result), "private-body")
}

// TestServerWait checks date parsing, hint precedence and saturation.
func TestServerWait(t *testing.T) {
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name, retry, reset string
		wait               time.Duration
	}{
		{"seconds", "5", "4", 5 * time.Second},
		{"date", now.Add(time.Minute).Format(http.TimeFormat), "", time.Minute},
		{"reset", "7", "9", 9 * time.Second},
		{"invalid", "-1", "1.5", 0},
		{"past", now.Add(-time.Hour).Format(http.TimeFormat), "", 0},
		{"decimal overflow", "9999999999999999999999999", "", maxServerWait},
		{"reset overflow", "", "9999999999999999999999999", maxServerWait},
		{"date cap", now.Add(31 * 24 * time.Hour).Format(http.TimeFormat), "", maxServerWait},
		{"invalid after saturation", "", "999999999999999999999x", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := make(http.Header)
			h.Set("Retry-After", tc.retry)
			h.Set("X-RateLimit-Reset-In", tc.reset)
			require.Equal(t, tc.wait, serverWait(h, now))
		})
	}
}

// TestDialPolicy exercises address normalization and operator exceptions.
func TestDialPolicy(t *testing.T) {
	for _, raw := range []string{
		"127.0.0.1", "::1", "::ffff:127.0.0.1", "0.10.0.1", "10.0.0.1",
		"172.16.0.1", "192.168.0.1", "100.64.0.1", "169.254.0.1", "192.0.0.1",
		"198.18.0.1", "224.0.0.1", "255.255.255.255", "0.0.0.0", "::",
		"fc00::1", "fe80::1%test", "ff02::1", "64:ff9b::a00:1", "2002:a00:1::",
	} {
		require.False(t, dialAllowed("relay.test:443", net.JoinHostPort(raw, "443"), nil, false), raw)
	}
	allowAll := allowFunc(func(string, netip.Addr) bool { return true })
	for _, raw := range []string{
		"169.254.169.254", "::ffff:169.254.169.254", "fd00:ec2::254",
		"169.254.170.2", "169.254.170.23", "fd00:ec2::23", "100.100.100.200", "192.0.0.192",
		"64:ff9b::a9fe:a9fe", "2002:a9fe:a9fe::",
	} {
		require.False(t, dialAllowed("relay.test:443", net.JoinHostPort(raw, "443"), allowAll, true), raw)
	}
	require.True(t, dialAllowed("relay.test:443", "8.8.8.8:443", nil, false))
	require.True(t, dialAllowed("[::1]:80", "[::1]:80", allowAll, true))
}

// TestDeliverRechecksDNS controls the DNS answer between two real HTTP attempts.
func TestDeliverRechecksDNS(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	base, err := url.Parse(server.URL)
	require.NoError(t, err)

	var rebound atomic.Bool
	resolver := &net.Resolver{PreferGo: true}
	resolver.Dial = func(_ context.Context, _, _ string) (net.Conn, error) {
		client, peer := net.Pipe()
		go answerDNS(peer, &rebound)
		return client, nil
	}
	allow := allowFunc(func(_ string, ip netip.Addr) bool { return ip == netip.MustParseAddr("127.0.0.1") })
	delivery := newDeliverer(allow, nil, resolver)
	endpoint := "http://relay.test.:" + base.Port() + "/submit-listens"
	require.Equal(t, Success, delivery.Deliver(t.Context(), endpoint, fixtureToken, nil).Class)
	rebound.Store(true)
	require.Equal(t, PolicyBlocked, delivery.Deliver(t.Context(), endpoint, fixtureToken, nil).Class)
	require.EqualValues(t, 1, requests.Load())
}

// answerDNS is a one-query TCP DNS fixture for the real Go dialer.
func answerDNS(conn net.Conn, rebound *atomic.Bool) {
	defer func() { _ = conn.Close() }() // The resolver may close first.
	var size uint16
	if err := binary.Read(conn, binary.BigEndian, &size); err != nil || size < 17 {
		return
	}
	query := make([]byte, size)
	if _, err := io.ReadFull(conn, query); err != nil {
		return
	}
	// Keep only the question, dropping any EDNS record. The test uses relay.test.
	const questionEnd = 28
	if len(query) < questionEnd {
		return
	}
	reply := query[:questionEnd:questionEnd]
	reply[2], reply[3] = 0x81, 0x80
	clear(reply[6:12])
	if binary.BigEndian.Uint16(reply[24:26]) == 1 {
		reply[7] = 1
		ip := byte(1)
		if rebound.Load() {
			ip = 2
		}
		reply = append(reply, 0xc0, 12, 0, 1, 0, 1, 0, 0, 0, 0, 0, 4, 127, 0, 0, ip)
	}
	packet := binary.BigEndian.AppendUint16(nil, uint16(len(reply)))
	_, _ = conn.Write(append(packet, reply...)) // The resolver reports a missing reply to the test.
}

// TestLegacyRelayGolden pins the v0.47 request through the default environment proxy.
func TestLegacyRelayGolden(t *testing.T) {
	const endpoint = "http://relay.test/prefix/"
	const body = " {\n \"payload\": []\n}\n"
	if os.Getenv("TUNETRAIL_RELAY_GOLDEN_CHILD") == "1" {
		require.NoError(t, (Sender{}).Send(t.Context(), endpoint, fixtureToken, []byte(body)))
		name, err := (Client{}).Test(t.Context(), endpoint, fixtureToken)
		require.NoError(t, err)
		require.Equal(t, "listener", name)
		// An environment proxy would reach the fixture unless the new transport ignores it.
		resolver := &net.Resolver{PreferGo: true, Dial: func(_ context.Context, _, _ string) (net.Conn, error) {
			client, peer := net.Pipe()
			go answerDNS(peer, &atomic.Bool{})
			return client, nil
		}}
		allow := allowFunc(func(hostPort string, _ netip.Addr) bool { return !strings.HasPrefix(hostPort, "relay.test") })
		require.Equal(t, PolicyBlocked, newDeliverer(allow, nil, resolver).
			Deliver(t.Context(), "http://relay.test./prefix/submit-listens", fixtureToken, []byte(body)).Class)
		return
	}
	requests := make(chan string, 2)
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error("failed to read request")
		}
		select {
		case requests <- strings.Join([]string{r.Method, r.RequestURI, r.Header.Get("Authorization"),
			r.Header.Get("Accept"), r.Header.Get("Content-Type"), string(data)}, "|"):
		default:
			t.Error("unexpected request")
		}
		_, _ = io.WriteString(w, `{"valid":true,"user_name":"listener"}`) // Child validates the response.
	}))
	defer proxy.Close()
	child := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestLegacyRelayGolden$")
	child.Env = append(os.Environ(), "TUNETRAIL_RELAY_GOLDEN_CHILD=1", "HTTP_PROXY="+proxy.URL, "NO_PROXY=", "REQUEST_METHOD=")
	output, err := child.CombinedOutput()
	require.NoError(t, err, "%s", output)
	require.Len(t, requests, 2)
	require.Equal(t, "POST|"+endpoint+"submit-listens|Token "+fixtureToken+"|application/json|application/json|"+body, <-requests)
	require.Equal(t, "GET|"+endpoint+"validate-token|Token "+fixtureToken+"|application/json||", <-requests)
}
