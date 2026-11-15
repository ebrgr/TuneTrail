package adminui

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/ebrgr/TuneTrail/internal/adminweb"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/models"
)

// mediaDestination retains personal public reads and global administrative access to survivors.
func (h *Handler) mediaDestination(ctx context.Context, p *models.ListeningProfile, kind string, id int32) string {
	ctx = db.WithProfile(ctx, p.ID)
	var err error
	switch kind {
	case "artist":
		_, err = h.store.GetArtist(ctx, db.GetArtistOpts{ID: id})
	case "album":
		_, err = h.store.GetAlbum(ctx, db.GetAlbumOpts{ID: id})
	case "track":
		_, err = h.store.GetTrack(ctx, db.GetTrackOpts{ID: id})
	}
	prefix := "/" + url.PathEscape(p.Slug)
	if err == nil {
		return prefix + fmt.Sprintf("/%s/%d", kind, id)
	}
	// After a committed mutation, a failed personal read falls back to the protected item index.
	return prefix + fmt.Sprintf("/manage/%s/%d", kind, id)
}

// historyMediaDestination avoids a dead public page after removing its last personal listen.
func (h *Handler) historyMediaDestination(ctx context.Context, p *models.ListeningProfile, target string) string {
	u, err := url.Parse(target)
	if err != nil {
		return target
	}
	parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	if len(parts) != 3 || parts[0] != p.Slug || adminweb.SafeHistoryReturn(target, p.Slug) == "" {
		return target
	}
	itemID, err := parseID(parts[2])
	if err != nil {
		return target
	}
	destination := h.mediaDestination(ctx, p, parts[1], itemID)
	if destination == u.Path {
		return target
	}
	// Public history filters do not apply to the protected index; retain the display zone.
	q := url.Values{}
	if zone := u.Query().Get("tz"); zone != "" {
		q.Set("tz", zone)
	}
	u.Path, u.RawQuery = destination, q.Encode()
	return u.String()
}
