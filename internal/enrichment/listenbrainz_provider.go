package enrichment

import "context"

// ListenBrainzProvider adapts the credential-bound client to the common Provider contract, so the
// chain runner treats it like any other source (ISSUE-214).
type ListenBrainzProvider struct{ client *ListenBrainz }

// NewListenBrainzProvider wraps an existing client.
func NewListenBrainzProvider(client *ListenBrainz) *ListenBrainzProvider {
	return &ListenBrainzProvider{client: client}
}

// Instance is the official API, the only destination the credential is ever sent to.
func (*ListenBrainzProvider) Instance() string { return ListenBrainzBase }

// Lookup forwards the prepared request and the profile credential. A request without a credential
// returns Disabled before any network use.
func (p *ListenBrainzProvider) Lookup(ctx context.Context, req Request) Result {
	return p.client.Lookup(ctx, req.Credential, ListenBrainzRequest{
		ProfileID: req.ProfileID, Generation: req.Generation, Snapshot: req.Snapshot,
		Recording: req.Recording, Artists: req.Artists,
	})
}
