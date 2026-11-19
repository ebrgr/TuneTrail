package imagefetch

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"time"
)

// PublicSourceFragment persists untrusted image provenance without changing the schema.
// Fragments are never sent over HTTP; keeping this marker in exports preserves the restriction.
const PublicSourceFragment = "tunetrail-public-only"

// PublicSource marks an imported URL for public-network-only fetching, including after restart.
func PublicSource(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	u.Fragment = ""
	if !validURL(u) {
		return ""
	}
	u.Fragment = PublicSourceFragment
	return u.String()
}

// publicAddress rejects local, private, unspecified and transition addresses that can embed IPv4.
func publicAddress(ip netip.Addr) bool {
	if !ip.IsValid() || ip.Zone() != "" {
		return false
	}
	ip = ip.Unmap()
	if ip.Is6() && !publicIPv6.Contains(ip) {
		return false
	}
	if !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
		return false
	}
	for _, p := range blockedPublicPrefixes {
		if p.Contains(ip) {
			return false
		}
	}
	return true
}

var publicIPv6 = netip.MustParsePrefix("2000::/3")

var blockedPublicPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"), netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("198.18.0.0/15"), netip.MustParsePrefix("240.0.0.0/4"), netip.MustParsePrefix("2001:db8::/32"), netip.MustParsePrefix("::/96"),
	netip.MustParsePrefix("64:ff9b::/96"), netip.MustParsePrefix("64:ff9b:1::/48"),
	netip.MustParsePrefix("2002::/16"), netip.MustParsePrefix("2001::/32"),
}

type lookupIPs func(context.Context, string, string) ([]netip.Addr, error)
type dialIP func(context.Context, string, string) (net.Conn, error)

// publicTransport resolves once, dials a validated literal IP and verifies the actual peer.
// Proxies are disabled so environment proxy configuration cannot bypass destination checks.
func publicTransport(lookup lookupIPs, dial dialIP) *http.Transport {
	if lookup == nil {
		lookup = net.DefaultResolver.LookupNetIP
	}
	if dial == nil {
		dial = (&net.Dialer{Timeout: 5 * time.Second}).DialContext
	}
	return &http.Transport{DisableKeepAlives: true, TLSHandshakeTimeout: 5 * time.Second, ResponseHeaderTimeout: 5 * time.Second, MaxResponseHeaderBytes: 64 << 10,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}
			ips, err := lookup(ctx, "ip", host)
			if err != nil {
				return nil, err
			}
			if len(ips) == 0 {
				return nil, errors.New("image host has no addresses")
			}
			for _, ip := range ips {
				if !publicAddress(ip) {
					return nil, errors.New("image host is not public")
				}
			}
			var last error
			for _, ip := range ips {
				conn, err := dial(ctx, network, net.JoinHostPort(ip.Unmap().String(), port))
				if err != nil {
					last = err
					continue
				}
				peer, ok := conn.RemoteAddr().(*net.TCPAddr)
				if !ok || peer.Zone != "" {
					// A rejected peer is never returned; close errors cannot make it eligible.
					conn.Close()
					return nil, errors.New("invalid image peer")
				}
				actual, ok := netip.AddrFromSlice(peer.IP)
				if !ok || !publicAddress(actual) || actual.Unmap() != ip.Unmap() {
					// Reject regardless of whether closing the unusable connection succeeds.
					conn.Close()
					return nil, errors.New("image peer is not the validated public address")
				}
				return conn, nil
			}
			return nil, last
		},
	}
}

// imageClient bounds every hop; public sources retain the restricted transport across redirects.
func imageClient(publicOnly bool, transport http.RoundTripper) *http.Client {
	if publicOnly && transport == nil {
		transport = publicTransport(nil, nil)
	}
	return &http.Client{Transport: transport, Timeout: 10 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 4 || !validURL(req.URL) {
			return errors.New("invalid image redirect")
		}
		return nil
	}}
}
