package importer

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/export"
	"github.com/ebrgr/TuneTrail/internal/imagefetch"
	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/google/uuid"
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
		// Validate the optional playback measurement before restoring the archive.
		if id == nil || id.EventID == "" || id.EventNamespace == "" || id.CatalogNamespace == "" ||
			id.OccurrenceID <= 0 || id.RecordingID <= 0 || releaseID < 0 || id.Origin == "" ||
			len(id.Provenance) == 0 || id.Provenance[0] != '[' || !json.Valid(id.Provenance) || l.ListenedAt.IsZero() ||
			(l.DurationPlayed != nil && (*l.DurationPlayed < 0 || *l.DurationPlayed > math.MaxInt32)) ||
			l.Track.Duration < 0 || l.Track.Duration > math.MaxInt32 || l.Track.DiscNumber < 0 || l.Track.TrackNumber < 0 ||
			getPrimaryAliasFromAliasSlice(l.Track.Aliases) == "" ||
			(releaseID > 0 && getPrimaryAliasFromAliasSlice(l.Album.Aliases) == "") {
			return invalidFile(fmt.Errorf("invalid TuneTrail archive item"))
		}
		if l.Track.MBIDSource != "" && l.Track.MBIDSource != "validated" && l.Track.MBIDSource != "legacy_declared" {
			source, at := ingest.DeclaredMBIDProvenance(l.Track.MBIDSource, l.Track.MBIDDeclaredAt)
			if source == "" || at == "" || l.Track.MBID == nil {
				return invalidFile(fmt.Errorf("invalid recording MBID provenance"))
			}
		}
		for _, declaration := range l.Track.MBIDDeclarations {
			source, at := ingest.DeclaredMBIDProvenance(declaration.Source, declaration.DeclaredAt)
			if source == "" || at == "" || declaration.MBID == uuid.Nil || len(declaration.Instance) > 1024 {
				return invalidFile(fmt.Errorf("invalid recording MBID declaration"))
			}
		}
		item := &db.ExportItem{
			EventID: id.EventID, EventNamespace: id.EventNamespace, CatalogNamespace: id.CatalogNamespace,
			TrackID: id.OccurrenceID, RecordingID: id.RecordingID, ReleaseID: releaseID,
			Origin: id.Origin, Provenance: id.Provenance, ListenedAt: l.ListenedAt, Client: &l.Client,
			TrackMbid: l.Track.MBID, TrackReference: l.Track.TrackMBID, DiscNumber: l.Track.DiscNumber,
			TrackNumber: l.Track.TrackNumber, MBIDSource: l.Track.MBIDSource,
			MBIDDeclaredAt:        l.Track.MBIDDeclaredAt,
			RecordingDeclarations: l.Track.MBIDDeclarations,
			DurationPlayed:        l.DurationPlayed, TrackDuration: int32(l.Track.Duration), TrackAliases: l.Track.Aliases,
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
		source, at := ingest.DeclaredMBIDProvenance(a.MBIDSource, a.MBIDDeclaredAt)
		if (a.MBIDSource != "" && (source == "" || at == "" || a.MBID == nil)) || (a.MBIDSource == "" && a.MBIDDeclaredAt != "") {
			return nil, invalidFile(fmt.Errorf("invalid artist MBID provenance"))
		}
		for _, declaration := range a.MBIDDeclarations {
			source, at := ingest.DeclaredMBIDProvenance(declaration.Source, declaration.DeclaredAt)
			if source == "" || at == "" || declaration.MBID == uuid.Nil || len(declaration.Instance) > 1024 {
				return nil, invalidFile(fmt.Errorf("invalid artist MBID declaration"))
			}
		}
		credits = append(credits, models.ArtistWithFullAliases{
			ID: a.ID, Name: name, MbzID: a.MBID, Aliases: a.Aliases, IsPrimary: a.IsPrimary,
			MBIDSource: source, MBIDDeclaredAt: at,
			MBIDDeclarations: a.MBIDDeclarations,
			ImageSource:      imagefetch.PublicSource(a.ImageUrl),
		})
	}
	return credits, nil
}
