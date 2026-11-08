// Package relay uses one bounded transport policy for validation and submissions.
package relay

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
)

var ErrUpstream = errors.New("relay request failed")

const MaxResponse = 64 << 10

// Normalize permits self-hosted HTTP(S) destinations while rejecting credentials in URLs.
func Normalize(raw, token string) (string, string, error) {
	raw = strings.TrimRight(strings.TrimSpace(raw), "/")
	token = strings.TrimSpace(token)
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil || (u.Scheme != "http" && u.Scheme != "https") || u.RawQuery != "" || u.ForceQuery || strings.Contains(raw, "#") || u.Fragment != "" || token == "" || strings.ContainsAny(token, "\r\n") || len(raw) > 2048 || len(token) > 4096 {
		return "", "", db.ErrInvalidInput
	}
	return raw, token, nil
}

// Client disables redirects for every destination, including local servers.
type Client struct {
	Transport http.RoundTripper
	Timeout   time.Duration
}

// request bounds the response, closes it before returning, and never reports URL/token/body in errors.
func (c Client) request(ctx context.Context, method, endpoint, token string, payload []byte) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(payload))
	if err != nil {
		return 0, nil, db.ErrInvalidInput
	}
	req.Header.Set("Authorization", "Token "+token)
	req.Header.Set("Accept", "application/json")
	if method == "POST" {
		req.Header.Set("Content-Type", "application/json")
	}
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	client := http.Client{Transport: c.Transport, Timeout: timeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, ErrUpstream
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, MaxResponse+1))
	if err != nil || len(data) > MaxResponse {
		return 0, nil, ErrUpstream
	}
	return resp.StatusCode, data, nil
}

// Test validates credentials without sending a listen or changing saved configuration.
func (c Client) Test(ctx context.Context, raw, token string) (string, error) {
	raw, token, err := Normalize(raw, token)
	if err != nil {
		return "", err
	}
	status, data, err := c.request(ctx, "GET", raw+"/validate-token", token, nil)
	if err != nil || status != 200 {
		return "", ErrUpstream
	}
	var result struct {
		Valid    bool   `json:"valid"`
		UserName string `json:"user_name"`
	}
	if json.Unmarshal(data, &result) != nil {
		return "", ErrUpstream
	}
	if !result.Valid {
		return "", db.ErrInvalidInput
	}
	return result.UserName, nil
}

// Sender retries server errors with a fresh request and a bounded, cancellable lifetime.
type Sender struct {
	Client                                  Client
	InitialBackoff, MaxBackoff, MaxDuration time.Duration
}

// Send is best-effort forwarding; cancellation and failure do not roll back the local listen.
func (s Sender) Send(ctx context.Context, raw, token string, payload []byte) error {
	raw, token, err := Normalize(raw, token)
	if err != nil {
		return err
	}
	duration := s.MaxDuration
	if duration <= 0 {
		duration = 3 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()
	delay := s.InitialBackoff
	if delay <= 0 {
		delay = 5 * time.Second
	}
	maxDelay := s.MaxBackoff
	if maxDelay <= 0 {
		maxDelay = 40 * time.Second
	}
	for {
		status, _, err := s.Client.request(ctx, "POST", raw+"/submit-listens", token, payload)
		if err != nil {
			return err
		}
		if status >= 200 && status < 300 {
			return nil
		}
		if status < 500 {
			return fmt.Errorf("%w: status %d", ErrUpstream, status)
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ErrUpstream
		case <-timer.C:
		}
		delay *= 2
		if delay > maxDelay {
			delay = maxDelay
		}
	}
}
