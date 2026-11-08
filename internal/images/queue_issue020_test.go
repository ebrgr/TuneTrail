package images

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/logger"
	"github.com/ebrgr/TuneTrail/queue"
	"github.com/rs/zerolog"
)

// TestProviderFailureDoesNotLogCredentials checks HTTP and transport errors with a fictional key.
func TestProviderFailureDoesNotLogCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusBadGateway) }))
	defer server.Close()
	var logs bytes.Buffer
	l := zerolog.New(&logs).Level(zerolog.DebugLevel)
	ctx := logger.NewContext(&l)
	c := &LastFMClient{baseUrl: server.URL, apiKey: "fictional-api-secret", requestQueue: queue.NewRequestQueue(100, 100)}
	defer c.Shutdown()
	var output map[string]any
	if err := c.getEntity(ctx, url.Values{"method": {"artist.getInfo"}}, &output); err == nil {
		t.Fatal("expected upstream error")
	}
	// A closed local listener deterministically returns a transport error containing the URL internally.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := ln.Addr().String()
	_ = ln.Close()
	c.baseUrl = "http://" + address
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	if err := c.getEntity(ctx, url.Values{"method": {"artist.getInfo"}}, &output); err == nil {
		t.Fatal("expected transport error")
	}
	if bytes.Contains(logs.Bytes(), []byte("fictional-api-secret")) {
		t.Fatal("credential appeared in logs")
	}
}
