package relay

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/stretchr/testify/require"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func response(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}
func TestValidationAndBoundedTransport(t *testing.T) {
	for _, raw := range []string{"http://user:secret@localhost", "https://example.com/?token=secret", "file:///tmp/x", "http://localhost?", "http://localhost#"} {
		_, _, err := Normalize(raw, "token")
		require.ErrorIs(t, err, db.ErrInvalidInput)
	}
	calls := 0
	c := Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		require.Equal(t, "/validate-token", r.URL.Path)
		require.Equal(t, "Token fake-token", r.Header.Get("Authorization"))
		return response(200, `{"valid":true,"user_name":"listener"}`), nil
	})}
	user, err := c.Test(context.Background(), "http://localhost:1234/", "fake-token")
	require.NoError(t, err)
	require.Equal(t, "listener", user)
	require.Equal(t, 1, calls)
	c.Transport = transportFunc(func(*http.Request) (*http.Response, error) {
		r := response(302, "")
		r.Header.Set("Location", "http://other/secret")
		return r, nil
	})
	_, err = c.Test(context.Background(), "http://localhost", "fake-token")
	require.ErrorIs(t, err, ErrUpstream)
	c.Transport = transportFunc(func(*http.Request) (*http.Response, error) {
		return response(200, strings.Repeat("x", MaxResponse+1)), nil
	})
	_, err = c.Test(context.Background(), "http://localhost", "fake-token")
	require.ErrorIs(t, err, ErrUpstream)
	c.Transport = transportFunc(func(*http.Request) (*http.Response, error) { return response(200, `{"valid":false}`), nil })
	_, err = c.Test(context.Background(), "http://localhost", "fake-token")
	require.ErrorIs(t, err, db.ErrInvalidInput)
	c.Transport = transportFunc(func(r *http.Request) (*http.Response, error) { <-r.Context().Done(); return nil, r.Context().Err() })
	c.Timeout = time.Millisecond
	_, err = c.Test(context.Background(), "http://localhost", "fake-token")
	require.ErrorIs(t, err, ErrUpstream)
}
func TestSenderRetriesWithCompletePayload(t *testing.T) {
	var bodies []string
	c := Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		b, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(b))
		if len(bodies) == 1 {
			return response(503, ""), nil
		}
		return response(200, ""), nil
	})}
	s := Sender{Client: c, InitialBackoff: time.Millisecond, MaxDuration: time.Second}
	require.NoError(t, s.Send(context.Background(), "http://localhost", "token", []byte(`{"listen":1}`)))
	require.Equal(t, []string{`{"listen":1}`, `{"listen":1}`}, bodies)
}

// TestSenderDoesNotRetry429AndStopsDuringBackoff checks bounded best-effort semantics.
func TestSenderDoesNotRetry429AndStopsDuringBackoff(t *testing.T) {
	calls := 0
	c := Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		return response(http.StatusTooManyRequests, ""), nil
	})}
	err := (Sender{Client: c, InitialBackoff: time.Hour, MaxDuration: time.Second}).Send(context.Background(), "http://localhost", "fictional", []byte(`{}`))
	require.ErrorIs(t, err, ErrUpstream)
	require.Equal(t, 1, calls)
	ctx, cancel := context.WithCancel(context.Background())
	attempted := make(chan struct{})
	c.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
		close(attempted)
		return response(http.StatusServiceUnavailable, ""), nil
	})
	done := make(chan error, 1)
	go func() {
		done <- (Sender{Client: c, InitialBackoff: time.Hour, MaxDuration: time.Hour}).Send(ctx, "http://localhost", "fictional", []byte(`{}`))
	}()
	<-attempted
	cancel()
	select {
	case err := <-done:
		require.ErrorIs(t, err, ErrUpstream)
	case <-time.After(time.Second):
		t.Fatal("relay backoff did not cancel")
	}
}
