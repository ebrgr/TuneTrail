package engine

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestListenBrainzResponsesDeclareJSON verifies headers committed on success and storage failure.
func TestListenBrainzResponsesDeclareJSON(t *testing.T) {
	a := newProfileTestApp(t)
	path := "/apis/listenbrainz/1/submit-listens"
	stamp := time.Now().Add(-time.Hour).Unix()
	success := a.request("POST", path, listenBody("single", "Header Success", stamp), "first-key", false)
	require.Equal(t, http.StatusOK, success.Code)
	require.Equal(t, "application/json", success.Result().Header.Get("Content-Type"))
	require.NoError(t, a.store.Exec(`CREATE TRIGGER issue021_fail_listen BEFORE INSERT ON ingest_events BEGIN SELECT RAISE(ABORT,'injected'); END`))
	failure := a.request("POST", path, listenBody("single", "Header Failure", stamp+1), "first-key", false)
	require.Equal(t, http.StatusInternalServerError, failure.Code)
	require.Equal(t, "application/json", failure.Result().Header.Get("Content-Type"))
}
