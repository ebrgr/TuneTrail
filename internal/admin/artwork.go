package admin

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/ebrgr/TuneTrail/imagecache"
	"github.com/ebrgr/TuneTrail/internal/catalog"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/imagepipeline"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/google/uuid"
)

const MaxArtworkBytes = 10 << 20

// ArtworkStore permits publication and reference checks against the shared catalog.
type ArtworkStore interface {
	MediaStore
	db.ImageStore
}

// ArtworkClient has one bounded policy for image URLs. Self-hosted HTTP(S) is supported.
type ArtworkClient struct {
	Transport http.RoundTripper
	Timeout   time.Duration
}

// validArtworkURL keeps the same HTTP(S), no-userinfo policy for every hop.
func validArtworkURL(u *url.URL) bool {
	return u != nil && u.Host != "" && u.User == nil && u.Fragment == "" && !u.ForceQuery && (u.Scheme == "http" || u.Scheme == "https") && len(u.String()) <= 2048
}
func (c ArtworkClient) download(ctx context.Context, raw string) ([]byte, error) {
	u, err := url.Parse(raw)
	if err != nil || !validArtworkURL(u) {
		return nil, db.ErrInvalidInput
	}
	req, err := http.NewRequestWithContext(ctx, "GET", raw, nil)
	if err != nil {
		return nil, db.ErrInvalidInput
	}
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	client := http.Client{Transport: c.Transport, Timeout: timeout, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 4 || !validArtworkURL(req.URL) {
			return fmt.Errorf("invalid artwork redirect")
		}
		return nil
	}}
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("artwork download failed")
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("artwork download failed")
	}
	data, err := io.ReadAll(io.LimitReader(res.Body, MaxArtworkBytes+1))
	if err != nil {
		return nil, fmt.Errorf("artwork download failed")
	}
	if len(data) > MaxArtworkBytes {
		return nil, imagepipeline.ErrTooLarge
	}
	return data, nil
}

// ReplaceArtwork publishes a new image and associates it atomically with collection.
// A track explicitly targets its current album; previous files remain for orphan collection.
func ReplaceArtwork(ctx context.Context, s ArtworkStore, user *models.User, kind string, id int32, rawURL string, upload io.Reader, c ArtworkClient) (uuid.UUID, error) {
	// Authorize before occupying a limited administrative image slot.
	if err := Authorize(user); err != nil {
		return uuid.Nil, err
	}
	ctx, release, err := AcquireImageAdmission(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer release()
	if err := MediaExists(ctx, s, user, kind, id); err != nil {
		return uuid.Nil, err
	}
	if (rawURL == "") == (upload == nil) {
		return uuid.Nil, db.ErrInvalidInput
	}
	target := id
	trackID := int32(0)
	ctx = db.WithoutProfile(ctx)
	if kind == "track" {
		trackID = id
		track, err := s.GetTrack(ctx, db.GetTrackOpts{ID: id})
		if err != nil {
			return uuid.Nil, err
		}
		target = track.AlbumID
		kind = "album"
	}
	var data []byte
	source := catalog.ImageSourceUserUpload
	if rawURL != "" {
		data, err = c.download(ctx, rawURL)
		source = rawURL
	} else {
		data, err = io.ReadAll(io.LimitReader(upload, MaxArtworkBytes+1))
		if err == nil && len(data) > MaxArtworkBytes {
			err = imagepipeline.ErrTooLarge
		}
	}
	if err != nil {
		return uuid.Nil, err
	}
	unlock, err := imagecache.LockPublicationContext(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer unlock()
	if err = ctx.Err(); err != nil {
		return uuid.Nil, err
	}
	uid := uuid.New()
	if err = imagecache.SaveImageContext(ctx, uid, bytes.NewReader(data)); err != nil {
		return uuid.Nil, fmt.Errorf("save artwork: %w", err)
	}
	switch kind {
	case "artist":
		err = s.UpdateArtist(ctx, db.UpdateArtistOpts{ID: target, Image: uid, ImageSrc: source})
	case "album":
		if trackID != 0 {
			if updater, ok := s.(interface {
				UpdateTrackAlbumArtwork(context.Context, int32, int32, uuid.UUID, string) error
			}); ok {
				err = updater.UpdateTrackAlbumArtwork(ctx, trackID, target, uid, source)
			} else {
				err = fmt.Errorf("track artwork update unavailable")
			}
		} else {
			err = s.UpdateAlbum(ctx, db.UpdateAlbumOpts{ID: target, Image: uid, ImageSrc: source})
		}
	default:
		err = db.ErrInvalidInput
	}
	if err != nil {
		used, e := s.ImageHasAssociation(context.WithoutCancel(ctx), uid)
		if e == nil && !used {
			_ = imagecache.DeleteImage(uid)
		}
		return uuid.Nil, err
	}
	return uid, nil
}
