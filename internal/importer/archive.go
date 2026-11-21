package importer

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/export"
	"github.com/ebrgr/TuneTrail/internal/imagefetch"
	"github.com/ebrgr/TuneTrail/internal/models"
)

type archiveRestorer interface {
	RestoreArchive(context.Context, int32, []*db.ExportItem) (int, error)
}

// importArchive restores accepted history by event identity, independently of ingestion flags.
func importArchive(ctx context.Context, store importStore, profileID int32, data *export.KoitoExport, filename string) error {
	restorer, ok := store.(archiveRestorer)
	if !ok {
		return fmt.Errorf("archive restore is unavailable")
	}
	items := make([]*db.ExportItem, 0, len(data.Listens))
	for _, l := range data.Listens {
		id := l.Identity
		var releaseID int32
		if id != nil && id.ReleaseID != nil {
			releaseID = *id.ReleaseID
		}
		if id == nil || id.EventID == "" || id.EventNamespace == "" || id.CatalogNamespace == "" ||
			id.OccurrenceID <= 0 || id.RecordingID <= 0 || releaseID < 0 || id.Origin == "" ||
			len(id.Provenance) == 0 || id.Provenance[0] != '[' || !json.Valid(id.Provenance) || l.ListenedAt.IsZero() ||
			l.Track.Duration < 0 || l.Track.Duration > math.MaxInt32 || l.Track.DiscNumber < 0 || l.Track.TrackNumber < 0 ||
			getPrimaryAliasFromAliasSlice(l.Track.Aliases) == "" ||
			(releaseID > 0 && getPrimaryAliasFromAliasSlice(l.Album.Aliases) == "") {
			return invalidFile(fmt.Errorf("invalid TuneTrail archive item"))
		}
		item := &db.ExportItem{
			EventID: id.EventID, EventNamespace: id.EventNamespace, CatalogNamespace: id.CatalogNamespace,
			TrackID: id.OccurrenceID, RecordingID: id.RecordingID, ReleaseID: releaseID,
			Origin: id.Origin, Provenance: id.Provenance, ListenedAt: l.ListenedAt, Client: &l.Client,
			TrackMbid: l.Track.MBID, TrackReference: l.Track.TrackMBID, DiscNumber: l.Track.DiscNumber,
			TrackNumber: l.Track.TrackNumber, MBIDSource: l.Track.MBIDSource,
			TrackDuration: int32(l.Track.Duration), TrackAliases: l.Track.Aliases,
			ReleaseMbid: l.Album.MBID, VariousArtists: l.Album.VariousArtists, ReleaseAliases: l.Album.Aliases,
			ReleaseImageSource: imagefetch.PublicSource(l.Album.ImageUrl),
		}
		var err error
		item.Artists, err = archiveCredits(l.Artists, releaseID > 0)
		if err != nil {
			return err
		}
		item.ReleaseArtists, err = archiveCredits(l.Album.Artists, true)
		if err != nil {
			return err
		}
		seen(ctx)
		if !inImportTimeWindow(l.ListenedAt) {
			ignored(ctx)
			continue
		}
		items = append(items, item)
	}
	count, err := restorer.RestoreArchive(ctx, profileID, items)
	if err != nil {
		return fmt.Errorf("restore TuneTrail archive: %w", err)
	}
	return finishImport(ctx, filename, count)
}

// archiveCredits validates visible names while preserving separate artist identities.
func archiveCredits(artists []export.KoitoArtist, identified bool) ([]models.ArtistWithFullAliases, error) {
	var credits []models.ArtistWithFullAliases
	for _, a := range artists {
		name := getPrimaryAliasFromAliasSlice(a.Aliases)
		if name == "" || (identified && a.ID <= 0) {
			return nil, invalidFile(fmt.Errorf("invalid TuneTrail archive artist"))
		}
		credits = append(credits, models.ArtistWithFullAliases{
			ID: a.ID, Name: name, MbzID: a.MBID, Aliases: a.Aliases, IsPrimary: a.IsPrimary,
			ImageSource: imagefetch.PublicSource(a.ImageUrl),
		})
	}
	return credits, nil
}
