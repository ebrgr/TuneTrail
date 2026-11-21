package relay

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// Class identifies an attempt outcome without exposing the request or response.
type Class string

const (
	Success       Class = "success"
	BadRequest    Class = "bad_request"
	Auth          Class = "auth"
	Redirect      Class = "redirect"
	Route         Class = "route"
	RateLimited   Class = "rate_limited"
	Server        Class = "server"
	Transient     Class = "transient"
	TLS           Class = "tls"
	PolicyBlocked Class = "policy_blocked"
	ConfigInvalid Class = "config_invalid"
)

// Result contains only sanitized data for the delivery scheduler.
type Result struct {
	Status        int
	Class         Class
	ServerWait    time.Duration
	RemainingZero bool
	Diagnostic    Diagnostic
}

// Deliverer makes one bounded delivery attempt. It is safe for concurrent use.
type Deliverer struct {
	https *http.Client
	http  *http.Client
	now   func() time.Time
}

// NewDeliverer builds direct transports with an optional operator allowlist and clock.
func NewDeliverer(allow Allowlist, now func() time.Time) *Deliverer {
	return newDeliverer(allow, now, nil)
}

// newDeliverer also supports an isolated DNS resolver for transport tests.
func newDeliverer(allow Allowlist, now func() time.Time, resolver *net.Resolver) *Deliverer {
	if allow == nil {
		allow = Permissions(nil)
	}
	if now == nil {
		now = time.Now
	}
	client := func(plainHTTP bool) *http.Client {
		return &http.Client{
			Transport: newDeliveryTransport(allow, resolver, plainHTTP),
			Timeout:   10 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	}
	return &Deliverer{https: client(false), http: client(true), now: now}
}

// Deliver posts to the full submission endpoint and reports only sanitized metadata.
func (d *Deliverer) Deliver(ctx context.Context, endpoint, token string, body []byte) Result {
	resp, class, diagnostic := d.request(ctx, http.MethodPost, endpoint, token, body)
	if resp == nil {
		return Result{Class: class, Diagnostic: diagnostic}
	}
	result := Result{
		Status:        resp.StatusCode,
		Class:         statusClass(resp.StatusCode),
		ServerWait:    serverWait(resp.Header, d.now()),
		RemainingZero: strings.TrimSpace(resp.Header.Get("X-RateLimit-Remaining")) == "0",
	}
	if result.Class == Redirect {
		result.Diagnostic = Describe(endpoint, "redirect_refused", "")
	}
	if result.Class == Auth {
		result.Diagnostic = Describe(endpoint, "token_invalid", "")
	}
	// The headers settle the attempt. A broken or oversized body cannot undo a 2xx.
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, MaxResponse))
	_ = resp.Body.Close() // A close error does not change the HTTP outcome.
	return result
}

// Test validates a credential with the same dialer and permissions as delivery.
func (d *Deliverer) Test(ctx context.Context, base, token string) (string, error) {
	base, token, err := Normalize(base, token)
	if err != nil {
		return "", err
	}
	resp, class, diagnostic := d.request(ctx, http.MethodGet, base+"/validate-token", token, nil)
	if resp == nil {
		return "", &ValidationError{Class: class, Diagnostic: diagnostic}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		class := statusClass(resp.StatusCode)
		code := ""
		if class == Redirect {
			code = "redirect_refused"
		}
		if class == Auth {
			code = "token_invalid"
		}
		return "", &ValidationError{Class: class, Diagnostic: Describe(base, code, "")}
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, MaxResponse+1))
	if err != nil || len(data) > MaxResponse {
		return "", ErrUpstream
	}
	var result struct {
		Valid    bool   `json:"valid"`
		Username string `json:"user_name"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", ErrUpstream
	}
	if !result.Valid {
		return "", &ValidationError{Class: Auth, Diagnostic: Describe(base, "token_invalid", "")}
	}
	return result.Username, nil
}

// request builds a bounded request without returning raw transport errors.
func (d *Deliverer) request(ctx context.Context, method, endpoint, token string, body []byte) (*http.Response, Class, Diagnostic) {
	_, token, err := Normalize(endpoint, token)
	if err != nil {
		return nil, PolicyBlocked, Describe(endpoint, "url_invalid", "")
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimSpace(endpoint), bytes.NewReader(body))
	if err != nil {
		return nil, PolicyBlocked, Describe(endpoint, "url_invalid", "")
	}
	authorization := "Token " + token
	if !validAuthorization(authorization) {
		return nil, PolicyBlocked, Describe(endpoint, "token_invalid", "")
	}
	req.Header.Set("Authorization", authorization)
	req.Header.Set("Accept", "application/json")
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
	}

	client := d.https
	if req.URL.Scheme == "http" {
		client = d.http
	}
	resp, err := client.Do(req)
	if err != nil {
		class := deliveryErrorClass(err)
		return nil, class, transportDiagnostic(endpoint, class, err)
	}
	return resp, "", Diagnostic{}
}

// ValidStoredTarget checks local configuration without making a network request.
func ValidStoredTarget(base, token string) bool {
	_, normalizedToken, err := Normalize(base, token)
	return err == nil && validAuthorization("Token "+normalizedToken)
}

// validAuthorization rejects control bytes that net/http cannot send in a header value.
func validAuthorization(value string) bool {
	for i := range len(value) {
		b := value[i]
		if (b < 0x20 && b != '\t') || b == 0x7f {
			return false
		}
	}
	return true
}

// statusClass maps the HTTP status to a scheduling outcome.
func statusClass(status int) Class {
	switch {
	case status >= 200 && status < 300:
		return Success
	case status >= 300 && status < 400:
		return Redirect
	case status == http.StatusBadRequest || status == http.StatusRequestEntityTooLarge:
		return BadRequest
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return Auth
	case status == http.StatusTooManyRequests:
		return RateLimited
	case status == http.StatusRequestTimeout:
		return Transient
	case status >= 400 && status < 500:
		return Route
	case status >= 500 && status < 600:
		return Server
	default:
		return Transient
	}
}

// deliveryErrorClass discards raw errors, which can contain endpoint details.
func deliveryErrorClass(err error) Class {
	var cert *tls.CertificateVerificationError
	var hostname x509.HostnameError
	var authority x509.UnknownAuthorityError
	var invalid x509.CertificateInvalidError
	var record tls.RecordHeaderError
	var alert tls.AlertError
	var op *net.OpError
	switch {
	case errors.Is(err, errDialPolicy):
		return PolicyBlocked
	case errors.As(err, &cert), errors.As(err, &hostname), errors.As(err, &authority),
		errors.As(err, &invalid), errors.As(err, &record), errors.As(err, &alert):
		return TLS
	case errors.As(err, &op) && op.Op == "remote error":
		return TLS
	default:
		return Transient
	}
}

const maxServerWait = 30 * 24 * time.Hour

// serverWait uses the longest valid delay across the two server hints.
func serverWait(headers http.Header, now time.Time) time.Duration {
	retryAfter := strings.TrimSpace(headers.Get("Retry-After"))
	retry, numeric := waitSeconds(retryAfter)
	if !numeric {
		if at, err := http.ParseTime(retryAfter); err == nil {
			retry = min(max(at.Sub(now), 0), maxServerWait)
		}
	}
	reset, _ := waitSeconds(strings.TrimSpace(headers.Get("X-RateLimit-Reset-In"))) // Invalid hints carry zero wait.
	return max(retry, reset)
}

// waitSeconds accepts decimal digits and saturates before integer conversion or multiplication.
func waitSeconds(value string) (time.Duration, bool) {
	if value == "" {
		return 0, false
	}
	const limit = int64(maxServerWait / time.Second)
	var seconds int64
	for _, digit := range value {
		if digit < '0' || digit > '9' {
			return 0, false
		}
		seconds = min(seconds*10+int64(digit-'0'), limit)
	}
	return time.Duration(seconds) * time.Second, true
}
