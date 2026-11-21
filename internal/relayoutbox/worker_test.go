package relayoutbox

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db/sqlite"
	"github.com/ebrgr/TuneTrail/internal/relay"
	"github.com/stretchr/testify/require"
)

type fixtureStore struct {
	Store
	base              string
	outcomes          map[string]relay.Class
	releases, missing int
}

func (s *fixtureStore) RelayTarget(context.Context, sqlite.RelayBatch) (string, string, relay.Permissions, error) {
	host := strings.TrimPrefix(s.base, "http://")
	return s.base, "fictional-token", relay.Permissions{
		{HostPort: host, HTTP: true},
		{HostPort: host, Private: true, Prefix: netip.MustParsePrefix("127.0.0.1/32")},
	}, nil
}

func (*fixtureStore) RelayPayload(_ context.Context, _ sqlite.RelayBatch, item sqlite.RelayItem) ([]byte, error) {
	if item.ID == "deleted" {
		return nil, sql.ErrNoRows
	}
	return []byte(`{"track_metadata":{"track_name":"` + item.ID + `"}}`), nil
}

func (s *fixtureStore) FailRelaySource(context.Context, sqlite.RelayBatch, sqlite.RelayItem, error) error {
	s.missing++
	return nil
}
func (*fixtureStore) RenewRelayBatch(context.Context, sqlite.RelayBatch, time.Time, ...sqlite.RelayItem) (bool, error) {
	return true, nil
}
func (s *fixtureStore) FinishRelayBatch(ctx context.Context, _ sqlite.RelayBatch, items []sqlite.RelayItem, result relay.Result, _ time.Time, release bool) (bool, error) {
	if release {
		s.releases++
	}
	if result.Class == relay.Success || (result.Class == relay.BadRequest && len(items) == 1) {
		for _, item := range items {
			s.outcomes[item.ID] = result.Class
		}
	}
	return true, nil
}

// TestWorkerBisectsAndSettles isolates one rejected item and still delivers the valid items.
func TestWorkerBisectsAndSettles(t *testing.T) {
	calls := make(chan string, 10)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var batch struct {
			Type    string            `json:"listen_type"`
			Payload []json.RawMessage `json:"payload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
			w.WriteHeader(500)
			return
		}
		calls <- batch.Type
		for _, item := range batch.Payload {
			if strings.Contains(string(item), `"bad"`) {
				w.WriteHeader(400)
				return
			}
		}
		w.WriteHeader(200)
	}))
	defer server.Close()
	store := &fixtureStore{base: server.URL, outcomes: map[string]relay.Class{}}
	process(t.Context(), store, sqlite.RelayBatch{PostBudget: 6, Items: []sqlite.RelayItem{{ID: "a"}, {ID: "bad"}, {ID: "b"}, {ID: "deleted"}}}, time.Now)
	require.Equal(t, map[string]relay.Class{"a": relay.Success, "bad": relay.BadRequest, "b": relay.Success}, store.outcomes)
	require.Equal(t, 1, store.missing)
	require.Equal(t, 1, store.releases)
	require.Len(t, calls, 5)
	require.Equal(t, "import", <-calls)
}

// TestWorkerCancellation releases the claim without settling the canceled request as a failure.
func TestWorkerCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cancel()
		w.WriteHeader(503)
	}))
	defer server.Close()
	store := &fixtureStore{base: server.URL, outcomes: map[string]relay.Class{}}
	process(ctx, store, sqlite.RelayBatch{PostBudget: 2, Items: []sqlite.RelayItem{{ID: "a"}}}, time.Now)
	require.Empty(t, store.outcomes)
	require.Equal(t, 1, store.releases)
}

type failingClaimStore struct {
	Store
	calls               chan time.Time
	claimErr, targetErr error
	paused              chan relay.Class
}

func (s *failingClaimStore) ClaimRelayBatch(_ context.Context, at time.Time) (sqlite.RelayBatch, error) {
	s.calls <- at
	return sqlite.RelayBatch{Items: []sqlite.RelayItem{{ID: "fixture"}}}, s.claimErr
}

func (s *failingClaimStore) RelayTarget(context.Context, sqlite.RelayBatch) (string, string, relay.Permissions, error) {
	return "", "", nil, s.targetErr
}

func (s *failingClaimStore) FinishRelayBatch(_ context.Context, _ sqlite.RelayBatch, _ []sqlite.RelayItem, result relay.Result, _ time.Time, _ bool) (bool, error) {
	if result.Class != "" {
		s.paused <- result.Class
	}
	return true, nil
}

// TestWorkerPacesFailedClaims proves a new attempt needs a scheduler tick.
func TestWorkerPacesFailedClaims(t *testing.T) {
	for _, tc := range []struct {
		name                string
		claimErr, targetErr error
	}{
		{"busy", errors.New("SQLITE_BUSY"), nil},
		{"target read", nil, errors.New("target lookup failed")},
		{"invalid config", nil, sqlite.ErrRelayConfigInvalid},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			done := make(chan struct{})
			ticks := make(chan time.Time)
			now := time.Unix(123, 0)
			store := &failingClaimStore{calls: make(chan time.Time, 16), claimErr: tc.claimErr, targetErr: tc.targetErr, paused: make(chan relay.Class, 4)}
			go func() { defer close(done); poll(ctx, ctx, store, func() time.Time { return now }, ticks) }()
			defer func() { cancel(); <-done }()
			require.Equal(t, now, <-store.calls)
			if errors.Is(tc.targetErr, sqlite.ErrRelayConfigInvalid) {
				require.Equal(t, relay.ConfigInvalid, <-store.paused)
			}
			select {
			case <-store.calls:
				t.Fatal("claim repeated before the scheduler tick")
			case <-time.After(20 * time.Millisecond):
			}
			ticks <- now
			require.Equal(t, now, <-store.calls)
		})
	}
}
