package ingest

import (
	"github.com/ebrgr/TuneTrail/internal/listenbrainz"
	"github.com/google/uuid"
)

// ListenBrainzSnapshot excludes arbitrary extensions, raw headers and URLs.
func ListenBrainzSnapshot(meta listenbrainz.LbzTrackMeta) Snapshot {
	info := meta.AdditionalInfo
	s := Snapshot{Artist: meta.ArtistName, Title: meta.TrackName, Release: meta.ReleaseName,
		ArtistNames: info.ArtistNames, DurationMS: int64(info.DurationMs), Client: info.SubmissionClient,
		ClientVersion: info.SubmissionClientVersion}
	s.ArtistMBIDs = validMBIDs(info.ArtistMBIDs)
	s.MappedArtistMBIDs = validMBIDs(meta.MBIDMapping.ArtistMBIDs)
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
