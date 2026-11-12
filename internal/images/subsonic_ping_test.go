package images

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestSubsonicProbe exercises success, redirect, status, response size and JSON validation.
func TestSubsonicProbe(t *testing.T) {
	for _, tc := range []struct {
		name     string
		status   int
		body     string
		redirect bool
		wantErr  bool
	}{
		{"ok", 200, `{"subsonic-response":{"status":"ok"}}`, false, false},
		{"remote error", 200, `{"subsonic-response":{"status":"failed"}}`, false, true},
		{"invalid JSON", 200, `not-json`, false, true},
		{"trailing JSON", 200, `{"subsonic-response":{"status":"ok"}} {}`, false, true},
		{"http error", 500, `error`, false, true},
		{"redirect", 302, "", true, true},
		{"oversized", 200, strings.Repeat("x", (8<<20)+1), false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/base/rest/ping.view" || r.URL.Query().Get("u") != "test" {
					t.Errorf("invalid request path or parameters")
				}
				if tc.redirect {
					w.Header().Set("Location", "/other")
				}
				w.WriteHeader(tc.status)
				fmt.Fprint(w, tc.body)
			}))
			defer srv.Close()
			err := TestSubsonic(context.Background(), srv.URL+"/base/", "u=test&p=value")
			if (err != nil) != tc.wantErr {
				t.Fatalf("error = %v", err)
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := TestSubsonic(ctx, "http://127.0.0.1:1", "u=test&p=value"); err == nil {
		t.Fatal("cancelled context accepted")
	}
}

// TestSubsonicProbeCancelsInFlight ensures cancellation interrupts an active response.
func TestSubsonicProbeCancelsInFlight(t *testing.T) {
	entered := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(entered); <-r.Context().Done() }))
	defer srv.Close()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- TestSubsonic(ctx, srv.URL, "u=test&p=value") }()
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		cancel()
		t.Fatal("request did not start")
	}
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("canceled request succeeded")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("request did not stop")
	}
}
