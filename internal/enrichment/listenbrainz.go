package enrichment

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/ebrgr/TuneTrail/internal/relay"
	"github.com/google/uuid"
)

const (
	ListenBrainzBase   = "https://api.listenbrainz.org/1"
	ListenBrainzSource = "listenbrainz-lookup"
	maxLookupChars     = 250
)

// ListenBrainzRequest carries local evidence and the individual fields still eligible.
type ListenBrainzRequest struct {
	ProfileID  int32
	Generation int64
	Snapshot   ingest.Snapshot
	Recording  bool
	Artists    []string
}

type listenBrainzKey struct {
	profile    int32
	generation int64
	query      string
}

type listenBrainzMatch struct {
	Recording string   `json:"recording_mbid"`
	Title     string   `json:"recording_name"`
	Artist    string   `json:"artist_credit_name"`
	ArtistIDs []string `json:"artist_mbids"`
	Release   string   `json:"release_name"`
	ReleaseID string   `json:"release_mbid"`
	Metadata  struct {
		Artist struct {
			Credits []struct {
				ID   string `json:"artist_mbid"`
				Name string `json:"name"`
			} `json:"artists"`
		} `json:"artist"`
	} `json:"metadata"`
}

type cachedListenBrainzMatch struct {
	match listenBrainzMatch
	at    time.Time
}

// ListenBrainz performs bounded lookups against the official service.
type ListenBrainz struct {
	endpoint  string
	userAgent string
	client    *http.Client
	interval  time.Duration
	gate      chan struct{}
	next      time.Time
	cooldown  time.Time
	cache     map[listenBrainzKey]cachedListenBrainzMatch
}

// NewListenBrainz fixes the credential destination, disables redirects and bypasses proxies.
func NewListenBrainz(userAgent string, transport http.RoundTripper) *ListenBrainz {
	if transport == nil {
		transport = relay.PublicHTTPSTransport()
	}
	return &ListenBrainz{
		endpoint: ListenBrainzBase + "/metadata/lookup/", userAgent: userAgent,
		client: &http.Client{Timeout: 5 * time.Second, Transport: transport,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
		interval: time.Second, gate: make(chan struct{}, 1),
		cache: make(map[listenBrainzKey]cachedListenBrainzMatch),
	}
}

// Lookup sends only text metadata and asks for structured credits when resolving artists.
func (c *ListenBrainz) Lookup(ctx context.Context, token string, request ListenBrainzRequest) Result {
	if token == "" {
		return Result{Status: Disabled}
	}
	s := request.Snapshot
	title, artist, release := strings.TrimSpace(s.Title), strings.TrimSpace(s.Artist), strings.TrimSpace(s.Release)
	if (!request.Recording && len(request.Artists) == 0) || s.TruncatedFields > 0 || title == "" || artist == "" ||
		(!validUUID(s.Recording) && !validUUID(s.MappedRecording) && release == "") ||
		utf8.RuneCountInString(title)+utf8.RuneCountInString(artist)+utf8.RuneCountInString(release) > maxLookupChars {
		return Result{Status: Absent}
	}
	query := url.Values{"artist_name": {artist}, "recording_name": {title}}
	if release != "" {
		query.Set("release_name", release)
	}
	if len(request.Artists) > 0 || len(s.ArtistMbidMappings) > 0 || len(s.ArtistMBIDs) > 0 || len(s.MappedArtistMBIDs) > 0 {
		query.Set("metadata", "true")
		query.Set("inc", "artist")
	}
	cached, ok := c.fetch(ctx, token, listenBrainzKey{request.ProfileID, request.Generation, query.Encode()})
	if !ok {
		return Result{Status: Failed}
	}
	return cached.match.declarations(request, cached.at)
}

// fetch serializes requests across profiles and caches only successful HTTP responses.
func (c *ListenBrainz) fetch(ctx context.Context, token string, key listenBrainzKey) (cachedListenBrainzMatch, bool) {
	select {
	case c.gate <- struct{}{}:
		defer func() { <-c.gate }()
	case <-ctx.Done():
		return cachedListenBrainzMatch{}, false
	}
	if cached, ok := c.cache[key]; ok && time.Since(cached.at) < 10*time.Minute {
		return cached, true
	}
	if time.Now().Before(c.cooldown) {
		return cachedListenBrainzMatch{}, false
	}
	if wait := time.Until(c.next); wait > 0 {
		timer := time.NewTimer(wait)
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-ctx.Done():
			return cachedListenBrainzMatch{}, false
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint+"?"+key.query, nil)
	if err != nil {
		return cachedListenBrainzMatch{}, false
	}
	req.Header.Set("Authorization", "Token "+token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	c.next = time.Now().Add(c.interval)
	resp, err := c.client.Do(req)
	if err != nil {
		return cachedListenBrainzMatch{}, false
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusTooManyRequests || resp.Header.Get("X-RateLimit-Remaining") == "0" {
		c.cooldown = lookupCooldown(resp.Header, time.Now())
	}
	if resp.StatusCode != http.StatusOK {
		return cachedListenBrainzMatch{}, false
	}
	const maxBody = 64 << 10
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody+1))
	var match listenBrainzMatch
	if err != nil || len(body) > maxBody || json.Unmarshal(body, &match) != nil {
		return cachedListenBrainzMatch{}, false
	}
	cached := cachedListenBrainzMatch{match: match, at: time.Now().UTC()}
	if len(c.cache) >= 1024 {
		clear(c.cache)
	}
	c.cache[key] = cached
	return cached, true
}

// lookupCooldown honors valid server hints and pauses briefly even without headers.
func lookupCooldown(headers http.Header, now time.Time) time.Time {
	end := now.Add(time.Second)
	for _, name := range []string{"Retry-After", "X-RateLimit-Reset-In"} {
		if seconds, err := strconv.ParseInt(headers.Get(name), 10, 64); err == nil && seconds > 0 {
			end = latestTime(end, now.Add(time.Duration(min(seconds, 86400))*time.Second))
		}
	}
	if reset, err := strconv.ParseInt(headers.Get("X-RateLimit-Reset"), 10, 64); err == nil {
		end = latestTime(end, time.Unix(reset, 0))
	}
	if retry, err := http.ParseTime(headers.Get("Retry-After")); err == nil {
		end = latestTime(end, retry)
	}
	return now.Add(min(end.Sub(now), 24*time.Hour))
}

// latestTime keeps the longer of two server-provided waits.
func latestTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

// declarations only accepts a corroborated candidate and independently binds named credits.
func (m listenBrainzMatch) declarations(request ListenBrainzRequest, at time.Time) Result {
	s := request.Snapshot
	if m.Recording == "" {
		return Result{Status: Absent}
	}
	if !validUUID(m.Recording) || !equalMetadata(m.Title, s.Title) || !equalMetadata(m.Artist, s.Artist) ||
		(strings.TrimSpace(s.Release) != "" && !equalMetadata(m.Release, s.Release)) {
		return Result{Status: Ambiguous}
	}
	for _, pair := range [][2]string{{s.Recording, m.Recording}, {s.MappedRecording, m.Recording}, {s.ReleaseMBID, m.ReleaseID}, {s.MappedReleaseMBID, m.ReleaseID}} {
		if validUUID(pair[0]) && !sameLookupID(pair[0], pair[1]) {
			return Result{Status: Ambiguous}
		}
	}
	names := s.ArtistNames
	if len(names) == 0 {
		names = []string{s.Artist}
	}
	for _, ids := range [][]string{s.ArtistMBIDs, s.MappedArtistMBIDs} {
		for i, id := range ids {
			if validUUID(id) && !slices.ContainsFunc(m.ArtistIDs, func(candidate string) bool { return sameLookupID(id, candidate) }) {
				return Result{Status: Ambiguous}
			}
			if len(ids) == len(names) && validUUID(id) && !sameLookupID(id, m.artistID(names[i])) {
				return Result{Status: Ambiguous}
			}
		}
	}
	for _, known := range s.ArtistMbidMappings {
		if validUUID(known.MBID) && !sameLookupID(known.MBID, m.artistID(known.Name)) {
			return Result{Status: Ambiguous}
		}
	}
	result := Result{Status: Absent}
	add := func(entity, name, id string) {
		result.Declarations = append(result.Declarations, Declaration{
			Entity: entity, Name: name, Field: "musicbrainz_id", Value: uuid.MustParse(id).String(),
			Source: ListenBrainzSource, ObservedAt: at,
		})
		result.Status = Found
	}
	if request.Recording {
		add("recording", "", m.Recording)
	}
	for _, name := range request.Artists {
		if id := m.artistID(name); id != "" {
			add("artist", name, id)
		}
	}
	return result
}

// artistID requires one matching credit and verifies its ID against the mapper's artist list.
func (m listenBrainzMatch) artistID(name string) string {
	credits := m.Metadata.Artist.Credits
	if len(credits) == 0 && len(m.ArtistIDs) == 1 && equalMetadata(m.Artist, name) && validUUID(m.ArtistIDs[0]) {
		return m.ArtistIDs[0]
	}
	var id string
	for _, credit := range credits {
		if !equalMetadata(credit.Name, name) {
			continue
		}
		if id != "" || !validUUID(credit.ID) {
			return ""
		}
		id = credit.ID
	}
	if slices.ContainsFunc(m.ArtistIDs, func(candidate string) bool { return sameLookupID(id, candidate) }) {
		return id
	}
	return ""
}

// sameLookupID compares UUIDs independently of their presentation.
func sameLookupID(a, b string) bool {
	left, leftErr := uuid.Parse(a)
	right, rightErr := uuid.Parse(b)
	return leftErr == nil && rightErr == nil && left != uuid.Nil && left == right
}
