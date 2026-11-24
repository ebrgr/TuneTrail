package enrichment

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/google/uuid"
)

const sourceSubsonic = "navidrome-subsonic"

type cachedResult struct {
	result Result
	until  time.Time
}

// Subsonic performs bounded, conservative lookups without exposing credentials.
type Subsonic struct {
	endpoint  string
	params    url.Values
	instance  string
	userAgent string
	client    *http.Client
	timeout   time.Duration
	interval  time.Duration
	mu        sync.Mutex
	next      time.Time
	cache     map[string]cachedResult
}

// NewSubsonic builds a provider from the same URL and form parameters as images.
func NewSubsonic(endpoint, params, userAgent string, timeout, interval time.Duration) (*Subsonic, error) {
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" || timeout <= 0 || interval <= 0 {
		return nil, errors.New("invalid Subsonic configuration")
	}
	auth, err := url.ParseQuery(strings.TrimPrefix(params, "?"))
	if err != nil {
		return nil, errors.New("invalid Subsonic parameters")
	}
	for _, key := range []string{"query", "id", "f", "v", "c", "songCount", "albumCount", "artistCount"} {
		auth.Del(key)
	}
	u.Path = strings.TrimRight(u.Path, "/")
	fingerprint := sha256.Sum256([]byte(u.String() + "\x00" + auth.Encode()))
	return &Subsonic{endpoint: u.String(), params: auth, instance: hex.EncodeToString(fingerprint[:]), userAgent: userAgent,
		timeout: timeout, interval: interval, cache: make(map[string]cachedResult),
		client: &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}

// Instance identifies the configuration without retaining credentials in storage.
func (c *Subsonic) Instance() string { return c.instance }

type subsonicArtist struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	MBID string `json:"musicBrainzId"`
}

type subsonicSong struct {
	MediaType string           `json:"mediaType"`
	Title     string           `json:"title"`
	Artist    string           `json:"artist"`
	ArtistID  string           `json:"artistId"`
	Album     string           `json:"album"`
	Duration  int64            `json:"duration"`
	Track     int              `json:"track"`
	Disc      int              `json:"discNumber"`
	MBID      string           `json:"musicBrainzId"`
	Artists   []subsonicArtist `json:"artists"`
}

type subsonicResponse struct {
	Response struct {
		Status string `json:"status"`
		Search struct {
			Songs []subsonicSong `json:"song"`
		} `json:"searchResult3"`
		Artist subsonicArtist `json:"artist"`
	} `json:"subsonic-response"`
}

// call serializes rate reservations and bounds both time and response size.
func (c *Subsonic) call(ctx context.Context, endpoint string, args url.Values) (subsonicResponse, error) {
	var result subsonicResponse
	c.mu.Lock()
	now := time.Now()
	start := c.next
	if start.Before(now) {
		start = now
	}
	c.next = start.Add(c.interval)
	c.mu.Unlock()
	timer := time.NewTimer(time.Until(start))
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return result, ctx.Err()
	case <-timer.C:
	}
	form := url.Values{}
	for k, values := range c.params {
		form[k] = append([]string(nil), values...)
	}
	for k, values := range args {
		form[k] = values
	}
	form.Set("f", "json")
	form.Set("v", "1.16.1")
	form.Set("c", "tunetrail")
	requestCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, c.endpoint+"/rest/"+endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return result, errors.New("Subsonic request could not be created")
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", c.userAgent)
	resp, err := c.client.Do(req)
	if err != nil {
		return result, errors.New("Subsonic request failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return result, errors.New("Subsonic HTTP error")
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, (1<<20)+1))
	if err != nil || len(body) > 1<<20 {
		return result, errors.New("Subsonic response unavailable")
	}
	if err := json.Unmarshal(body, &result); err != nil || result.Response.Status != "ok" {
		return result, errors.New("invalid Subsonic response")
	}
	return result, nil
}

// equalMetadata requires exact trimmed metadata while allowing case differences.
func equalMetadata(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

// songMatches refuses missing fields and conflicting duration or position evidence.
func songMatches(s ingest.Snapshot, song subsonicSong) bool {
	if song.MediaType != "" && song.MediaType != "song" {
		return false
	}
	if strings.TrimSpace(s.Title) == "" || strings.TrimSpace(s.Artist) == "" || strings.TrimSpace(s.Release) == "" ||
		!equalMetadata(s.Title, song.Title) || !equalMetadata(s.Artist, song.Artist) || !equalMetadata(s.Release, song.Album) {
		return false
	}
	if s.DurationMS > 0 && (song.Duration <= 0 || song.Duration > math.MaxInt64/1000 || abs64(s.DurationMS-song.Duration*1000) > 2000) {
		return false
	}
	// A returned recording ID must agree with every supplied recording declaration.
	if song.MediaType == "song" && validUUID(song.MBID) {
		for _, known := range []string{s.Recording, s.MappedRecording} {
			if validUUID(known) && !strings.EqualFold(known, song.MBID) {
				return false
			}
		}
	}
	// Compare named IDs and any IDs with proven positions against the same song credit.
	knownArtists := slices.Clone(s.ArtistMbidMappings)
	knownArtists = append(knownArtists, s.PositionalArtistMappings(s.ArtistMBIDs)...)
	knownArtists = append(knownArtists, s.PositionalArtistMappings(s.MappedArtistMBIDs)...)
	for _, mapping := range knownArtists {
		if !validUUID(mapping.MBID) {
			continue
		}
		for _, artist := range song.Artists {
			if equalMetadata(mapping.Name, artist.Name) && validUUID(artist.MBID) && !strings.EqualFold(mapping.MBID, artist.MBID) {
				return false
			}
		}
	}
	if s.TrackNumber > 0 && s.TrackNumber != song.Track {
		return false
	}
	if s.DiscNumber > 0 && s.DiscNumber != song.Disc {
		return false
	}
	return true
}

func abs64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

// MissingArtistNames avoids inferring positions from partial or invalid ID lists.
func MissingArtistNames(s ingest.Snapshot) []string {
	// Keep holes in an aligned list and avoid reinterpreting compacted historical lists.
	names := s.ArtistCreditNames()
	if len(names) == 0 || (len(s.ArtistMBIDs) > 0 && len(s.ArtistMBIDs) != len(names)) || (len(s.MappedArtistMBIDs) > 0 && len(s.MappedArtistMBIDs) != len(names)) {
		return nil
	}
	known := make(map[string]bool)
	for _, m := range s.ArtistMbidMappings {
		if validUUID(m.MBID) {
			known[m.Name] = true
		}
	}
	for i, name := range names {
		if (len(s.ArtistMBIDs) == len(names) && validUUID(s.ArtistMBIDs[i])) || (len(s.MappedArtistMBIDs) == len(names) && validUUID(s.MappedArtistMBIDs[i])) {
			known[name] = true
		}
	}
	var missing []string
	seen := make(map[string]bool)
	for _, name := range names {
		key := strings.ToLower(strings.TrimSpace(name))
		if key == "" || seen[key] {
			return nil
		}
		seen[key] = true
		if !known[name] {
			missing = append(missing, name)
		}
	}
	return missing
}

func validUUID(raw string) bool { id, err := uuid.Parse(raw); return err == nil && id != uuid.Nil }

// Lookup never chooses between multiple plausible songs or a truncated page.
func (c *Subsonic) Lookup(ctx context.Context, req Request) Result {
	s := req.Snapshot
	missingRecording := !validUUID(s.Recording) && !validUUID(s.MappedRecording)
	missingArtists := MissingArtistNames(s)
	if !missingRecording && len(missingArtists) == 0 {
		return Result{Status: Absent}
	}
	if s.Title == "" || s.Artist == "" || s.Release == "" {
		return Result{Status: Absent}
	}
	// Ignore event timestamps and unrelated fields so repeated listens share a bounded cache.
	keyBytes, err := json.Marshal(Request{ProfileID: req.ProfileID, Snapshot: ingest.Snapshot{
		Title: s.Title, Artist: s.Artist, Release: s.Release, DurationMS: s.DurationMS,
		DiscNumber: s.DiscNumber, TrackNumber: s.TrackNumber, Recording: s.Recording, MappedRecording: s.MappedRecording,
		ArtistNames: s.ArtistNames, ArtistMBIDs: s.ArtistMBIDs, MappedArtistMBIDs: s.MappedArtistMBIDs, ArtistMbidMappings: s.ArtistMbidMappings,
	}})
	if err != nil {
		return Result{Status: Failed}
	}
	digest := sha256.Sum256(keyBytes)
	key := hex.EncodeToString(digest[:])
	c.mu.Lock()
	cached, ok := c.cache[key]
	c.mu.Unlock()
	if ok && time.Now().Before(cached.until) {
		return cached.result
	}
	result := c.lookup(ctx, s, missingRecording, missingArtists)
	if result.Status != Failed && ctx.Err() == nil {
		c.mu.Lock()
		if len(c.cache) >= 1024 {
			clear(c.cache)
		}
		c.cache[key] = cachedResult{result: result, until: time.Now().Add(10 * time.Minute)}
		c.mu.Unlock()
	}
	return result
}

// lookup only asks for artists attached to the uniquely matched song.
func (c *Subsonic) lookup(ctx context.Context, s ingest.Snapshot, recording bool, names []string) Result {
	const limit = 100
	resp, err := c.call(ctx, "search3", url.Values{"query": {s.Title}, "artistCount": {"0"}, "albumCount": {"0"}, "songCount": {strconv.Itoa(limit)}})
	if err != nil {
		return Result{Status: Failed}
	}
	if len(resp.Response.Search.Songs) >= limit {
		return Result{Status: Ambiguous}
	}
	var matches []subsonicSong
	for _, song := range resp.Response.Search.Songs {
		if songMatches(s, song) {
			matches = append(matches, song)
		}
	}
	if len(matches) == 0 {
		return Result{Status: Absent}
	}
	if len(matches) != 1 {
		return Result{Status: Ambiguous}
	}
	song := matches[0]
	at := time.Now().UTC()
	result := Result{Status: Absent}
	if recording && song.MediaType == "song" && validUUID(song.MBID) {
		result.Declarations = append(result.Declarations, Declaration{Entity: "recording", Field: "musicbrainz_id", Value: uuid.MustParse(song.MBID).String(), Source: sourceSubsonic, ObservedAt: at})
	}
	artists := song.Artists
	if len(artists) == 0 && song.ArtistID != "" {
		artists = []subsonicArtist{{ID: song.ArtistID, Name: song.Artist}}
	}
	for _, name := range names {
		var matches []subsonicArtist
		for _, artist := range artists {
			if equalMetadata(name, artist.Name) {
				matches = append(matches, artist)
			}
		}
		if len(matches) != 1 {
			continue
		}
		artist := matches[0]
		if !validUUID(artist.MBID) && artist.ID != "" {
			response, err := c.call(ctx, "getArtist", url.Values{"id": {artist.ID}})
			if err != nil {
				continue
			}
			if !equalMetadata(name, response.Response.Artist.Name) || (response.Response.Artist.ID != "" && response.Response.Artist.ID != artist.ID) {
				continue
			}
			artist = response.Response.Artist
		}
		if validUUID(artist.MBID) {
			result.Declarations = append(result.Declarations, Declaration{Entity: "artist", Name: name, Field: "musicbrainz_id", Value: uuid.MustParse(artist.MBID).String(), Source: sourceSubsonic, ObservedAt: at})
		}
	}
	if len(result.Declarations) > 0 {
		result.Status = Found
	}
	return result
}
