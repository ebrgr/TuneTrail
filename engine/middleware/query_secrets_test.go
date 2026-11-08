package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ebrgr/TuneTrail/internal/logger"
	"github.com/rs/zerolog"
)

// TestRequestLoggerNeverRecordsQuerySecrets covers access and downstream context logs.
func TestRequestLoggerNeverRecordsQuerySecrets(t *testing.T) {
	var out bytes.Buffer
	log := zerolog.New(&out).Level(zerolog.DebugLevel)
	handler := Logger(&log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.FromContext(r.Context()).Info().Msg("handler")
		w.WriteHeader(http.StatusNoContent)
	}))
	for _, query := range []string{
		"token=fictional-token-marker&csrf_token=fictional-csrf-marker&next=%2Fadmin%3Fapi_key%3Dfictional-next-marker&ordinary=fictional-other-marker&ordinary=fictional-second-marker",
		"fictional-key-marker=value&bad=%zz&safe=fictional-malformed-marker",
	} {
		req := httptest.NewRequest("POST", "/apis/web/v1/login", nil)
		req.URL.RawQuery = query
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d", rec.Code)
		}
	}
	for _, marker := range []string{"fictional-token-marker", "fictional-csrf-marker", "fictional-next-marker", "fictional-other-marker", "fictional-second-marker", "fictional-key-marker", "fictional-malformed-marker"} {
		if strings.Contains(out.String(), marker) {
			t.Fatalf("query data was logged: %s", marker)
		}
	}
	if !strings.Contains(out.String(), "POST /apis/web/v1/login") || !strings.Contains(out.String(), "204") {
		t.Fatal("access log lost method, path, or status")
	}
}
