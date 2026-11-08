// package mbz provides functions for interacting with the musicbrainz api
package mbz

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/ebrgr/TuneTrail/internal/cfg"
	"github.com/ebrgr/TuneTrail/internal/logger"
	"github.com/ebrgr/TuneTrail/queue"
	"github.com/google/uuid"
)

type MusicBrainzArea struct {
	Name           string   `json:"name"`
	Iso3166_1Codes []string `json:"iso-3166-1-codes"`
}

type MusicBrainzClient struct {
	url          string
	userAgent    string
	requestQueue *queue.RequestQueue
}

type MusicBrainzCaller interface {
	GetArtistPrimaryAliases(ctx context.Context, id uuid.UUID) ([]string, error)
	GetReleaseTitles(ctx context.Context, RGID uuid.UUID) ([]string, error)
	GetTrack(ctx context.Context, id uuid.UUID) (*MusicBrainzTrack, error)
	GetReleaseGroup(ctx context.Context, id uuid.UUID) (*MusicBrainzReleaseGroup, error)
	GetRelease(ctx context.Context, id uuid.UUID) (*MusicBrainzRelease, error)
	Shutdown()
}

func NewMusicBrainzClient() *MusicBrainzClient {
	ret := new(MusicBrainzClient)
	ret.url = cfg.MusicBrainzUrl()
	ret.userAgent = cfg.UserAgent()
	ret.requestQueue = queue.NewRequestQueue(cfg.MusicBrainzRateLimit(), cfg.MusicBrainzRateLimit())
	return ret
}

func (c *MusicBrainzClient) Shutdown() {
	c.requestQueue.Shutdown()
}

func (c *MusicBrainzClient) getEntity(ctx context.Context, fmtStr string, id uuid.UUID, result any) error {
	l := logger.FromContext(ctx)
	url := fmt.Sprintf(fmtStr, c.url, id.String())
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		l.Err(err).Msg("Failed to build MusicBrainz request")
		return fmt.Errorf("getEntity: %w", err)
	}
	l.Debug().Msg("Adding MusicBrainz request to queue")
	body, err := c.queue(ctx, req)
	if err != nil {
		l.Err(err).Msg("MusicBrainz request failed")
		return fmt.Errorf("getEntity: %w", err)
	}

	err = json.Unmarshal(body, result)
	if err != nil {
		l.Err(err).Str("body", string(body)).Msg("Failed to unmarshal MusicBrainz response body")
		return fmt.Errorf("getEntity: %w", err)
	}

	return nil
}

// queue propagates caller and shutdown cancellation through the HTTP request.
func (c *MusicBrainzClient) queue(ctx context.Context, req *http.Request) ([]byte, error) {
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
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return queue.RequestResult{Err: fmt.Errorf("MusicBrainz returned HTTP %d", resp.StatusCode)}
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, (8<<20)+1))
		if err == nil && len(body) > 8<<20 {
			err = fmt.Errorf("MusicBrainz response exceeds 8 MiB")
		}
		return queue.RequestResult{Body: body, Err: err}
	})
	return result.Body, result.Err
}
