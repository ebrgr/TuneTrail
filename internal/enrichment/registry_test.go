package enrichment

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/stretchr/testify/require"
)

type stubProvider struct{ instance string }

func (s stubProvider) Lookup(context.Context, Request) Result { return Result{Status: Absent} }
func (s stubProvider) Instance() string                       { return s.instance }

// TestRegistryOrdersByPriorityThenID keeps the default order deterministic.
func TestRegistryOrdersByPriorityThenID(t *testing.T) {
	r := NewRegistry()
	for _, d := range []Descriptor{
		{ID: "late", Source: "s", Chains: []Chain{ChainIdentity}, Priority: 300, Eligibility: EligibleConfigured},
		{ID: "b-tie", Source: "s", Chains: []Chain{ChainIdentity}, Priority: 100, Eligibility: EligibleConfigured},
		{ID: "a-tie", Source: "s", Chains: []Chain{ChainMetadata}, Priority: 100, Eligibility: EligibleConfigured},
	} {
		require.NoError(t, r.Register(d, nil))
	}
	var ids []string
	for _, d := range r.Descriptors() {
		ids = append(ids, d.ID)
	}
	require.Equal(t, []string{"a-tie", "b-tie", "late"}, ids)
}

// TestRegistryRejectsInvalidAndDuplicateDescriptors protects the stored keys from bad input.
func TestRegistryRejectsInvalidAndDuplicateDescriptors(t *testing.T) {
	ok := Descriptor{ID: "x", Source: "s", Chains: []Chain{ChainIdentity}, Eligibility: EligibleConfigured}
	for name, mutate := range map[string]func(*Descriptor){
		"empty id":        func(d *Descriptor) { d.ID = "" },
		"padded id":       func(d *Descriptor) { d.ID = " x" },
		"no source":       func(d *Descriptor) { d.Source = "" },
		"negative":        func(d *Descriptor) { d.Priority = -1 },
		"no chain":        func(d *Descriptor) { d.Chains = nil },
		"unknown chain":   func(d *Descriptor) { d.Chains = []Chain{"other"} },
		"no eligibility":  func(d *Descriptor) { d.Eligibility = "" },
		"bad eligibility": func(d *Descriptor) { d.Eligibility = "anyone" },
	} {
		t.Run(name, func(t *testing.T) {
			d := ok
			mutate(&d)
			require.Error(t, NewRegistry().Register(d, nil))
		})
	}
	r := NewRegistry()
	require.NoError(t, r.Register(ok, nil))
	require.Error(t, r.Register(ok, nil), "an ID can be registered once")
}

// TestRegistryAvailabilityFollowsTheProvider shows that a descriptor alone is known but unavailable.
func TestRegistryAvailabilityFollowsTheProvider(t *testing.T) {
	r := NewRegistry()
	require.NoError(t, r.Register(SubsonicDescriptor, nil))
	require.NoError(t, r.Register(ListenBrainzDescriptor, NewListenBrainzProvider(NewListenBrainz("TuneTrail/test", nil))))
	require.False(t, r.Available(SubsonicDescriptor.ID))
	require.True(t, r.Available(ListenBrainzDescriptor.ID))
	_, known := r.Descriptor(SubsonicDescriptor.ID)
	require.True(t, known)
	_, ok := r.Provider("missing")
	require.False(t, ok)
	require.True(t, SubsonicDescriptor.In(ChainIdentity))
	require.False(t, SubsonicDescriptor.In(ChainMetadata))
}

// TestBuiltInDescriptorsAreValidAndOrdered pins the order that ISSUE-174 and 175 already had.
func TestBuiltInDescriptorsAreValidAndOrdered(t *testing.T) {
	require.NoError(t, SubsonicDescriptor.Validate())
	require.NoError(t, ListenBrainzDescriptor.Validate())
	require.Less(t, SubsonicDescriptor.Priority, ListenBrainzDescriptor.Priority)
	require.Equal(t, "navidrome-subsonic", SubsonicDescriptor.Source)
	require.Equal(t, ListenBrainzSource, ListenBrainzDescriptor.Source)
}

// TestRequestExtraFieldsStayOutOfProviderCacheKeys keeps the hashed JSON of the request unchanged.
func TestRequestExtraFieldsStayOutOfProviderCacheKeys(t *testing.T) {
	plain, err := json.Marshal(Request{ProfileID: 1, Snapshot: ingest.Snapshot{Title: "T"}})
	require.NoError(t, err)
	extended, err := json.Marshal(Request{ProfileID: 1, Snapshot: ingest.Snapshot{Title: "T"},
		Chain: ChainIdentity, Credential: "secret-token", Generation: 7, Recording: true, Artists: []string{"A"}})
	require.NoError(t, err)
	require.JSONEq(t, string(plain), string(extended))
	require.NotContains(t, string(extended), "secret-token")
}

// TestListenBrainzProviderForwardsTheCredential checks the adapter sends the prepared request as is.
func TestListenBrainzProviderForwardsTheCredential(t *testing.T) {
	var got string
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()
	client := NewListenBrainz("TuneTrail/test", server.Client().Transport)
	client.endpoint = server.URL + "/metadata/lookup/"
	provider := NewListenBrainzProvider(client)
	require.Equal(t, ListenBrainzBase, provider.Instance())

	req := Request{ProfileID: 1, Generation: 3, Recording: true, Credential: "adapter-token",
		Snapshot: ingest.Snapshot{Title: "Song", Artist: "Artist", Release: "Album"}}
	provider.Lookup(t.Context(), req)
	require.Equal(t, "Token adapter-token", got)

	got = ""
	req.Credential = ""
	require.Equal(t, Disabled, provider.Lookup(t.Context(), req).Status, "no credential means no network use")
	require.Empty(t, got)
}

type countingBudget struct {
	mu    sync.Mutex
	calls int
}

func (b *countingBudget) Do(ctx context.Context, job func(context.Context, *http.Client) ([]byte, error)) ([]byte, error) {
	b.mu.Lock()
	b.calls++
	b.mu.Unlock()
	return job(ctx, http.DefaultClient)
}

// TestRegistryBudgetsAreSharedByName proves two callers receive the same allowance.
func TestRegistryBudgetsAreSharedByName(t *testing.T) {
	r := NewRegistry()
	_, ok := r.Budget(BudgetMusicBrainz)
	require.False(t, ok)
	shared := &countingBudget{}
	r.SetBudget(BudgetMusicBrainz, shared)
	first, _ := r.Budget(BudgetMusicBrainz)
	second, _ := r.Budget(BudgetMusicBrainz)
	for _, b := range []Budget{first, second} {
		_, err := b.Do(t.Context(), func(context.Context, *http.Client) ([]byte, error) { return nil, nil })
		require.NoError(t, err)
	}
	require.Equal(t, 2, shared.calls)
}
