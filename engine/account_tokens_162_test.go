package engine

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"testing"

	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestAccountTokenRoutes162 uses the production router to check the public entry point and JSON surfaces.
func TestAccountTokenRoutes162(t *testing.T) {
	ctx := context.Background()
	a := newProfileTestApp(t)
	p, err := a.store.GetListeningProfile(ctx, "second")
	require.NoError(t, err)
	token, err := admin.IssueProfileAccountToken(ctx, a.store, uuid.MustParse(a.cookie), p.ID)
	require.NoError(t, err)
	raw, err := base64.RawURLEncoding.DecodeString(token)
	require.NoError(t, err)
	digest := sha256.Sum256(raw)
	root, err := a.store.GetAdminUser(ctx)
	require.NoError(t, err)
	for _, path := range []string{"/apis/web/v1/profiles", "/apis/web/v1/profiles/second/", "/apis/web/v1/profiles/second/management"} {
		rec := a.request("GET", path, "", "", true)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		for _, secret := range []string{token, hex.EncodeToString(digest[:]), base64.StdEncoding.EncodeToString(digest[:]), string(root.Password), base64.StdEncoding.EncodeToString(root.Password), "test-password"} {
			require.NotContains(t, rec.Body.String(), secret)
		}
	}
	rec := a.request("GET", "/admin/redeem", "", "", false)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "no-referrer", rec.Header().Get("Referrer-Policy"))
	require.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
}
