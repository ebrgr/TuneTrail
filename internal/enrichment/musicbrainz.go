package enrichment

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/google/uuid"
)

const musicBrainzSource = "musicbrainz-search"

// MusicBrainzDescriptor places open identity lookup between Subsonic and ListenBrainz.
var MusicBrainzDescriptor = Descriptor{ID: "musicbrainz", Source: musicBrainzSource, Chains: []Chain{ChainIdentity}, Priority: 150, Eligibility: EligibleConfigured}

type mbCredit struct {
	Name   string `json:"name"`
	Artist struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"artist"`
}
type mbRecording struct {
	ID       string     `json:"id"`
	Score    int        `json:"score"`
	Length   *int64     `json:"length"`
	Credits  []mbCredit `json:"artist-credit"`
	Releases []struct {
		Title string `json:"title"`
	} `json:"releases"`
}
type mbResponse struct {
	Count      int           `json:"count"`
	Recordings []mbRecording `json:"recordings"`
}

// MusicBrainz searches through the same budget as the existing resolver.
type MusicBrainz struct {
	endpoint, userAgent string
	budget              Budget
	mu                  sync.Mutex
	cache               map[string]cachedResult
}

// NewMusicBrainz creates a provider backed exclusively by the shared budget.
func NewMusicBrainz(endpoint, userAgent string, budget Budget) *MusicBrainz {
	return &MusicBrainz{endpoint: strings.TrimRight(endpoint, "/"), userAgent: userAgent, budget: budget, cache: make(map[string]cachedResult)}
}

// Instance identifies the configured MusicBrainz server.
func (m *MusicBrainz) Instance() string { return m.endpoint }

// luceneEscape protects the phrase from Lucene operators and quoted-string delimiters.
func luceneEscape(s string) string {
	var b strings.Builder
	for _, r := range s {
		if strings.ContainsRune(`+ -&|!(){}[]^"~*?:\/`, r) && r != ' ' {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// normalizedMetadata compares case, whitespace and punctuation consistently.
func normalizedMetadata(s string) string {
	var words []string
	for _, word := range strings.Fields(strings.ToLower(s)) {
		var b strings.Builder
		for _, r := range word {
			if unicode.IsLetter(r) || unicode.IsNumber(r) {
				b.WriteRune(r)
			}
		}
		if b.Len() > 0 {
			words = append(words, b.String())
		}
	}
	return strings.Join(words, " ")
}

// call bounds responses and uses only the HTTP client supplied by the budget.
func (m *MusicBrainz) call(ctx context.Context, path string, query url.Values) (mbResponse, error) {
	var out mbResponse
	body, err := m.budget.Do(ctx, func(ctx context.Context, client *http.Client) ([]byte, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.endpoint+path+"?"+query.Encode(), nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", m.userAgent)
		req.Header.Set("Accept", "application/json")
		// A redirect is a new request and must not bypass the shared budget.
		boundedClient := *client
		boundedClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
		resp, err := boundedClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, errors.New("MusicBrainz HTTP error")
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, (8<<20)+1))
		if err != nil {
			return nil, err
		}
		if len(body) > 8<<20 {
			return nil, errors.New("MusicBrainz response too large")
		}
		return body, nil
	})
	if err != nil {
		return out, err
	}
	if err = json.Unmarshal(body, &out); err != nil {
		return out, err
	}
	return out, nil
}

// matches checks the measured score, duration, release and artist-credit conditions.
func (r mbRecording) matches(s ingest.Snapshot, prior []Declaration) bool {
	if !validUUID(r.ID) || r.Score < 95 || strings.TrimSpace(s.Release) == "" || (s.DurationMS > 0 && (r.Length == nil || *r.Length <= 0 || abs64(s.DurationMS-*r.Length) > 5000)) {
		return false
	}
	album := false
	for _, release := range r.Releases {
		if normalizedMetadata(release.Title) != "" && normalizedMetadata(release.Title) == normalizedMetadata(s.Release) {
			album = true
			break
		}
	}
	if !album {
		return false
	}
	for _, name := range s.ArtistCreditNames() {
		found := false
		for _, credit := range r.Credits {
			if equalMetadata(name, credit.Name) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	// A conflicting known artist ID is evidence against this recording.
	known := slices.Clone(s.ArtistMbidMappings)
	known = append(known, s.PositionalArtistMappings(s.ArtistMBIDs)...)
	known = append(known, s.PositionalArtistMappings(s.MappedArtistMBIDs)...)
	for _, p := range prior {
		if p.Entity == "artist" && p.Field == "musicbrainz_id" {
			known = append(known, ingest.ArtistMapping{Name: p.Name, MBID: p.Value})
		}
	}
	for _, k := range known {
		if !validUUID(k.MBID) {
			continue
		}
		for _, c := range r.Credits {
			if equalMetadata(k.Name, c.Name) && validUUID(c.Artist.ID) && !strings.EqualFold(k.MBID, c.Artist.ID) {
				return false
			}
		}
	}
	return len(r.Credits) > 0
}

// Lookup returns only missing declarations and caches bounded successful answers.
func (m *MusicBrainz) Lookup(ctx context.Context, req Request) Result {
	s := req.Snapshot
	missingRecording := !validUUID(s.Recording) && !validUUID(s.MappedRecording)
	names := MissingArtistNames(s)
	for _, p := range req.Prior {
		if p.Field != "musicbrainz_id" || !validUUID(p.Value) {
			continue
		}
		if p.Entity == "recording" {
			missingRecording = false
		}
		if p.Entity == "artist" {
			filtered := names[:0]
			for _, n := range names {
				if !equalMetadata(n, p.Name) {
					filtered = append(filtered, n)
				}
			}
			names = filtered
		}
	}
	if (!missingRecording && len(names) == 0) || strings.TrimSpace(s.Title) == "" || strings.TrimSpace(s.Artist) == "" || strings.TrimSpace(s.Release) == "" || s.TruncatedFields > 0 {
		return Result{Status: Absent}
	}
	keyBytes, err := json.Marshal(struct {
		Profile   int32
		Snapshot  ingest.Snapshot
		Recording bool
		Names     []string
		Prior     []Declaration
	}{req.ProfileID, ingest.Snapshot{Title: s.Title, Artist: s.Artist, Release: s.Release, DurationMS: s.DurationMS, Recording: s.Recording, MappedRecording: s.MappedRecording, ArtistNames: s.ArtistNames, ArtistMBIDs: s.ArtistMBIDs, MappedArtistMBIDs: s.MappedArtistMBIDs, ArtistMbidMappings: s.ArtistMbidMappings}, missingRecording, names, req.Prior})
	if err != nil {
		return Result{Status: Failed}
	}
	digest := sha256.Sum256(keyBytes)
	key := string(digest[:])
	m.mu.Lock()
	cached, ok := m.cache[key]
	m.mu.Unlock()
	if ok && time.Now().Before(cached.until) {
		return cached.result
	}
	query := `recording:"` + luceneEscape(s.Title) + `" AND artist:"` + luceneEscape(s.Artist) + `" AND release:"` + luceneEscape(s.Release) + `"`
	response, err := m.call(ctx, "/ws/2/recording", url.Values{"query": {query}, "fmt": {"json"}, "limit": {"100"}})
	if err != nil {
		return Result{Status: Failed}
	}
	matches := map[string]mbRecording{}
	for _, r := range response.Recordings {
		if r.matches(s, req.Prior) {
			matches[uuid.MustParse(r.ID).String()] = r
		}
	}
	result := Result{Status: Absent}
	if len(response.Recordings) >= 100 || response.Count > len(response.Recordings) {
		result.Status = Ambiguous
	} else if len(matches) > 1 {
		result.Status = Ambiguous
	} else if len(matches) == 1 {
		for id, r := range matches {
			if (validUUID(s.Recording) && !strings.EqualFold(s.Recording, id)) || (validUUID(s.MappedRecording) && !strings.EqualFold(s.MappedRecording, id)) {
				break
			}
			for _, p := range req.Prior {
				if p.Entity == "recording" && p.Field == "musicbrainz_id" && validUUID(p.Value) && !strings.EqualFold(p.Value, id) {
					return Result{Status: Absent}
				}
			}
			at := time.Now().UTC()
			if missingRecording {
				result.Declarations = append(result.Declarations, Declaration{Entity: "recording", Field: "musicbrainz_id", Value: id, Source: musicBrainzSource, ObservedAt: at})
			}
			for _, name := range names {
				var credits []mbCredit
				for _, c := range r.Credits {
					if equalMetadata(name, c.Name) {
						credits = append(credits, c)
					}
				}
				if len(credits) == 1 && validUUID(credits[0].Artist.ID) {
					result.Declarations = append(result.Declarations, Declaration{Entity: "artist", Name: name, Field: "musicbrainz_id", Value: uuid.MustParse(credits[0].Artist.ID).String(), Source: musicBrainzSource, ObservedAt: at})
				}
			}
			if len(result.Declarations) > 0 {
				result.Status = Found
			}
		}
	}
	if ctx.Err() == nil {
		m.mu.Lock()
		if len(m.cache) >= 1024 {
			clear(m.cache)
		}
		m.cache[key] = cachedResult{result: result, until: time.Now().Add(10 * time.Minute)}
		m.mu.Unlock()
	}
	return result
}

// lookupISRC is deliberately disconnected until snapshots carry ISRC (ISSUE-219).
// It accepts only one distinct recording ID and treats multiple IDs as ambiguous.
func (m *MusicBrainz) lookupISRC(ctx context.Context, isrc string) Result {
	if strings.TrimSpace(isrc) == "" || strings.ContainsAny(isrc, "/\\") {
		return Result{Status: Absent}
	}
	response, err := m.call(ctx, "/ws/2/isrc/"+url.PathEscape(isrc), url.Values{"fmt": {"json"}})
	if err != nil {
		return Result{Status: Failed}
	}
	ids := map[string]bool{}
	for _, r := range response.Recordings {
		if validUUID(r.ID) {
			ids[uuid.MustParse(r.ID).String()] = true
		}
	}
	if len(ids) > 1 || len(response.Recordings) > 1 || response.Count > 1 {
		return Result{Status: Ambiguous}
	}
	for id := range ids {
		return Result{Status: Found, Declarations: []Declaration{{Entity: "recording", Field: "musicbrainz_id", Value: id, Source: musicBrainzSource, ObservedAt: time.Now().UTC()}}}
	}
	return Result{Status: Absent}
}
