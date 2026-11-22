package adminui_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ebrgr/TuneTrail/internal/adminui"
	"github.com/ebrgr/TuneTrail/internal/db/sqlite"
	"github.com/stretchr/testify/require"
)

// TestRedeemRefusedOriginShowsExpiredPage125 covers the real-browser failure: under
// "Referrer-Policy: no-referrer" a form POST carries "Origin: null", which the same-origin policy
// refuses. On the redemption form the refusal must read as an expired page, never as a permission
// problem, and the form must come back usable (sign-in keeps its generic answer). The page itself must not request that policy any more.
func TestRedeemRefusedOriginShowsExpiredPage125(t *testing.T) {
	ctx := context.Background()
	s, err := sqlite.NewInMemory()
	require.NoError(t, err)
	defer s.Close(ctx)
	h, err := adminui.New(s, adminui.Options{DisableRateLimit: true, RequireSubmissionToken: true})
	require.NoError(t, err)

	page := httptest.NewRecorder()
	h.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/admin/redeem", nil))
	require.Equal(t, http.StatusOK, page.Code)
	require.NotEqual(t, "no-referrer", page.Header().Get("Referrer-Policy"), "browsers send Origin: null under no-referrer")

	for _, path := range []string{"/admin/redeem"} {
		form := url.Values{"username": {"maria"}, "account_token": {"x"}, "password": {"genius12345"}, "confirm_password": {"genius12345"}}
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Origin", "null")
		req.Header.Set("Sec-Fetch-Site", "same-origin")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		require.Equal(t, http.StatusForbidden, rec.Code, path)
		body := rec.Body.String()
		require.Contains(t, body, "This page expired or was opened in another tab. Try again.", path)
		require.NotContains(t, body, "does not have permission", path)
		require.NotContains(t, body, "genius12345", path)
		require.Contains(t, body, `name="csrf_token"`, path)
	}
}
