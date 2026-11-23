package importer

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/export"
	"github.com/ebrgr/TuneTrail/internal/imagefetch"
	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/ebrgr/TuneTrail/internal/logger"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/google/uuid"
	"math"
)

// ImportKoitoFile preserves owner-import provenance through catalog writes and image downloads.
func ImportKoitoFile(ctx context.Context, store importStore, filename string) error {
	if db.ProfileID(ctx) <= 0 {
		return fmt.Errorf("explicit import profile required")
	}
	profileID := db.ProfileID(ctx)
	ctx = db.WithoutProfile(ctx)
	if untrustedImport(ctx) {
		ctx = db.WithImportedCatalog(ctx, profileID)
	}
	l := logger.FromContext(ctx)
	l.Info().Msgf("Beginning TuneTrail/Koito import on file: %s", filename) // same JSON schema (ISSUE-111)
	data := new(export.KoitoExport)
	f, err := os.Open(importPath(ctx, filename))
	if err != nil {
		return fmt.Errorf("ImportKoitoFile: os.Open: %w", err)
	}
	defer f.Close()
	err = decodeJSON(ctx, f, data)
	if err != nil {
		return fmt.Errorf("ImportKoitoFile: Decode: %w", err)
	}

	// Native archives bypass heuristic history matching. Koito v1 keeps its existing path.
	if len(data.Listens) > MaxImportItems {
		return invalidFile(fmt.Errorf("too many import items"))
	}
	if data.Version == export.ArchiveVersion {
		return importArchive(ctx, store, profileID, data, filename)
	}
	if data.Version != "1" {
		return invalidFile(fmt.Errorf("unsupported TuneTrail/Koito export version"))
	}
	ctx, err = withFileIngest(ctx, store, profileID, "koito", f)
	if err != nil {
		return err
	}
	// Validate all required identities before writing anything. Uploaded exports
	// are untrusted and an empty artist list must not panic at artistIds[0].
	for _, item := range data.Listens {
		if err := ctx.Err(); err != nil {
			return err
		}
		if item.Track.Duration < 0 || item.Track.Duration > math.MaxInt32 ||
			(item.DurationPlayed != nil && (*item.DurationPlayed < 0 || *item.DurationPlayed > math.MaxInt32)) {
			return invalidFile(fmt.Errorf("invalid track duration"))
		}
		if len(item.Artists) == 0 || getPrimaryAliasFromAliasSlice(item.Track.Aliases) == "" {
			return invalidFile(fmt.Errorf("invalid TuneTrail/Koito export: artists and track required"))
		}
		for _, artist := range item.Artists {
			if getPrimaryAliasFromAliasSlice(artist.Aliases) == "" {
				return invalidFile(fmt.Errorf("invalid TuneTrail/Koito export: artist name required"))
			}
		}
	}
	l.Info().Msg("Beginning TuneTrail/Koito data import")

	for i := range data.Listens {
		seen(ctx)
		if err := ctx.Err(); err != nil {
			return err
		}
		if !inImportTimeWindow(data.Listens[i].ListenedAt) {
			l.Debug().Msgf("Skipping import due to import time rules")
			ignored(ctx)
			continue
		}
		item := data.Listens[i]
		snapshot := ingest.Snapshot{
			Artist:  getPrimaryAliasFromAliasSlice(item.Artists[0].Aliases),
			Title:   getPrimaryAliasFromAliasSlice(item.Track.Aliases),
			Release: getPrimaryAliasFromAliasSlice(item.Album.Aliases),
			// Preserve optional playback time in legacy-format imports too.
			DurationPlayed: item.DurationPlayed,
			DurationMS:     int64(item.Track.Duration) * 1000,
			Client:         item.Client,
			OriginalTime:   item.ListenedAt.Format(time.RFC3339Nano),
		}
		// Admit the observed native item before resolving or mutating catalog metadata.
		snapshot.OriginalTime = item.OriginalListenedAt
		snapshot.OwnerImport = untrustedImport(ctx)
		snapshot.NativeVariousArtists = item.Album.VariousArtists
		snapshot.NativeReleaseImage = imagefetch.PublicSource(item.Album.ImageUrl)
		snapshot.TrackMBID = item.Track.TrackMBID
		snapshot.DiscNumber, snapshot.TrackNumber = item.Track.DiscNumber, item.Track.TrackNumber
		snapshot.MBIDSource, snapshot.MBIDDeclaredAt = ingest.DeclaredMBIDProvenance(item.Track.MBIDSource, item.Track.MBIDDeclaredAt)
		if item.Track.MBID != nil && *item.Track.MBID != uuid.Nil {
			snapshot.Recording = item.Track.MBID.String()
		}
		if item.Album.MBID != nil && *item.Album.MBID != uuid.Nil {
			snapshot.ReleaseMBID = item.Album.MBID.String()
		}
		for _, artist := range item.Artists {
			name := getPrimaryAliasFromAliasSlice(artist.Aliases)
			snapshot.ArtistNames = append(snapshot.ArtistNames, name)
			if source := imagefetch.PublicSource(artist.ImageUrl); source != "" {
				if snapshot.NativeArtistImages == nil {
					snapshot.NativeArtistImages = make(map[string]string)
				}
				snapshot.NativeArtistImages[name] = source
			}
			if artist.MBID != nil && *artist.MBID != uuid.Nil {
				snapshot.ArtistMbidMappings = append(snapshot.ArtistMbidMappings, ingest.ArtistMapping{Name: name, MBID: artist.MBID.String()})
			}
		}
		if err := captureFileListen(ctx, snapshot, item.ListenedAt, item); err != nil {
			return err
		}
	}

	if err := flushFileIngest(ctx); err != nil {
		return err
	}
	return finishImport(ctx, filename, 0)
}
func getPrimaryAliasFromAliasSlice(aliases []models.Alias) string {
	for _, a := range aliases {
		if a.Primary {
			return a.Alias
		}
	}
	return ""
}

// budgetNativeAliases keeps the longest prefix that fits with the required snapshot fields.
func budgetNativeAliases(snapshot ingest.Snapshot, item export.KoitoListen) ingest.Snapshot {
	type entry struct{ field, artist, name string }
	var entries []entry
	add := func(field, artist string, aliases []models.Alias) {
		for _, alias := range aliases {
			if name := strings.TrimSpace(alias.Alias); !alias.Primary && name != "" {
				entries = append(entries, entry{field, artist, name})
			}
		}
	}
	seen := make(map[string]bool)
	for _, field := range item.AliasFieldOrder {
		if seen[field] {
			continue
		}
		seen[field] = true
		switch field {
		case "track":
			add(field, "", item.Track.Aliases)
		case "album":
			add(field, "", item.Album.Aliases)
		case "artists":
			for _, artist := range item.Artists {
				add(field, getPrimaryAliasFromAliasSlice(artist.Aliases), artist.Aliases)
			}
		}
	}
	if len(entries) == 0 {
		return snapshot
	}
	build := func(n int) ingest.Snapshot {
		bounded := snapshot
		bounded.IgnoredAliasLimit = len(entries) - n
		if n == 0 {
			return bounded
		}
		aliases := &ingest.NativeAliases{Artists: make(map[string][]string)}
		for _, alias := range entries[:n] {
			switch alias.field {
			case "track":
				aliases.Track = append(aliases.Track, alias.name)
			case "album":
				aliases.Release = append(aliases.Release, alias.name)
			case "artists":
				aliases.Artists[alias.artist] = append(aliases.Artists[alias.artist], alias.name)
			}
		}
		bounded.NativeAliases = aliases
		return bounded
	}
	// No encoded alias uses fewer than three bytes including JSON delimiters.
	upper := min(len(entries), ingest.MaxItemBytes/3)
	firstTooLarge := sort.Search(upper+1, func(n int) bool {
		encoded, _ := (ingest.Item{Snapshot: build(n), ListenedAt: item.ListenedAt}).Encode()
		return len(encoded) > ingest.MaxItemBytes
	})
	return build(max(0, firstTooLarge-1))
}
