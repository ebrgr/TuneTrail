package export

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/logger"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/google/uuid"
)

type KoitoExport struct {
	Version    string        `json:"version"`
	ExportedAt time.Time     `json:"exported_at"` // RFC3339
	User       string        `json:"user"`        // username
	Listens    []KoitoListen `json:"listens"`
}

const ArchiveVersion = "2"

type ArchiveIdentity struct {
	EventID          string          `json:"event_id"`
	EventNamespace   string          `json:"event_namespace"`
	CatalogNamespace string          `json:"catalog_namespace"`
	OccurrenceID     int32           `json:"occurrence_id"`
	RecordingID      int64           `json:"recording_id"`
	ReleaseID        *int32          `json:"release_id"`
	Origin           string          `json:"origin"`
	Provenance       json.RawMessage `json:"provenance"`
}

type KoitoListen struct {
	AliasFieldOrder    []string         `json:"-"`
	Identity           *ArchiveIdentity `json:"identity,omitempty"`
	OriginalListenedAt string           `json:"-"`
	ListenedAt         time.Time        `json:"listened_at"`
	DurationPlayed     *int64           `json:"duration_played,omitempty"`
	Client             string           `json:"client"`
	Track              KoitoTrack       `json:"track"`
	Album              KoitoAlbum       `json:"album"`
	Artists            []KoitoArtist    `json:"artists"`
}

// UnmarshalJSON retains the original timestamp for occurrence import receipts.
func (item *KoitoListen) UnmarshalJSON(data []byte) error {
	type listen KoitoListen
	var value struct {
		*listen
		Time string `json:"listened_at"`
	}
	value.listen = (*listen)(item)
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	// Keep the order of alias-bearing fields for the bounded v1 import snapshot.
	fields := json.NewDecoder(bytes.NewReader(data))
	if _, err := fields.Token(); err != nil {
		return err
	}
	for fields.More() {
		token, err := fields.Token()
		if err != nil {
			return err
		}
		key, _ := token.(string)
		switch key {
		case "track", "album", "artists":
			item.AliasFieldOrder = append(item.AliasFieldOrder, key)
		}
		var ignored json.RawMessage
		if err := fields.Decode(&ignored); err != nil {
			return err
		}
	}
	item.OriginalListenedAt = value.Time
	if value.Time == "" {
		return nil
	}
	at, err := time.Parse(time.RFC3339Nano, value.Time)
	item.ListenedAt = at
	return err
}

type KoitoTrack struct {
	TrackMBID      string         `json:"track_mbid,omitempty"`
	DiscNumber     int            `json:"disc_number,omitempty"`
	TrackNumber    int            `json:"track_number,omitempty"`
	MBIDSource     string         `json:"mbid_source,omitempty"`
	MBIDDeclaredAt string         `json:"mbid_declared_at,omitempty"`
	MBID           *uuid.UUID     `json:"mbid"`
	Duration       int            `json:"duration"`
	Aliases        []models.Alias `json:"aliases"`
}
type KoitoAlbum struct {
	Artists        []KoitoArtist  `json:"artists,omitempty"`
	ImageUrl       string         `json:"image_url"`
	MBID           *uuid.UUID     `json:"mbid"`
	Aliases        []models.Alias `json:"aliases"`
	VariousArtists bool           `json:"various_artists"`
}
type KoitoArtist struct {
	ID        int32          `json:"id,omitempty"`
	ImageUrl  string         `json:"image_url"`
	MBID      *uuid.UUID     `json:"mbid"`
	IsPrimary bool           `json:"is_primary"`
	Aliases   []models.Alias `json:"aliases"`
}

// checkedWriter converts malformed short writes into an observable stream error.
type checkedWriter struct{ io.Writer }

func (w checkedWriter) Write(p []byte) (int, error) {
	n, err := w.Writer.Write(p)
	if err == nil && n != len(p) {
		err = io.ErrShortWrite
	}
	return n, err
}

// ExportData writes the current TuneTrail archive.
func ExportData(ctx context.Context, profile *models.ListeningProfile, store db.ExportStore, out io.Writer) error {
	return ExportVersion(ctx, profile, store, out, ArchiveVersion)
}

// ExportVersion writes the selected wire format using the same event cursor.
func ExportVersion(ctx context.Context, profile *models.ListeningProfile, store db.ExportStore, out io.Writer, version string) error {
	if version != "1" && version != ArchiveVersion {
		return db.ErrInvalidInput
	}
	out = checkedWriter{out}
	var cursor string
	var ceiling int64
	pageSize := int32(1000)

	l := logger.FromContext(ctx)
	// Version 2 adds native identities while retaining the Koito presentation fields.
	l.Info().Msg("ExportData: generating TuneTrail archive")
	exportedAt := time.Now()

	encodedName, err := json.Marshal(profile.DisplayName)
	if err != nil {
		return err
	}
	// Write the opening of the JSON manually
	_, err = fmt.Fprintf(out, "{\n  \"version\": \""+version+"\",\n  \"exported_at\": \"%s\",\n  \"user\": %s,\n  \"listens\": [\n", exportedAt.UTC().Format(time.RFC3339), encodedName)
	if err != nil {
		return fmt.Errorf("ExportData: %w", err)
	}

	first := true
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		rows, err := store.GetExportPage(ctx, db.GetExportPageOpts{
			ProfileID:       profile.ID,
			AfterEventID:    cursor,
			ThroughSequence: ceiling,
			Limit:           pageSize,
		})
		if err != nil {
			return fmt.Errorf("ExportData: %w", err)
		}
		if len(rows) == 0 {
			break
		}

		// Both wire versions reuse the first page's insertion ceiling.
		ceiling = rows[0].ThroughSequence
		for _, r := range rows {
			if err := ctx.Err(); err != nil {
				return err
			}
			cursor = r.Cursor
			// The legacy format requires an album and has no event identity fields.
			if version == "1" && r.ReleaseID == 0 {
				continue
			}
			// Adds a comma after each listen item
			if !first {
				if _, err := io.WriteString(out, ",\n"); err != nil {
					return fmt.Errorf("ExportData: %w", err)
				}
			}
			first = false

			exported := convertToExportFormat(r)
			if version == "1" {
				exported.Identity = nil
				exported.Album.Artists = nil
				exported.Track.TrackMBID = ""
				exported.Track.DiscNumber, exported.Track.TrackNumber = 0, 0
				exported.Track.MBIDSource = ""
				exported.Track.MBIDDeclaredAt = ""
				for i := range exported.Artists {
					exported.Artists[i].ID = 0
				}
			}

			raw, err := json.MarshalIndent(exported, "    ", "  ")

			// needed to make the listen item start at the right indent level
			if _, err := io.WriteString(out, "    "); err != nil {
				return fmt.Errorf("ExportData: %w", err)
			}

			if err != nil {
				return fmt.Errorf("ExportData: marshal: %w", err)
			}
			if _, err := out.Write(raw); err != nil {
				return fmt.Errorf("ExportData: %w", err)
			}
		}
	}

	// Write closing of the JSON array and object
	_, err = out.Write([]byte("\n  ]\n}\n"))
	if err != nil {
		return fmt.Errorf("ExportData: f.Write: %w", err)
	}

	l.Info().Msgf("Export successfully created")
	return nil
}

// convertToExportFormat carries scoped recording declarations into the archive.
func convertToExportFormat(item *db.ExportItem) *KoitoListen {
	var client string
	if item.Client != nil {
		client = *item.Client
	}
	ret := &KoitoListen{
		Identity: &ArchiveIdentity{
			EventID: item.EventID, EventNamespace: item.EventNamespace, CatalogNamespace: item.CatalogNamespace,
			OccurrenceID: item.TrackID, RecordingID: item.RecordingID, Origin: item.Origin,
			Provenance: item.Provenance,
		},
		// A missing value stays absent in archives for compatibility with older exports.
		ListenedAt: item.ListenedAt.UTC(), DurationPlayed: item.DurationPlayed,
		Track: KoitoTrack{
			MBID:      item.TrackMbid,
			TrackMBID: item.TrackReference, DiscNumber: item.DiscNumber, TrackNumber: item.TrackNumber, MBIDSource: item.MBIDSource, MBIDDeclaredAt: item.MBIDDeclaredAt,
			Duration: int(item.TrackDuration),
			Aliases:  item.TrackAliases,
		},
		Client: client,
		Album: KoitoAlbum{
			MBID:           item.ReleaseMbid,
			ImageUrl:       item.ReleaseImageSource,
			VariousArtists: item.VariousArtists,
			Aliases:        item.ReleaseAliases,
		},
	}
	if item.ReleaseID != 0 {
		ret.Identity.ReleaseID = &item.ReleaseID
	}
	ret.Artists = exportCredits(item.Artists)
	ret.Album.Artists = exportCredits(item.ReleaseArtists)
	return ret
}

// exportCredits serializes album and occurrence credits with independent roles.
func exportCredits(artists []models.ArtistWithFullAliases) []KoitoArtist {
	var credits []KoitoArtist
	for _, artist := range artists {
		credits = append(credits, KoitoArtist{
			ID: artist.ID, IsPrimary: artist.IsPrimary, MBID: artist.MbzID,
			Aliases: artist.Aliases, ImageUrl: artist.ImageSource,
		})
	}
	return credits
}
