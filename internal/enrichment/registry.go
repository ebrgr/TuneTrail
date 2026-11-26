package enrichment

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
)

// Chain is one ordered list of sources that answers one question about a listen.
type Chain string

const (
	// ChainIdentity resolves the MusicBrainz IDs that a listen did not carry.
	ChainIdentity Chain = "identity"
	// ChainMetadata completes genres, tags and dates once the IDs are known.
	ChainMetadata Chain = "metadata"
)

// Chains lists the chains in execution order: identity first, because metadata needs the IDs.
var Chains = []Chain{ChainIdentity, ChainMetadata}

// Eligibility says what must hold before a receipt is queued for a provider.
type Eligibility string

const (
	// EligibleConfigured needs only the process configuration, such as a Subsonic server.
	EligibleConfigured Eligibility = "configured"
	// EligibleProfileRelay needs an approved relay credential on the profile that owns the receipt.
	EligibleProfileRelay Eligibility = "profile-relay"
)

// Descriptor declares what a provider is. The stored registry row only decides whether it runs
// and where in the order; the descriptor, which lives in code, decides what it can do.
type Descriptor struct {
	// ID is the stable registry key. It never changes once rows have been written.
	ID string
	// Source labels every declaration the provider writes.
	Source string
	// Chains lists the questions the provider can answer.
	Chains []Chain
	// Priority is the default position when no stored priority exists. Lower runs first.
	Priority    int
	Eligibility Eligibility
}

// In reports whether the provider takes part in the chain.
func (d Descriptor) In(chain Chain) bool { return slices.Contains(d.Chains, chain) }

// Validate rejects descriptors that could not be stored or ordered.
func (d Descriptor) Validate() error {
	switch {
	case strings.TrimSpace(d.ID) == "" || d.ID != strings.TrimSpace(d.ID):
		return errors.New("enrichment provider needs a trimmed ID")
	case d.Source == "":
		return fmt.Errorf("enrichment provider %q needs a declaration source", d.ID)
	case d.Priority < 0:
		return fmt.Errorf("enrichment provider %q has a negative priority", d.ID)
	case len(d.Chains) == 0:
		return fmt.Errorf("enrichment provider %q takes part in no chain", d.ID)
	}
	for _, chain := range d.Chains {
		if !slices.Contains(Chains, chain) {
			return fmt.Errorf("enrichment provider %q names unknown chain %q", d.ID, chain)
		}
	}
	switch d.Eligibility {
	case EligibleConfigured, EligibleProfileRelay:
	default:
		return fmt.Errorf("enrichment provider %q has unknown eligibility %q", d.ID, d.Eligibility)
	}
	return nil
}

// SubsonicDescriptor and ListenBrainzDescriptor are the providers that exist today.
var (
	SubsonicDescriptor = Descriptor{ID: "navidrome-subsonic", Source: sourceSubsonic, Chains: []Chain{ChainIdentity},
		Priority: 100, Eligibility: EligibleConfigured}
	ListenBrainzDescriptor = Descriptor{ID: "listenbrainz", Source: ListenBrainzSource, Chains: []Chain{ChainIdentity},
		Priority: 200, Eligibility: EligibleProfileRelay}
)

// Budget is one shared request allowance for an external service. Every caller of that service,
// the receipt resolver and each enrichment provider alike, draws from the same rate.
type Budget interface {
	// Do runs one request under the shared rate and returns its body.
	Do(ctx context.Context, job func(context.Context, *http.Client) ([]byte, error)) ([]byte, error)
}

// BudgetMusicBrainz names the allowance shared by every MusicBrainz caller.
const BudgetMusicBrainz = "musicbrainz"

type entry struct {
	descriptor Descriptor
	provider   Provider
}

// Registry holds the providers this process knows. A descriptor without a provider is known but
// unavailable, for example Subsonic when no server is configured.
type Registry struct {
	mu      sync.RWMutex
	entries map[string]entry
	budgets map[string]Budget
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{entries: make(map[string]entry), budgets: make(map[string]Budget)}
}

// Register adds a provider. A nil provider records the descriptor as unavailable.
func (r *Registry) Register(d Descriptor, p Provider) error {
	if err := d.Validate(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, dup := r.entries[d.ID]; dup {
		return fmt.Errorf("enrichment provider %q is already registered", d.ID)
	}
	r.entries[d.ID] = entry{descriptor: d, provider: p}
	return nil
}

// Descriptors lists every known provider by default priority, then ID.
func (r *Registry) Descriptors() []Descriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Descriptor, 0, len(r.entries))
	for _, e := range r.entries {
		out = append(out, e.descriptor)
	}
	slices.SortFunc(out, func(a, b Descriptor) int {
		if a.Priority != b.Priority {
			return a.Priority - b.Priority
		}
		return strings.Compare(a.ID, b.ID)
	})
	return out
}

// Descriptor returns the descriptor of a known provider.
func (r *Registry) Descriptor(id string) (Descriptor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.entries[id]
	return e.descriptor, ok
}

// Provider returns the provider only when it is available in this process.
func (r *Registry) Provider(id string) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.entries[id]
	return e.provider, ok && e.provider != nil
}

// Available reports whether the provider can run in this process.
func (r *Registry) Available(id string) bool {
	_, ok := r.Provider(id)
	return ok
}

// SetBudget shares one request allowance under a service name.
func (r *Registry) SetBudget(name string, b Budget) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.budgets[name] = b
}

// Budget returns the shared allowance of a service, if one was registered.
func (r *Registry) Budget(name string) (Budget, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	b, ok := r.budgets[name]
	return b, ok
}
