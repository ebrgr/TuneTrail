package mbz

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/queue"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestResolverAndBudgetShareOneRate proves that requests made through the resolver methods and
// through Do draw from the same queue: six requests at 20 per second cannot finish faster than
// five intervals, whereas two independent queues of three would finish in about two.
func TestResolverAndBudgetShareOneRate(t *testing.T) {
	var mu sync.Mutex
	var agents []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		agents = append(agents, r.Header.Get("User-Agent"))
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"` + uuid.NewString() + `","title":"Song"}`))
	}))
	defer server.Close()
	client := &MusicBrainzClient{url: server.URL, userAgent: "TuneTrail/test", requestQueue: queue.NewRequestQueue(20, 1)}
	defer client.Shutdown()

	start := time.Now()
	var group sync.WaitGroup
	for range 3 {
		group.Go(func() {
			_, err := client.GetTrack(t.Context(), uuid.New())
			require.NoError(t, err)
		})
		group.Go(func() {
			_, err := client.Do(t.Context(), func(ctx context.Context, c *http.Client) ([]byte, error) {
				req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/ws/2/recording/"+uuid.NewString(), nil)
				if err != nil {
					return nil, err
				}
				req.Header.Set("User-Agent", "TuneTrail/test")
				resp, err := c.Do(req)
				if err != nil {
					return nil, err
				}
				return nil, resp.Body.Close()
			})
			require.NoError(t, err)
		})
	}
	group.Wait()

	require.GreaterOrEqual(t, time.Since(start), 200*time.Millisecond, "one shared rate spaces all six requests")
	require.Len(t, agents, 6)
	for _, agent := range agents {
		require.Equal(t, "TuneTrail/test", agent)
	}
}

// TestBudgetPropagatesCancellation keeps a canceled caller from holding the shared rate.
func TestBudgetPropagatesCancellation(t *testing.T) {
	client := &MusicBrainzClient{url: "http://127.0.0.1:1", userAgent: "TuneTrail/test", requestQueue: queue.NewRequestQueue(1, 1)}
	defer client.Shutdown()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := client.Do(ctx, func(context.Context, *http.Client) ([]byte, error) {
		t.Fatal("a canceled request must not run")
		return nil, nil
	})
	require.ErrorIs(t, err, context.Canceled)
}
