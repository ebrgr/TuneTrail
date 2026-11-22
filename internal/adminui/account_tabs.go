package adminui

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/adminweb"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/i18n"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/ebrgr/TuneTrail/internal/webassets"
)

// accountRow is one entry of the root accounts list (ISSUE-126).
type accountRow struct {
	Profile models.ListeningProfile
	Account db.AccountDetails
	Avatar  webassets.NavProfile
	Own     bool
}

// cardTab names the tab whose data a swappable profile card needs.
var cardTab = map[string]string{
	"profile-identity": webassets.TabProfile,
	"profile-avatar":   webassets.TabProfile,
	"profile-keys":     webassets.TabKeys,
	"profile-relay":    webassets.TabRelay,
	"account-role":     webassets.TabAccount,
}

// operationTab maps a profile POST operation to the tab it belongs to, so that the redirect
// after a successful action lands where the user was working.
func operationTab(operation string) string {
	switch {
	case strings.HasPrefix(operation, "keys/"):
		return webassets.TabKeys
	case strings.HasPrefix(operation, "relay/"):
		return webassets.TabRelay
	case operation == "import":
		return webassets.TabHistory
	case operation == "account-role" || operation == "account-token":
		return webassets.TabAccount
	}
	return webassets.TabProfile
}

// operationCard maps an operation to the card that a script may swap; "" means a full page.
func operationCard(operation string) string {
	switch {
	case operation == "save":
		return "profile-identity"
	case operation == "avatar" || operation == "avatar-clear":
		return "profile-avatar"
	case strings.HasPrefix(operation, "keys/"):
		return "profile-keys"
	case strings.HasPrefix(operation, "relay/"):
		return "profile-relay"
	case operation == "account-role":
		return "account-role"
	}
	return ""
}

// ownProfile resolves the profile of the signed-in user from the session, never from the request.
func (h *Handler) ownProfile(ctx context.Context, user *models.User) (*models.ListeningProfile, error) {
	profiles, err := h.store.ListListeningProfiles(ctx)
	if err != nil {
		return nil, err
	}
	own := webassets.OwnProfile(user, profiles)
	if own == nil {
		return nil, db.ErrNotFound
	}
	return own, nil
}

// accountTab serves /account/<tab> for the session's own profile (ISSUE-126). The target is
// derived from the session, so these documents never read an ID or slug from the request.
func (h *Handler) accountTab(w http.ResponseWriter, r *http.Request) {
	identity, err := adminweb.SessionIdentity(r, h.store)
	if errors.Is(err, admin.ErrUnauthorized) {
		h.loginPage(w, r, "")
		return
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	if err := admin.AuthorizeAction(identity.User, admin.ManageAccount, nil); err != nil {
		h.failForSession(w, r, identity, err)
		return
	}
	if len(r.URL.Query()) != 0 {
		h.fail(w, db.ErrInvalidInput)
		return
	}
	own, err := h.ownProfile(r.Context(), identity.User)
	if err == nil {
		// The same policy as every profile action decides before anything is loaded.
		_, err = admin.AuthorizeProfile(r.Context(), h.store, identity.User, own.ID)
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	h.renderProfileTab(w, r, identity, own, strings.TrimPrefix(r.URL.Path, "/account/"), true, accountState{}, http.StatusOK, "")
}

// renderProfileTab renders one tab of the account shell for the target profile, or only the
// card named by fragment. self is true for the session's own profile and false for a root
// operating another account; callers derive it from the session, never from the request.
func (h *Handler) renderProfileTab(w http.ResponseWriter, r *http.Request, id admin.Identity, profile *models.ListeningProfile,
	tab string, self bool, st accountState, status int, fragment string) {
	if !self && admin.AuthorizeAction(id.User, admin.ManageAccounts, nil) != nil {
		h.fail(w, admin.ErrForbidden)
		return
	}
	token, err := h.csrf.SessionToken(id)
	if err != nil {
		h.fail(w, err)
		return
	}
	ticket, err := h.newSubmission(id.SessionID)
	if err != nil {
		h.fail(w, err)
		return
	}
	locale := i18n.ViewOf(w)
	if fragment == "" && st.Notice == "" && st.Error == "" {
		st.Notice = h.takeNotice(id.SessionID, profile.ID, locale)
	}
	v := view{User: id.User, CSRF: token, Submission: ticket, Profile: profile, Tab: tab, Section: "account",
		Tabs: webassets.AccountTabs(self, profile.ID, tab), Context: !self, AccountPage: &st}
	switch tab {
	case webassets.TabKeys:
		v.Keys, err = admin.Keys(r.Context(), h.store, id.User, profile.ID, profile.Slug)
	case webassets.TabRelay:
		v.RelayURL, _, err = admin.Relay(r.Context(), h.store, id.User, profile.ID, profile.Slug)
		if store, ok := h.store.(relayPanelStore); err == nil && ok && admin.AuthorizeAction(id.User, admin.ManageServer, nil) == nil {
			if states, _ := store.RelayPolicyProfiles(r.Context(), id.SessionID, profile.ID); len(states) != 0 && states[0].Configured {
				v.RelayState = &states[0]
			}
		}
	case webassets.TabAccount:
		if !self {
			// Account details and root tools exist only in the administration context.
			v.Account, err = h.store.ProfileAccount(r.Context(), profile.ID)
			v.CanIssueToken = err == nil && admin.AuthorizeAction(id.User, admin.IssueAccountToken, profile) == nil
		}
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	if self {
		v.Nav = h.navigation(r, id, nil, "account")
	} else {
		v.Nav = h.navigation(r, id, profile, "profiles")
	}
	if fragment == "" {
		h.render(w, status, v)
		return
	}
	v.Locale = locale
	tmpl, err := h.adminTemplate(locale)
	var out strings.Builder
	if err == nil {
		err = tmpl.ExecuteTemplate(&out, "admin/card-"+fragment, v)
	}
	if err != nil {
		textError(w, "operation failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-TuneTrail-Submission", ticket)
	w.Header().Set("X-TuneTrail-Fragment", fragment)
	w.WriteHeader(status)
	_, _ = io.WriteString(w, out.String())
}

// accountRows lists every account and its profile for the root accounts list.
func (h *Handler) accountRows(ctx context.Context, actor *models.User, profiles []models.ListeningProfile) []accountRow {
	rows := make([]accountRow, 0, len(profiles))
	for _, p := range profiles {
		row := accountRow{Profile: p, Own: p.UserID == actor.ID, Avatar: webassets.NavProfile{ID: int64(p.ID), Name: p.DisplayName, Slug: p.Slug}}
		if p.AvatarImageID != nil {
			row.Avatar.Image = *p.AvatarImageID
		}
		if p.AvatarIcon != nil {
			row.Avatar.Icon = *p.AvatarIcon
		}
		// A missing detail only hides the role and state of that row.
		if details, err := h.store.ProfileAccount(ctx, p.ID); err == nil {
			row.Account = details
		}
		rows = append(rows, row)
	}
	return rows
}
