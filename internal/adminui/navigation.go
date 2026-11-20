package adminui

import (
	"context"

	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/ebrgr/TuneTrail/internal/version"
	"github.com/ebrgr/TuneTrail/internal/webassets"
)

// LogoutForm shares the existing CSRF and logout submission contract with public pages.
func (h *Handler) LogoutForm(identity admin.Identity) (*webassets.LogoutForm, error) {
	if err := admin.AuthorizeAction(identity.User, admin.ManageAccount, nil); err != nil {
		return nil, err
	}
	csrf, err := h.csrf.SessionToken(identity)
	if err != nil {
		return nil, err
	}
	ticket, err := h.issueSubmission(identity.SessionID, true)
	if err != nil {
		return nil, err
	}
	return &webassets.LogoutForm{CSRF: csrf, Submission: ticket}, nil
}

// navigation resolves the own profile independently of the selected profile.
func (h *Handler) navigation(ctx context.Context, identity admin.Identity, profile *models.ListeningProfile, section string) webassets.Nav {
	nav := webassets.Nav{Kind: "admin", Version: version.Current}
	if profile != nil {
		nav.Profile = &webassets.NavProfile{ID: int64(profile.ID), Name: profile.DisplayName, Slug: profile.Slug}
		if profile.AvatarImageID != nil {
			nav.Profile.Image = *profile.AvatarImageID
		}
		if profile.AvatarIcon != nil {
			nav.Profile.Icon = *profile.AvatarIcon
		}
	}
	var own *models.ListeningProfile
	if identity.User != nil {
		profiles, err := h.store.ListListeningProfiles(ctx)
		if err == nil {
			own = webassets.OwnProfile(identity.User, profiles)
		}
		// A failure to load optional navigation does not replace the page response.
	}
	nav.NavAccess = webassets.AccessNavigation(identity.User, own, profile, section)
	if nav.Account != nil {
		// Ticket exhaustion leaves the other links and the existing account form available.
		nav.Logout, _ = h.LogoutForm(identity)
	}
	nav.PeriodLinks = nav.DefaultPeriods()
	return nav
}
