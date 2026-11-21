package relay

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"syscall"
	"time"
)

// Allowlist grants exceptions by endpoint host:port or resolved IP/CIDR.
// Implementations must be safe for concurrent calls.
type Allowlist interface {
	Allows(scheme, hostPort string, ip netip.Addr) bool
}

var errDialPolicy = errors.New("relay destination blocked by policy")

var reservedPrefixes = [...]netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("10.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("127.0.0.0/8"),
	netip.MustParsePrefix("169.254.0.0/16"),
	netip.MustParsePrefix("172.16.0.0/12"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("192.168.0.0/16"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("2001::/32"),
	netip.MustParsePrefix("::/96"),
	netip.MustParsePrefix("2001:db8::/32"),
	netip.MustParsePrefix("fc00::/7"),
	netip.MustParsePrefix("fe80::/10"),
	netip.MustParsePrefix("fec0::/10"),
}

// Block translated addresses outright. Public relays reached only through DNS64
// are unavailable until the embedded IPv4 can be checked safely.
var forbiddenPrefixes = [...]netip.Prefix{
	netip.MustParsePrefix("169.254.169.254/32"),
	netip.MustParsePrefix("169.254.170.2/32"),
	netip.MustParsePrefix("169.254.170.23/32"),
	netip.MustParsePrefix("fd00:ec2::23/128"),
	netip.MustParsePrefix("100.100.100.200/32"),
	netip.MustParsePrefix("192.0.0.192/32"),
	netip.MustParsePrefix("fd00:ec2::254/128"),
	netip.MustParsePrefix("64:ff9b::/96"),
	netip.MustParsePrefix("64:ff9b:1::/48"),
	netip.MustParsePrefix("2002::/16"),
}

// dialAllowed checks the address the kernel will actually dial, including mapped IPv4.
func dialAllowed(hostPort, sockaddr string, allow Allowlist, plainHTTP bool) bool {
	return dialPolicy(hostPort, sockaddr, allow, plainHTTP) == nil
}

// dialPolicy classifies a denial using only the category of the resolved address.
func dialPolicy(hostPort, sockaddr string, allow Allowlist, plainHTTP bool) error {
	address, err := netip.ParseAddrPort(sockaddr)
	if err != nil {
		return blockedAddress{"forbidden_range", "reserved"}
	}
	ip := address.Addr().Unmap().WithZone("")
	if IsForbiddenAddress(ip) {
		for _, prefix := range forbiddenPrefixes {
			if !prefix.Addr().Is4() && prefix.Bits() < 128 {
				continue
			}
			if prefix.Contains(ip) {
				return blockedAddress{"metadata_endpoint", "metadata"}
			}
		}
		return blockedAddress{"forbidden_range", addressCategory(ip)}
	}
	scheme := "https"
	if plainHTTP {
		scheme = "http"
	}
	if allow != nil {
		host, port, err := net.SplitHostPort(hostPort)
		if err == nil && port != "" {
			canonical := net.JoinHostPort(strings.ToLower(strings.TrimSuffix(host, ".")), port)
			if allow.Allows(scheme, canonical, ip) {
				return nil
			}
			if IsReservedAddress(ip) && !allow.Allows("https", canonical, ip) {
				return blockedAddress{"private_network_not_allowed", addressCategory(ip)}
			}
		}
	}
	if plainHTTP {
		return blockedAddress{"http_not_allowed", addressCategory(ip)}
	}
	if IsReservedAddress(ip) {
		return blockedAddress{"private_network_not_allowed", addressCategory(ip)}
	}
	return nil
}

// IsForbiddenAddress reports addresses that operator exceptions cannot authorize.
func IsForbiddenAddress(ip netip.Addr) bool {
	ip = ip.Unmap().WithZone("")
	if !ip.IsValid() || ip.IsUnspecified() || ip.IsMulticast() || ip == netip.AddrFrom4([4]byte{255, 255, 255, 255}) {
		return true
	}
	for _, prefix := range forbiddenPrefixes {
		if prefix.Contains(ip) {
			return true
		}
	}
	return false
}

// IsReservedAddress reports addresses that require an operator exception.
func IsReservedAddress(ip netip.Addr) bool {
	ip = ip.Unmap().WithZone("")
	if !ip.IsGlobalUnicast() {
		return true
	}
	for _, prefix := range reservedPrefixes {
		if prefix.Contains(ip) {
			return true
		}
	}
	return false
}

// IsReservedPrefix reports whether a whole prefix falls under the reserved-network policy.
func IsReservedPrefix(prefix netip.Prefix) bool {
	for _, reserved := range reservedPrefixes {
		if prefix.Bits() >= reserved.Bits() && reserved.Contains(prefix.Masked().Addr()) {
			return true
		}
	}
	return false
}

// ValidExceptionPrefix rejects default routes and networks overlapping forbidden ranges.
func ValidExceptionPrefix(prefix netip.Prefix) bool {
	if !prefix.IsValid() || prefix.Bits() == 0 || prefix.Addr().Is4In6() || IsForbiddenAddress(prefix.Addr()) {
		return false
	}
	for _, blocked := range forbiddenPrefixes {
		if prefix.Overlaps(blocked) {
			return false
		}
	}
	for _, blocked := range [...]netip.Prefix{
		netip.MustParsePrefix("224.0.0.0/4"), netip.MustParsePrefix("ff00::/8"),
		netip.MustParsePrefix("0.0.0.0/32"), netip.MustParsePrefix("::/128"),
		netip.MustParsePrefix("255.255.255.255/32"), netip.MustParsePrefix("::ffff:0:0/96"),
	} {
		if prefix.Overlaps(blocked) {
			return false
		}
	}
	return true
}

// newDeliveryTransport checks each resolved address through Control before connect.
func newDeliveryTransport(allow Allowlist, resolver *net.Resolver, plainHTTP bool) *http.Transport {
	return &http.Transport{
		Proxy:                 nil,
		DisableKeepAlives:     true, // Recheck policy on every attempt.
		TLSClientConfig:       &tls.Config{MinVersion: tls.VersionTLS12},
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 8 * time.Second,
		DialContext: func(ctx context.Context, network, hostPort string) (net.Conn, error) {
			dialer := net.Dialer{
				Timeout:  5 * time.Second,
				Resolver: resolver,
				Control: func(_ string, sockaddr string, _ syscall.RawConn) error {
					return dialPolicy(hostPort, sockaddr, allow, plainHTTP)
				},
			}
			// DialContext can report the first policy error when a later public address is unreachable.
			return dialer.DialContext(ctx, network, hostPort)
		},
	}
}
