package adminui

import (
	"net/http"

	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/adminweb"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/ebrgr/TuneTrail/internal/version"
	"github.com/ebrgr/TuneTrail/internal/webassets"
)

// navigation resolves the own profile independently of the selected profile.
func (h *Handler) navigation(r *http.Request, identity admin.Identity, profile *models.ListeningProfile, section string) webassets.Nav {
	nav := webassets.Nav{Kind: "admin", Version: version.Current}
	if profile != nil {
		nav.Profile = &webassets.NavProfile{ID: int64(profile.ID), Name: profile.DisplayName, Slug: profile.Slug}
		if profile.AvatarImageID != nil {
			nav.Profile.Image = *profile.AvatarImageID
		}
		if profile.AvatarIcon != nil {
			nav.Profile.Icon = *profile.AvatarIcon
		}
		if profile.AvatarColor != nil {
			nav.Profile.Color = *profile.AvatarColor
		}
	}
	var own *models.ListeningProfile
	if identity.User != nil {
		profiles, err := h.store.ListListeningProfiles(r.Context())
		if err == nil {
			own = webassets.OwnProfile(identity.User, profiles)
		}
		// A failure to load optional navigation does not replace the page response.
	}
	nav.NavAccess = webassets.AccessNavigation(identity.User, own, profile, section)
	// Offer a return only for allowlisted read URLs. The confirmation checks actor access.
	nav.SignOutFor(adminweb.SafeReturn(r.URL.RequestURI()))
	nav.PeriodLinks = nav.DefaultPeriods()
	return nav
}
