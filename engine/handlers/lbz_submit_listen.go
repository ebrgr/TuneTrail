package handlers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ebrgr/TuneTrail/engine/middleware"
	"github.com/ebrgr/TuneTrail/internal/catalog"
	"github.com/ebrgr/TuneTrail/internal/cfg"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/ebrgr/TuneTrail/internal/listenbrainz"
	"github.com/ebrgr/TuneTrail/internal/logger"
	"github.com/ebrgr/TuneTrail/internal/mbz"
	"github.com/ebrgr/TuneTrail/internal/relay"
	"github.com/ebrgr/TuneTrail/internal/utils"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"golang.org/x/sync/singleflight"
)

type LbzListenType string

const (
	ListenTypeSingle     LbzListenType = "single"
	ListenTypePlayingNow LbzListenType = "playing_now"
	ListenTypeImport     LbzListenType = "import"
)

type LbzSubmitListenRequest struct {
	ListenType      LbzListenType            `json:"listen_type,omitempty"`
	Payload         []LbzSubmitListenPayload `json:"payload,omitempty"`
	oversizedListen bool
	trailingJSON    bool
}

type LbzSubmitListenPayload = listenbrainz.LbzSubmitListenPayload
type LbzTrackMeta = listenbrainz.LbzTrackMeta
type LbzArtist = listenbrainz.LbzArtist
type LbzMBIDMapping = listenbrainz.LbzMBIDMapping
type LbzAdditionalInfo = listenbrainz.LbzAdditionalInfo

const (
	maxListensPerRequest = 1000
	maxListenBodyBytes   = 16 << 20
)

var sfGroup singleflight.Group
var errTooManyListens = errors.New("payload exceeds maximum listen count")

// decodeListenRequest caps the array before allocating another typed payload.
func decodeListenRequest(reader io.Reader) (LbzSubmitListenRequest, error) {
	var result LbzSubmitListenRequest
	dec := json.NewDecoder(reader)
	token, err := dec.Token()
	if err != nil {
		return result, err
	}
	if token == nil {
		return result, nil
	}
	if token != json.Delim('{') {
		return result, errors.New("expected request object")
	}
	for dec.More() {
		key, err := dec.Token()
		if err != nil {
			return result, err
		}
		name, ok := key.(string)
		if !ok {
			return result, errors.New("expected field name")
		}
		switch {
		case strings.EqualFold(name, "listen_type"):
			if err := dec.Decode(&result.ListenType); err != nil {
				return result, err
			}
		case strings.EqualFold(name, "payload"):
			t, err := dec.Token()
			if err != nil {
				return result, err
			}
			result.Payload = nil
			result.oversizedListen = false
			if t == nil {
				continue
			}
			if t != json.Delim('[') {
				return result, errors.New("expected payload array")
			}
			for dec.More() {
				if len(result.Payload) >= maxListensPerRequest {
					return result, errTooManyListens
				}
				start := dec.InputOffset()
				var item LbzSubmitListenPayload
				if err := dec.Decode(&item); err != nil {
					return result, err
				}
				if raw, ok := reader.(*bytes.Reader); ok {
					// Exclude array separators and inter-item whitespace from the size.
					size := dec.InputOffset() - start
					section := io.NewSectionReader(raw, start, size)
					if size > 10<<10 {
						actual, _ := io.ReadAll(section) // bytes.Reader cannot fail.
						result.oversizedListen = result.oversizedListen || len(bytes.TrimLeft(actual, ", \t\r\n")) > 10<<10
					}
				}
				result.Payload = append(result.Payload, item)
			}
			if _, err := dec.Token(); err != nil {
				return result, err
			}
		default:
			var ignored json.RawMessage
			if err := dec.Decode(&ignored); err != nil {
				return result, err
			}
		}
	}
	_, err = dec.Token()
	if err == nil {
		// Report trailing data only to active admission, preserving legacy decoding.
		_, trailing := dec.Token()
		result.trailingJSON = !errors.Is(trailing, io.EOF)
	}
	return result, err
}

type submitListenHandlerStore interface {
	db.ArtistStore
	db.AlbumStore
	db.TrackStore
	db.ListenStore
}

// LbzSubmitListenHandler bounds raw bodies before buffering, decoding or relaying them.
func LbzSubmitListenHandler(store submitListenHandlerStore, mbzc mbz.MusicBrainzCaller) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		receivedAt := time.Now()
		l := logger.FromContext(r.Context())

		l.Debug().Msg("LbzSubmitListenHandler: Received request to submit listens")

		r.Body = http.MaxBytesReader(w, r.Body, maxListenBodyBytes)
		requestBytes, err := io.ReadAll(r.Body)
		if err != nil {
			var limit *http.MaxBytesError
			if errors.As(err, &limit) {
				utils.WriteError(w, "listen request exceeds 16 MiB", http.StatusRequestEntityTooLarge)
				return
			}
			l.Err(err).Msg("LbzSubmitListenHandler: Failed to read request body")
			utils.WriteError(w, "failed to read request body", http.StatusBadRequest)
			return
		}

		req, err := decodeListenRequest(bytes.NewReader(requestBytes))
		if errors.Is(err, errTooManyListens) {
			utils.WriteError(w, "payload exceeds max listens per request", http.StatusBadRequest)
			return
		}
		if err != nil {
			l.Err(err).Msg("LbzSubmitListenHandler: Failed to decode request")
			utils.WriteError(w, "failed to decode request", http.StatusBadRequest)
			return
		}

		profile := middleware.ProfileFromContext(r.Context())
		if profile == nil {
			l.Debug().Msg("LbzSubmitListenHandler: Unauthorized request (user context is nil)")
			utils.WriteError(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		// Log only bounded counts; the payload may contain personal or sensitive metadata.
		l.Info().Int("listen_count", len(req.Payload)).Msg("LbzSubmitListenHandler: Parsed request")

		if len(req.Payload) < 1 {
			l.Debug().Msg("LbzSubmitListenHandler: Payload is empty")
			utils.WriteError(w, "payload is nil", http.StatusBadRequest)
			return
		}

		if len(req.Payload) > maxListensPerRequest {
			l.Debug().Msgf("LbzSubmitListenHandler: Payload exceeds max listens per request (%d > %d)", len(req.Payload), maxListensPerRequest)
			utils.WriteError(w, "payload exceeds max listens per request", http.StatusBadRequest)
			return
		}

		if len(req.Payload) != 1 && req.ListenType != "import" {
			l.Debug().Msg("LbzSubmitListenHandler: Payload must only contain one listen for non-import requests")
			utils.WriteError(w, "payload must only contain one listen for non-import requests", http.StatusBadRequest)
			return
		}

		// The active path validates and commits the entire batch before acknowledging.
		if receiver, ok := store.(ingest.Receiver); ok && req.ListenType != ListenTypePlayingNow {
			batch, invalid := listenIngestBatch(profile.ID, req, r.Header.Get("Idempotency-Key"), receivedAt)
			var active bool
			var err error
			if admission, ok := store.(ingest.Admission); ok {
				active, _, err = admission.AdmitIngest(r.Context(), batch, cfg.ActiveIngestEnabled(), nil, invalid)
			} else {
				_, err = receiver.ReceiveBatch(r.Context(), batch)
			}
			var unavailable *ingest.ErrUnavailable
			switch {
			case errors.As(err, &unavailable):
				l.Warn().Msg("ListenBrainz receipt storage unavailable")
				if active {
					w.Header().Set("Retry-After", "1")
				}
				utils.WriteError(w, "listen receipt storage unavailable", http.StatusServiceUnavailable)
				return
			case active && invalid != nil && errors.Is(err, invalid):
				utils.WriteError(w, invalid.Error(), http.StatusBadRequest)
				return
			case active && err != nil:
				l.Warn().Msg("ListenBrainz active receipt failed")
				utils.WriteError(w, "listen receipt failed", http.StatusInternalServerError)
				return
			case active:
				writeListenOK(w)
				relayListenRequest(r.Context(), store, profile.ID, requestBytes, l)
				return
			case err != nil:
				l.Warn().Msg("ListenBrainz shadow receipt failed")
			}
		}

		for _, payload := range req.Payload {
			if payload.TrackMeta.ArtistName == "" || payload.TrackMeta.TrackName == "" {
				l.Debug().Msg("LbzSubmitListenHandler: Artist name or track name are missing")
				utils.WriteError(w, "Artist name or track name are missing", http.StatusBadRequest)
				return
			}

			if req.ListenType != ListenTypePlayingNow && req.ListenType != ListenTypeSingle && req.ListenType != ListenTypeImport {
				l.Debug().Msg("LbzSubmitListenHandler: No listen type provided, assuming 'single'")
				req.ListenType = "single"
			}

			artistMbzIDs, err := utils.ParseUUIDSlice(payload.TrackMeta.AdditionalInfo.ArtistMBIDs)
			if err != nil {
				l.Debug().AnErr("error", err).Msg("LbzSubmitListenHandler: Failed to parse one or more UUIDs")
			}
			if len(artistMbzIDs) < 1 {
				l.Debug().AnErr("error", err).Msg("LbzSubmitListenHandler: Attempting to parse artist UUIDs from mbid_mapping")
				utils.ParseUUIDSlice(payload.TrackMeta.MBIDMapping.ArtistMBIDs)
				if err != nil {
					l.Debug().AnErr("error", err).Msg("LbzSubmitListenHandler: Failed to parse one or more UUIDs")
				}
			}
			rgMbzID, err := uuid.Parse(payload.TrackMeta.AdditionalInfo.ReleaseGroupMBID)
			if err != nil {
				rgMbzID = uuid.Nil
			}
			releaseMbzID, err := uuid.Parse(payload.TrackMeta.AdditionalInfo.ReleaseMBID)
			if err != nil {
				releaseMbzID, err = uuid.Parse(payload.TrackMeta.MBIDMapping.ReleaseMBID)
				if err != nil {
					releaseMbzID = uuid.Nil
				}
			}
			recordingMbzID, err := uuid.Parse(payload.TrackMeta.AdditionalInfo.RecordingMBID)
			if err != nil {
				recordingMbzID, err = uuid.Parse(payload.TrackMeta.MBIDMapping.RecordingMBID)
				if err != nil {
					recordingMbzID = uuid.Nil
				}
			}

			var client string
			if payload.TrackMeta.AdditionalInfo.MediaPlayer != "" {
				client = payload.TrackMeta.AdditionalInfo.MediaPlayer
			} else if payload.TrackMeta.AdditionalInfo.SubmissionClient != "" {
				client = payload.TrackMeta.AdditionalInfo.SubmissionClient
			}

			var duration int32
			if payload.TrackMeta.AdditionalInfo.Duration != 0 {
				duration = payload.TrackMeta.AdditionalInfo.Duration
			} else if payload.TrackMeta.AdditionalInfo.DurationMs != 0 {
				duration = payload.TrackMeta.AdditionalInfo.DurationMs / 1000
			}

			var listenedAt = time.Now()
			if payload.ListenedAt != 0 {
				listenedAt = time.Unix(int64(payload.ListenedAt), 0)
			}

			var artistMbidMap []catalog.ArtistMbidMap
			for _, a := range payload.TrackMeta.MBIDMapping.Artists {
				if a.ArtistMBID == "" || a.ArtistName == "" {
					continue
				}
				mbid, err := uuid.Parse(a.ArtistMBID)
				if err != nil {
					l.Debug().AnErr("error", err).Msgf("LbzSubmitListenHandler: Failed to parse UUID for artist '%s'", a.ArtistName)
				}
				artistMbidMap = append(artistMbidMap, catalog.ArtistMbidMap{Artist: a.ArtistName, Mbid: mbid})
			}

			opts := catalog.SubmitListenOpts{
				MbzCaller:          mbzc,
				ArtistNames:        payload.TrackMeta.AdditionalInfo.ArtistNames,
				Artist:             payload.TrackMeta.ArtistName,
				ArtistMbzIDs:       artistMbzIDs,
				TrackTitle:         payload.TrackMeta.TrackName,
				RecordingMbzID:     recordingMbzID,
				ReleaseTitle:       payload.TrackMeta.ReleaseName,
				ReleaseMbzID:       releaseMbzID,
				ReleaseGroupMbzID:  rgMbzID,
				ArtistMbidMappings: artistMbidMap,
				Duration:           duration,
				Time:               listenedAt,
				ProfileID:          profile.ID,
				Client:             client,
				IsNowPlaying:       req.ListenType == ListenTypePlayingNow,
				SkipSaveListen:     req.ListenType == ListenTypePlayingNow,
			}

			_, err, shared := sfGroup.Do(fmt.Sprintf("%d:%s:%d:%s", profile.ID, req.ListenType, listenedAt.Unix(), buildCaolescingKey(payload)), func() (interface{}, error) {
				return 0, catalog.SubmitListen(r.Context(), store, opts)
			})
			if shared {
				l.Info().Msg("LbzSubmitListenHandler: Duplicate requests detected; results were coalesced")
			}
			if err != nil {
				l.Err(err).Msg("LbzSubmitListenHandler: Failed to submit listen")
				// Set the content type before committing the response status.
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("{\"status\": \"internal server error\"}"))
				return
			}
		}

		relayListenRequest(r.Context(), store, profile.ID, requestBytes, l)
		l.Debug().Msg("LbzSubmitListenHandler: Successfully processed listens")
		writeListenOK(w)
	}
}

// writeListenOK preserves the ListenBrainz response body and status.
func writeListenOK(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{\"status\": \"ok\"}"))
}

// relayListenRequest reads the current profile destination and dispatches the raw payload.
func relayListenRequest(ctx context.Context, store submitListenHandlerStore, profile int32, body []byte, l *zerolog.Logger) {
	if relayStore, ok := store.(db.ProfileStore); ok {
		url, token, err := relayStore.GetProfileRelay(ctx, profile)
		if err == nil && url != "" && token != "" {
			go doLbzRelay(body, l, url, token)
		}
	}
}

// doLbzRelay uses the same bounded transport policy as the administrator's relay test.
func doLbzRelay(requestBytes []byte, l *zerolog.Logger, relayURL, relayToken string) {
	if err := (relay.Sender{}).Send(context.Background(), relayURL, relayToken, requestBytes); err != nil {
		l.Warn().Msg("ListenBrainz relay submission failed")
	}
}

// buildCaolescingKey separates metadata fields and includes all typed payload data.
func buildCaolescingKey(p LbzSubmitListenPayload) string {
	// The payload contains only JSON-supported strings, integers and slices.
	data, _ := json.Marshal(p)
	return fmt.Sprintf("%x", sha256.Sum256(data))
}

// listenIngestBatch supplies the legacy prefix and the validation result for active admission.
func listenIngestBatch(profile int32, req LbzSubmitListenRequest, key string, receivedAt time.Time) (ingest.Batch, error) {
	operation := string(ListenTypeSingle)
	if req.ListenType == ListenTypeImport {
		operation = string(ListenTypeImport)
	}
	batch := ingest.Batch{ProfileID: profile, Source: ingest.ShadowSourceListenBrainz, Operation: operation, ReceivedAt: receivedAt}
	if key != "" {
		batch.ClientKey = ingest.Digest([]byte(key))
	}
	var invalid error
	if req.oversizedListen {
		invalid = errors.New("listen exceeds 10 KiB")
	}
	if req.trailingJSON {
		invalid = errors.New("unexpected data after listen request")
	}
	for ordinal, payload := range req.Payload {
		meta := payload.TrackMeta
		if meta.ArtistName == "" || meta.TrackName == "" {
			return batch, errors.New("Artist name or track name are missing")
		}
		item := ingest.Item{Ordinal: ordinal, Snapshot: ingest.ListenBrainzSnapshot(meta)}
		if payload.ListenedAt != 0 {
			item.ListenedAt = time.Unix(int64(payload.ListenedAt), 0)
		}
		batch.Items = append(batch.Items, item)
	}
	return batch, invalid
}
