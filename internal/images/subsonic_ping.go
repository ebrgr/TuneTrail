package images

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// TestSubsonic validates a connection without saving settings or fetching listening history.
func TestSubsonic(ctx context.Context, baseURL, params string) error {
	q, err := url.ParseQuery(params)
	if err != nil {
		return fmt.Errorf("invalid Subsonic parameters")
	}
	q.Set("f", "json")
	q.Set("v", "1")
	q.Set("c", "koito")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/rest/ping.view?"+q.Encode(), nil)
	if err != nil {
		return fmt.Errorf("invalid Subsonic address")
	}
	client := &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Subsonic connection unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Subsonic returned HTTP %d", resp.StatusCode)
	}
	const limit = 8 << 20
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil || len(body) > limit {
		return fmt.Errorf("invalid Subsonic response size")
	}
	var result struct {
		Response struct {
			Status string `json:"status"`
		} `json:"subsonic-response"`
	}
	if json.Unmarshal(body, &result) != nil || result.Response.Status != "ok" {
		return fmt.Errorf("invalid Subsonic response")
	}
	return nil
}
