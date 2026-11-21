package relay

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ebrgr/TuneTrail/internal/logger"
)

// Diagnostic contains only configured authority and stable policy codes.
type Diagnostic struct {
	Code, Scheme, Host, Port, Category string
}

// Describe extracts the configured authority without copying request data.
func Describe(endpoint, code, category string) Diagnostic {
	d := Diagnostic{Code: code, Category: category}
	u, err := url.Parse(strings.TrimSpace(endpoint))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return d
	}
	d.Scheme, d.Host, d.Port = u.Scheme, strings.ToLower(u.Hostname()), u.Port()
	if d.Port == "" {
		if d.Scheme == "http" {
			d.Port = "80"
		} else {
			d.Port = "443"
		}
	}
	return d
}

// HostPort returns the sanitized configured authority.
func (d Diagnostic) HostPort() string {
	if d.Host == "" {
		return ""
	}
	return net.JoinHostPort(d.Host, d.Port)
}

// ValidationError exposes only a stable class and sanitized diagnostic.
type ValidationError struct {
	Class      Class
	Diagnostic Diagnostic
}

func (e *ValidationError) Error() string { return string(e.Class) }
func (e *ValidationError) Unwrap() error { return ErrUpstream }

type blockedAddress struct{ code, category string }

func (e blockedAddress) Error() string { return e.code }
func (e blockedAddress) Unwrap() error { return errDialPolicy }

// addressCategory labels a range and never returns the address.
func addressCategory(ip netip.Addr) string {
	switch {
	case ip.IsLoopback():
		return "loopback"
	case ip.IsLinkLocalUnicast():
		return "link_local"
	case ip.Is4() && ip.IsPrivate():
		return "rfc1918"
	case ip.IsPrivate():
		return "unique_local"
	case netip.MustParsePrefix("100.64.0.0/10").Contains(ip):
		return "carrier_grade_nat"
	default:
		return "reserved"
	}
}

// transportDiagnostic unwraps policy and DNS errors without retaining their messages.
func transportDiagnostic(endpoint string, class Class, err error) Diagnostic {
	var blocked blockedAddress
	var dns *net.DNSError
	switch {
	case errors.As(err, &blocked):
		return Describe(endpoint, blocked.code, blocked.category)
	case class == TLS:
		return Describe(endpoint, "tls_error", "")
	case errors.As(err, &dns):
		return Describe(endpoint, "dns_resolution_failed", "")
	default:
		return Diagnostic{}
	}
}

var diagnosticLog struct {
	sync.Mutex
	next map[diagnosticKey]time.Time
}

type diagnosticKey struct {
	profile int32
	code    string
}

// LogDiagnostic emits at most one warning per profile and reason per minute.
func LogDiagnostic(ctx context.Context, profile int32, d Diagnostic, now time.Time) {
	if d.Code == "" {
		return
	}
	diagnosticLog.Lock()
	defer diagnosticLog.Unlock()
	key := diagnosticKey{profile, d.Code}
	if now.Before(diagnosticLog.next[key]) {
		return
	}
	if diagnosticLog.next == nil {
		diagnosticLog.next = make(map[diagnosticKey]time.Time)
	}
	if len(diagnosticLog.next) >= 1024 {
		for k, deadline := range diagnosticLog.next {
			if !now.Before(deadline) {
				delete(diagnosticLog.next, k)
			}
		}
		if len(diagnosticLog.next) >= 1024 {
			return
		}
	}
	diagnosticLog.next[key] = now.Add(time.Minute)
	logger.FromContext(ctx).Warn().Int32("profile_id", profile).Str("destination_id", destinationID(profile)).
		Str("reason", d.Code).Str("scheme", d.Scheme).Str("host", d.Host).Str("port", d.Port).Str("range_category", d.Category).
		Msg("Relay destination")
}

func destinationID(profile int32) string { return "relay:" + strconv.FormatInt(int64(profile), 10) }

// Remedy gives an operator instruction without disclosing DNS answers.
func Remedy(code string) string {
	switch code {
	case "http_not_allowed":
		return "Enable allow_http or add an HTTP exception for this destination."
	case "private_network_not_allowed":
		return "Enable allow_private_network or add a private-network exception for this destination."
	case "forbidden_range", "metadata_endpoint", "seed_skipped_resolves_forbidden":
		return "This destination is permanently forbidden."
	case "seed_skipped_non_admin":
		return "Ask an instance administrator to add the required exception."
	case "dns_resolution_failed":
		return "Check the configured hostname and DNS."
	case "redirect_refused":
		return "Configure the final relay URL without redirects."
	case "tls_error":
		return "Check the HTTPS certificate and trust chain."
	case "token_invalid":
		return "Check the relay token."
	case "url_invalid":
		return "Check the relay base URL."
	default:
		return "Check the relay configuration and try again."
	}
}
