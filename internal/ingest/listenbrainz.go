package ingest

import (
	"math"
	"strconv"
	"strings"

	"github.com/ebrgr/TuneTrail/internal/listenbrainz"
	"github.com/google/uuid"
)

// ListenBrainzSnapshot excludes arbitrary extensions, raw headers and URLs.
func ListenBrainzSnapshot(meta listenbrainz.LbzTrackMeta) Snapshot {
	info := meta.AdditionalInfo
	s := Snapshot{Artist: meta.ArtistName, Title: meta.TrackName, Release: meta.ReleaseName,
		ArtistNames: info.ArtistNames, DurationMS: int64(info.DurationMs), Client: info.SubmissionClient,
		ClientVersion: info.SubmissionClientVersion}
	// Prefer seconds when both units provide a usable playback time.
	if info.DurationPlayed != nil {
		s.DurationPlayed = PlayedSeconds(*info.DurationPlayed)
	}
	if s.DurationPlayed == nil && info.DurationPlayedMs != nil {
		s.DurationPlayed = PlayedSeconds(*info.DurationPlayedMs / 1000)
	}
	s.DiscNumber = listenBrainzPosition(info.DiscNumber)
	s.TrackNumber = listenBrainzPosition(info.TrackNumber)
	if id, err := uuid.Parse(info.TrackMBID); err == nil && id != uuid.Nil {
		s.TrackMBID = id.String()
	}
	s.MBIDSource, s.MBIDDeclaredAt = DeclaredMBIDProvenance(info.MBIDSource, info.MBIDDeclaredAt)
	s.ArtistMBIDs = validMBIDs(info.ArtistMBIDs)
	s.MappedArtistMBIDs = validMBIDs(meta.MBIDMapping.ArtistMBIDs)
	for _, credit := range meta.MBIDMapping.Artists {
		if id, err := uuid.Parse(credit.ArtistMBID); err == nil && credit.ArtistName != "" {
			s.ArtistMbidMappings = append(s.ArtistMbidMappings, ArtistMapping{Name: credit.ArtistName, MBID: id.String()})
		}
	}
	if id, err := uuid.Parse(info.ReleaseGroupMBID); err == nil {
		s.ReleaseGroupMBID = id.String()
	}
	if info.Duration != 0 {
		s.DurationMS = int64(info.Duration) * 1000
	}
	if info.MediaPlayer != "" {
		s.Client = info.MediaPlayer
	}
	if id, err := uuid.Parse(info.RecordingMBID); err == nil {
		s.Recording = id.String()
	}
	if id, err := uuid.Parse(meta.MBIDMapping.RecordingMBID); err == nil {
		s.MappedRecording = id.String()
	}
	if id, err := uuid.Parse(info.ReleaseMBID); err == nil {
		s.ReleaseMBID = id.String()
	}
	if id, err := uuid.Parse(meta.MBIDMapping.ReleaseMBID); err == nil {
		s.MappedReleaseMBID = id.String()
	}
	return s
}

// PlayedSeconds truncates a measurement to whole seconds and preserves an explicit zero.
func PlayedSeconds(seconds float64) *int64 {
	if !(seconds >= 0 && seconds <= math.MaxInt32) {
		return nil
	}
	value := int64(seconds)
	return &value
}

// validMBIDs retains typed identifiers while dropping arbitrary strings.
func validMBIDs(values []string) []string {
	var ids []string
	for _, value := range values {
		if id, err := uuid.Parse(value); err == nil {
			ids = append(ids, id.String())
		}
	}
	return ids
}

// listenBrainzPosition accepts integer positions sent either as JSON strings or numbers.
func listenBrainzPosition(raw []byte) int {
	value, err := strconv.Atoi(strings.Trim(string(raw), `"`))
	if err != nil || value < 1 {
		return 0
	}
	return value
}
