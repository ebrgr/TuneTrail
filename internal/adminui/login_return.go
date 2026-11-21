package adminui

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/adminweb"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/ebrgr/TuneTrail/internal/webassets"
	"github.com/ebrgr/TuneTrail/internal/webquery"
)

// loginDestination chooses an authorized read before falling back to the role's home.
func (h *Handler) loginDestination(r *http.Request, actor *models.User, requested string) string {
	if next := adminweb.SafeReturn(requested); next != "" {
		u, err := url.Parse(next)
		if err == nil && h.canReturnToRead(r, actor, u) {
			return next
		}
	}
	if admin.AuthorizeAction(actor, admin.ManageServer, nil) == nil {
		return "/admin"
	}
	return webassets.AccountDestination
}

// canReturnToRead checks the read route, its query and the canonical profile before redirection.
func (h *Handler) canReturnToRead(r *http.Request, actor *models.User, u *url.URL) bool {
	if u.Path == "/" {
		return u.RawQuery == ""
	}
	parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	read := r.Clone(r.Context())
	read.Method, read.URL = http.MethodGet, u
	switch u.Path {
	case "/admin":
		return h.authorizePage(read, admin.Identity{User: actor}) == nil && h.returnProfileExists(read)
	case "/admin/health":
		// Recheck the selected profile against the new session before redirecting.
		profiles, err := h.store.ListListeningProfiles(read.Context())
		if err != nil {
			return false
		}
		_, err = h.healthProfile(read, actor, profiles)
		return err == nil
	case "/admin/settings":
		return admin.AuthorizeAction(actor, admin.ManageServer, nil) == nil
	}
	if len(parts) > 1 && parts[1] == "manage" {
		return h.canReturnToMediaTask(read, actor, parts)
	}
	profile, err := h.store.GetListeningProfile(r.Context(), parts[0])
	if err != nil || admin.AuthorizeAction(actor, admin.ReadListening, profile) != nil {
		return false
	}
	read = read.WithContext(db.WithProfile(r.Context(), profile.ID))
	section := strings.Join(parts[1:], "/")
	if section == "rewind" {
		_, err := webquery.Rewind(read, time.Now())
		return err == nil
	}
	if len(parts) == 3 {
		for key, values := range u.Query() {
			if key != "tz" && !(key == "period" && values[0] == "all_time") {
				return false
			}
		}
		n, err := strconv.ParseInt(parts[2], 10, 32)
		if err != nil {
			return false
		}
		ctx := read.Context()
		if admin.AuthorizeAction(actor, admin.ManageCatalog, nil) == nil {
			ctx = db.WithoutProfile(ctx)
		}
		return h.returnMediaExists(ctx, parts[1], int32(n))
	}
	opts, _, err := webquery.HTML(read, section == "")
	return err == nil && webquery.RankingFiltersAllowed(section, opts)
}

// returnProfileExists validates the optional legacy profile selection before redirecting.
func (h *Handler) returnProfileExists(r *http.Request) bool {
	raw := r.URL.Query().Get("profile")
	if raw == "" {
		return true
	}
	id, err := parseID(raw)
	if err != nil {
		return false
	}
	_, err = h.store.GetListeningProfileByID(r.Context(), id)
	return err == nil
}

// canReturnToMediaTask preserves authorized legacy GET task documents after login.
func (h *Handler) canReturnToMediaTask(r *http.Request, actor *models.User, parts []string) bool {
	if !IsPage(r.URL.Path) {
		return false
	}
	profile, err := h.store.GetListeningProfile(r.Context(), parts[0])
	if err != nil {
		return false
	}
	if admin.AuthorizeAction(actor, mediaCapability(r, parts), profile) != nil {
		return false
	}
	id, err := parseID(parts[3])
	if err != nil {
		return false
	}
	return h.returnMediaExists(db.WithoutProfile(r.Context()), parts[2], id)
}

// returnMediaExists checks a resolved ID in the caller's authorized read scope.
func (h *Handler) returnMediaExists(ctx context.Context, kind string, id int32) bool {
	var err error
	switch kind {
	case "artist":
		_, err = h.store.GetArtist(ctx, db.GetArtistOpts{ID: id})
	case "album":
		_, err = h.store.GetAlbum(ctx, db.GetAlbumOpts{ID: id})
	case "track":
		_, err = h.store.GetTrack(ctx, db.GetTrackOpts{ID: id})
	default:
		return false
	}
	return err == nil
}
