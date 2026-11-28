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

var errISRCNotFound = errors.New("ISRC not found")

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
		if strings.ContainsRune(`+-&|!(){}[]^"~*?:\/`, r) {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// normalizedMetadata follows the comparison measured in ISSUE-215: case fold, trim and collapsed
// spaces. It deliberately keeps punctuation, so "Help!" and "Help" stay different albums.
func normalizedMetadata(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(s)), " ")
}

// hasWordCharacter reports whether a phrase has a letter or digit. The search analyzer drops a
// phrase made only of punctuation, and the title is not rechecked locally, so such a title would
// silently widen the query to every recording of the artist on the album.
func hasWordCharacter(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return true
		}
	}
	return false
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
		if resp.StatusCode == http.StatusNotFound && strings.HasPrefix(path, "/ws/2/isrc/") {
			return nil, errISRCNotFound
		}
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
	return r.artistIDsConsistent(s, prior) && len(r.Credits) > 0
}

// artistIDsConsistent prevents artist claims when a known credit conflicts.
func (r mbRecording) artistIDsConsistent(s ingest.Snapshot, prior []Declaration) bool {
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
	return true
}

// supportsKnownArtists requires each known artist to appear by name or ID and rejects conflicts.
func (r mbRecording) supportsKnownArtists(s ingest.Snapshot, prior []Declaration) bool {
	known := slices.Clone(s.ArtistMbidMappings)
	known = append(known, s.PositionalArtistMappings(s.ArtistMBIDs)...)
	known = append(known, s.PositionalArtistMappings(s.MappedArtistMBIDs)...)
	for _, p := range prior {
		if p.Entity == "artist" && p.Field == "musicbrainz_id" {
			known = append(known, ingest.ArtistMapping{Name: p.Name, MBID: p.Value})
		}
	}
	// Preserve unaligned received IDs without guessing their positions or names.
	for _, id := range append(slices.Clone(s.ArtistMBIDs), s.MappedArtistMBIDs...) {
		if !validUUID(id) {
			continue
		}
		mapped := false
		for _, k := range known {
			if validUUID(k.MBID) && strings.EqualFold(k.MBID, id) {
				mapped = true
				break
			}
		}
		if !mapped {
			known = append(known, ingest.ArtistMapping{MBID: id})
		}
	}
	for _, k := range known {
		if !validUUID(k.MBID) {
			continue
		}
		found := false
		for _, credit := range r.Credits {
			sameName := strings.TrimSpace(k.Name) != "" && equalMetadata(k.Name, credit.Name)
			if sameName && validUUID(credit.Artist.ID) && !strings.EqualFold(k.MBID, credit.Artist.ID) {
				return false
			}
			if sameName || (validUUID(credit.Artist.ID) && strings.EqualFold(k.MBID, credit.Artist.ID)) {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	return true
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
	if (!missingRecording && len(names) == 0) || !hasWordCharacter(s.Title) || strings.TrimSpace(s.Artist) == "" || strings.TrimSpace(s.Release) == "" || s.TruncatedFields > 0 {
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
	// Bound ISRC lookups to four codes so the shared one-request-per-second budget leaves
	// time for name fallback and other providers within the job's finite lease.
	isrcFailed := false
	if missingRecording {
		unique := map[string]bool{}
		for _, p := range req.Prior {
			if p.Entity == "recording" && p.Field == "isrc" && strings.TrimSpace(p.Value) != "" {
				unique[strings.TrimSpace(p.Value)] = true
			}
		}
		codes := make([]string, 0, len(unique))
		for code := range unique {
			codes = append(codes, code)
		}
		slices.Sort(codes)
		if len(codes) > 4 {
			isrcFailed = true
		} else {
			recordings := map[string]mbRecording{}
			for _, code := range codes {
				list, status := m.lookupISRCRecordings(ctx, code)
				if status == Failed {
					isrcFailed = true
					continue
				}
				if status == Ambiguous {
					result := Result{Status: Ambiguous}
					if ctx.Err() == nil {
						m.cacheResult(key, result)
					}
					return result
				}
				for _, rec := range list {
					id := uuid.MustParse(rec.ID).String()
					if old, ok := recordings[id]; ok {
						for _, credit := range rec.Credits {
							duplicate := false
							for _, existing := range old.Credits {
								if equalMetadata(existing.Name, credit.Name) && strings.EqualFold(existing.Artist.ID, credit.Artist.ID) {
									duplicate = true
									break
								}
							}
							if !duplicate {
								old.Credits = append(old.Credits, credit)
							}
						}
						recordings[id] = old
					} else {
						recordings[id] = rec
					}
				}
			}
			if !isrcFailed && len(recordings) > 1 {
				result := Result{Status: Ambiguous}
				if ctx.Err() == nil {
					m.cacheResult(key, result)
				}
				return result
			}
			if !isrcFailed && len(recordings) == 1 {
				result := Result{Status: Found}
				for id, rec := range recordings {
					// Existing MusicBrainz identities take precedence over ISRC evidence.
					if !rec.supportsKnownArtists(s, req.Prior) {
						continue
					}
					result.Declarations = append(result.Declarations, Declaration{Entity: "recording", Field: "musicbrainz_id", Value: id, Source: musicBrainzSource, ObservedAt: time.Now().UTC()})
					complete := true
					for _, name := range s.ArtistCreditNames() {
						found := false
						for _, credit := range rec.Credits {
							if equalMetadata(name, credit.Name) {
								found = true
								break
							}
						}
						if !found {
							complete = false
							break
						}
					}
					if !complete {
						continue
					}
					for _, name := range names {
						var matches []mbCredit
						for _, credit := range rec.Credits {
							if equalMetadata(name, credit.Name) {
								matches = append(matches, credit)
							}
						}
						if len(matches) != 1 || !validUUID(matches[0].Artist.ID) {
							continue
						}
						result.Declarations = append(result.Declarations, Declaration{Entity: "artist", Name: name, Field: "musicbrainz_id", Value: uuid.MustParse(matches[0].Artist.ID).String(), Source: musicBrainzSource, ObservedAt: time.Now().UTC()})
					}
				}
				if len(result.Declarations) > 0 {
					if ctx.Err() == nil {
						m.cacheResult(key, result)
					}
					return result
				}
				// Conflicting ISRC evidence is discarded; a name search may still resolve it.
			}
		}
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
	if isrcFailed && result.Status == Absent {
		return Result{Status: Failed}
	}
	if ctx.Err() == nil {
		m.cacheResult(key, result)
	}
	return result
}

// cacheResult retains bounded successful or negative lookup results.
func (m *MusicBrainz) cacheResult(key string, result Result) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.cache) >= 1024 {
		clear(m.cache)
	}
	m.cache[key] = cachedResult{result: result, until: time.Now().Add(10 * time.Minute)}
}

// lookupISRCRecordings requests credits and preserves the distinction between 404 and failure.
func (m *MusicBrainz) lookupISRCRecordings(ctx context.Context, isrc string) ([]mbRecording, Status) {
	if strings.TrimSpace(isrc) == "" || strings.ContainsAny(isrc, "/\\") {
		return nil, Absent
	}
	response, err := m.call(ctx, "/ws/2/isrc/"+url.PathEscape(isrc), url.Values{"fmt": {"json"}, "inc": {"artist-credits"}})
	if errors.Is(err, errISRCNotFound) {
		return nil, Absent
	}
	if err != nil {
		return nil, Failed
	}
	ids := map[string]mbRecording{}
	for _, rec := range response.Recordings {
		if validUUID(rec.ID) {
			ids[uuid.MustParse(rec.ID).String()] = rec
		}
	}
	if len(ids) > 1 || response.Count > len(response.Recordings) {
		return nil, Ambiguous
	}
	result := make([]mbRecording, 0, len(ids))
	for _, rec := range ids {
		result = append(result, rec)
	}
	if len(result) == 0 {
		return nil, Absent
	}
	return result, Found
}

// lookupISRC keeps the single-code interface used by callers and its existing tests.
func (m *MusicBrainz) lookupISRC(ctx context.Context, isrc string) Result {
	records, status := m.lookupISRCRecordings(ctx, isrc)
	if status != Found {
		return Result{Status: status}
	}
	return Result{Status: Found, Declarations: []Declaration{{Entity: "recording", Field: "musicbrainz_id", Value: uuid.MustParse(records[0].ID).String(), Source: musicBrainzSource, ObservedAt: time.Now().UTC()}}}
}
