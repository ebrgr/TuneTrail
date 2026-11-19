// Package imagefetch bounds image-provider requests while allowing self-hosted HTTP(S).
package imagefetch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const MaxBytes = 10 << 20

// validURL accepts HTTP(S) including signed queries, without embedded credentials.
func validURL(u *url.URL) bool {
	return u != nil && len(u.String()) <= 4096 && u.Host != "" && u.User == nil && u.Fragment == "" && (u.Scheme == "http" || u.Scheme == "https")
}

// request applies a deadline and validates every redirect without logging the URL.
func request(ctx context.Context, method, raw string) (*http.Response, error) {
	u, err := url.Parse(raw)
	publicOnly := err == nil && u.Fragment == PublicSourceFragment
	if publicOnly {
		u.Fragment = ""
	}
	if err != nil || !validURL(u) {
		return nil, errors.New("invalid image URL")
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), nil)
	if err != nil {
		return nil, errors.New("invalid image request")
	}
	client := imageClient(publicOnly, nil)
	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, errors.New("image request failed")
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("image server returned HTTP %d", resp.StatusCode)
	}
	return resp, nil
}

// Validate checks an image source with a bounded, cancellable HEAD request.
func Validate(ctx context.Context, raw string, requireImage bool) error {
	resp, err := request(ctx, http.MethodHead, raw)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if requireImage && !strings.HasPrefix(strings.ToLower(resp.Header.Get("Content-Type")), "image/") {
		return errors.New("source is not an image")
	}
	return nil
}

// Download reads no more than MaxBytes; the caller validates supported image data.
func Download(ctx context.Context, raw string) ([]byte, error) {
	resp, err := request(ctx, http.MethodGet, raw)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, MaxBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > MaxBytes {
		return nil, errors.New("image exceeds 10 MiB")
	}
	if len(data) == 0 {
		return nil, errors.New("empty image response")
	}
	return data, nil
}
