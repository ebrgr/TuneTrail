package images

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/ebrgr/TuneTrail/internal/cfg"
	"github.com/ebrgr/TuneTrail/internal/logger"
	"github.com/ebrgr/TuneTrail/queue"
	"github.com/google/uuid"
)

// i told gemini to write this cuz i figured it would be simple enough and
// it looks like it just works? maybe ai is actually worth one quintillion gallons of water

type LastFMClient struct {
	apiKey       string
	baseUrl      string
	userAgent    string
	requestQueue *queue.RequestQueue
}

// LastFM JSON structures use "#text" for the value of XML-mapped fields
type lastFMImage struct {
	URL  string `json:"#text"`
	Size string `json:"size"`
}

type lastFMAlbumResponse struct {
	Album struct {
		Name  string        `json:"name"`
		Image []lastFMImage `json:"image"`
	} `json:"album"`
	Error   int    `json:"error"`
	Message string `json:"message"`
}

type lastFMArtistResponse struct {
	Artist struct {
		Name  string        `json:"name"`
		Image []lastFMImage `json:"image"`
	} `json:"artist"`
	Error   int    `json:"error"`
	Message string `json:"message"`
}

const (
	lastFMApiBaseUrl = "http://ws.audioscrobbler.com/2.0/"
)

func NewLastFMClient() *LastFMClient {
	ret := new(LastFMClient)
	ret.apiKey = cfg.LastFMApiKey()
	ret.baseUrl = lastFMApiBaseUrl
	ret.userAgent = cfg.UserAgent()
	ret.requestQueue = queue.NewRequestQueue(5, 5)
	return ret
}

// queue propagates caller and shutdown cancellation through the HTTP request.
// Shutdown cancels pending provider requests.
func (c *LastFMClient) Shutdown() { c.requestQueue.Shutdown() }

func (c *LastFMClient) queue(ctx context.Context, req *http.Request) ([]byte, error) {
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")
	result := <-c.requestQueue.EnqueueContext(ctx, func(jobCtx context.Context, client *http.Client) queue.RequestResult {
		resp, err := client.Do(req.WithContext(jobCtx))
		if err != nil {
			if jobCtx.Err() != nil {
				return queue.RequestResult{Err: jobCtx.Err()}
			}
			return queue.RequestResult{Err: fmt.Errorf("upstream request failed")}
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 500 {
			return queue.RequestResult{Err: fmt.Errorf("LastFM returned HTTP %d", resp.StatusCode)}
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, (8<<20)+1))
		if err == nil && len(body) > 8<<20 {
			err = fmt.Errorf("LastFM response exceeds 8 MiB")
		}
		return queue.RequestResult{Body: body, Err: err}
	})
	return result.Body, result.Err
}

func (c *LastFMClient) getEntity(ctx context.Context, params url.Values, result any) error {
	l := logger.FromContext(ctx)

	// Add standard parameters
	params.Set("api_key", c.apiKey)
	params.Set("format", "json")

	// Construct URL
	reqUrl, _ := url.Parse(c.baseUrl)
	reqUrl.RawQuery = params.Encode()

	l.Debug().Msg("Sending request to LastFM")

	req, err := http.NewRequest("GET", reqUrl.String(), nil)
	if err != nil {
		return fmt.Errorf("getEntity: %w", err)
	}

	l.Debug().Msg("Adding LastFM request to queue")
	body, err := c.queue(ctx, req)
	if err != nil {
		l.Err(err).Msg("LastFM request failed")
		return fmt.Errorf("getEntity: %w", err)
	}

	err = json.Unmarshal(body, result)
	if err != nil {
		l.Err(err).Msg("Failed to unmarshal LastFM response")
		return fmt.Errorf("getEntity: %w", err)
	}

	return nil
}

// lastFMPlaceholderHash is the hash of Last.fm's generic "no artist image" placeholder.
// Last.fm stopped serving real artist images years ago and returns this for nearly every artist.
const lastFMPlaceholderHash = "2a96cbd8b46e442fc41c2b86b821562f"

// selectBestImage checks image candidates with the caller context.
func (c *LastFMClient) selectBestImage(ctx context.Context, images []lastFMImage) string {
	// Rank preference: mega > extralarge > large > medium > small
	// Since LastFM usually returns them in order of size, we could take the last one,
	// but a map lookup is safer against API changes.

	imgMap := make(map[string]string)
	for _, img := range images {
		if img.URL != "" && !strings.Contains(img.URL, lastFMPlaceholderHash) {
			imgMap[img.Size] = img.URL
		}
	}

	if url, ok := imgMap["mega"]; ok {
		if err := ValidateImageURLContext(ctx, overrideImgSize(url)); err == nil {
			return overrideImgSize(url)
		} else {
			return url
		}
	}
	if url, ok := imgMap["extralarge"]; ok {
		if err := ValidateImageURLContext(ctx, overrideImgSize(url)); err == nil {
			return overrideImgSize(url)
		} else {
			return url
		}
	}
	if url, ok := imgMap["large"]; ok {
		if err := ValidateImageURLContext(ctx, overrideImgSize(url)); err == nil {
			return overrideImgSize(url)
		} else {
			return url
		}
	}
	if url, ok := imgMap["medium"]; ok {
		return url
	}
	if url, ok := imgMap["small"]; ok {
		return url
	}

	return ""
}

// lastfm seems to only return a 300x300 image even for "mega" and "extralarge" images, so I'm cheating
func overrideImgSize(url string) string {
	return strings.Replace(url, "300x300", "600x600", 1)
}

func (c *LastFMClient) GetAlbumImage(ctx context.Context, mbid *uuid.UUID, artist, album string) (string, error) {
	l := logger.FromContext(ctx)
	resp := new(lastFMAlbumResponse)
	l.Debug().Msgf("Finding album image for %s from artist %s", album, artist)

	// Helper to run the fetch
	fetch := func(query paramsBuilder) error {
		params := url.Values{}
		params.Set("method", "album.getInfo")
		query(params)
		return c.getEntity(ctx, params, resp)
	}

	// 1. Try MBID search first
	if mbid != nil {
		l.Debug().Str("mbid", mbid.String()).Msg("Searching album image by MBID")
		err := fetch(func(p url.Values) {
			p.Set("mbid", mbid.String())
		})

		// If success and no API error code
		if err == nil && resp.Error == 0 && len(resp.Album.Image) > 0 {
			best := c.selectBestImage(ctx, resp.Album.Image)
			if best != "" {
				return best, nil
			}
		} else if resp.Error != 0 {
			l.Debug().Int("api_error", resp.Error).Msg("LastFM MBID lookup failed, falling back to name")
		}
	}

	// 2. Fallback to Artist + Album name match
	l.Debug().Str("title", album).Str("artist", artist).Msg("Searching album image by title and artist")

	// Clear previous response structure just in case
	resp = new(lastFMAlbumResponse)

	err := fetch(func(p url.Values) {
		p.Set("artist", artist)
		p.Set("album", album)
		// Auto-correct spelling is useful for name lookups
		p.Set("autocorrect", "1")
	})

	if err != nil {
		return "", fmt.Errorf("GetAlbumImage: %v", err)
	}

	if resp.Error != 0 {
		return "", fmt.Errorf("GetAlbumImage: LastFM API error %d: %s", resp.Error, resp.Message)
	}

	best := c.selectBestImage(ctx, resp.Album.Image)
	if best == "" {
		return "", fmt.Errorf("GetAlbumImage: no suitable image found")
	}

	return best, nil
}

func (c *LastFMClient) GetArtistImage(ctx context.Context, mbid *uuid.UUID, artist string) (string, error) {
	l := logger.FromContext(ctx)
	resp := new(lastFMArtistResponse)
	l.Debug().Msgf("Finding artist image for %s", artist)

	fetch := func(query paramsBuilder) error {
		params := url.Values{}
		params.Set("method", "artist.getInfo")
		query(params)
		return c.getEntity(ctx, params, resp)
	}

	// 1. Try MBID search
	if mbid != nil {
		l.Debug().Str("mbid", mbid.String()).Msg("Searching artist image by MBID")
		err := fetch(func(p url.Values) {
			p.Set("mbid", mbid.String())
		})

		if err == nil && resp.Error == 0 && len(resp.Artist.Image) > 0 {
			best := c.selectBestImage(ctx, resp.Artist.Image)
			if best != "" {
				// Validate to match Subsonic implementation behavior
				if err := ValidateImageURLContext(ctx, best); err == nil {
					return best, nil
				}
			}
		}
	}

	// 2. Fallback to Artist name
	l.Debug().Str("artist", artist).Msg("Searching artist image by name")
	resp = new(lastFMArtistResponse)

	err := fetch(func(p url.Values) {
		p.Set("artist", artist)
		p.Set("autocorrect", "1")
	})

	if err != nil {
		return "", fmt.Errorf("GetArtistImage: %v", err)
	}

	if resp.Error != 0 {
		return "", fmt.Errorf("GetArtistImage: LastFM API error %d: %s", resp.Error, resp.Message)
	}

	best := c.selectBestImage(ctx, resp.Artist.Image)
	if best == "" {
		return "", fmt.Errorf("GetArtistImage: no suitable image found")
	}

	if err := ValidateImageURLContext(ctx, best); err != nil {
		return "", fmt.Errorf("GetArtistImage: failed to validate image url")
	}

	return best, nil
}

type paramsBuilder func(url.Values)
