package admin

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/models"
)

// LocalListenTime rejects nonexistent local times and requires an explicit UTC offset for ambiguous times.
func LocalListenTime(value, zone, offset string) (time.Time, error) {
	loc, err := time.LoadLocation(zone)
	if err != nil {
		return time.Time{}, db.ErrInvalidInput
	}
	wall, err := time.Parse("2006-01-02T15:04", value)
	if err != nil {
		return time.Time{}, db.ErrInvalidInput
	}
	var matches []time.Time
	// Only minute-aligned offsets are accepted; historical second offsets are rejected.
	for seconds := -24 * 3600; seconds <= 24*3600; seconds += 60 {
		candidate := wall.Add(-time.Duration(seconds) * time.Second)
		if candidate.In(loc).Format("2006-01-02T15:04") == value {
			_, o := candidate.In(loc).Zone()
			if o == seconds {
				matches = append(matches, candidate)
			}
		}
	}
	if len(matches) == 0 {
		return time.Time{}, db.ErrInvalidInput
	}
	if len(matches) > 1 && offset == "" {
		return time.Time{}, db.ErrConflict
	}
	if offset != "" {
		if len(offset) != 6 || (offset[0] != '+' && offset[0] != '-') || offset[3] != ':' || offset[1] < '0' || offset[1] > '9' || offset[2] < '0' || offset[2] > '9' || offset[4] < '0' || offset[4] > '9' || offset[5] < '0' || offset[5] > '9' {
			return time.Time{}, db.ErrInvalidInput
		}
		hh, e1 := strconv.Atoi(offset[1:3])
		mm, e2 := strconv.Atoi(offset[4:6])
		if e1 != nil || e2 != nil || hh > 23 || mm > 59 {
			return time.Time{}, db.ErrInvalidInput
		}
		wanted := (hh*60 + mm) * 60
		if strings.HasPrefix(offset, "-") {
			wanted = -wanted
		}
		for _, t := range matches {
			_, o := t.In(loc).Zone()
			if o == wanted {
				return t, nil
			}
		}
		return time.Time{}, db.ErrInvalidInput
	}
	return matches[0], nil
}

// ManualListen inserts exactly once for the selected profile and track.
func ManualListen(ctx context.Context, s interface {
	db.TrackStore
	db.ListenStore
}, user *models.User, profileID, trackID int32, at time.Time, client string) (bool, error) {
	if err := Authorize(user); err != nil {
		return false, err
	}
	if profileID <= 0 || trackID <= 0 || at.IsZero() || at.After(time.Now()) {
		return false, db.ErrInvalidInput
	}
	if _, err := s.GetTrack(db.WithoutProfile(ctx), db.GetTrackOpts{ID: trackID}); err != nil {
		return false, err
	}
	if client == "" {
		client = "Koito Web UI"
	}
	opts := db.SaveListenOpts{ProfileID: profileID, TrackID: trackID, Time: at, Client: client}
	if recorder, ok := s.(interface {
		SaveListenWithResult(context.Context, db.SaveListenOpts) (bool, error)
	}); ok {
		return recorder.SaveListenWithResult(ctx, opts)
	}
	err := s.SaveListen(ctx, opts)
	return err == nil, err
}

// RemovePersonalListen addresses a single profile, track and instant.
func RemovePersonalListen(ctx context.Context, s db.ListenStore, user *models.User, profileID, trackID int32, at time.Time) error {
	if err := Authorize(user); err != nil {
		return err
	}
	if profileID <= 0 || trackID <= 0 || at.IsZero() {
		return db.ErrInvalidInput
	}
	return s.DeleteListen(db.WithProfile(ctx, profileID), trackID, at)
}
