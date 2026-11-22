package adminui

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/adminweb"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/i18n"
	"github.com/ebrgr/TuneTrail/internal/webassets"
)

// accountState carries the card-level feedback of the account document (ISSUE-125).
// Card is the section that owns Error/Notice; Field names the input an error refers to.
type accountState struct {
	Card, Field   string
	Error, Notice string
	Username      string
}

// Fixed, non-secret card messages; their keys live in adminMessages.
const (
	msgUsernameInvalid  = "Use 1 to 32 letters, numbers, dots, underscores or hyphens."
	msgUsernameTaken    = "This username is not available."
	msgPasswordInvalid  = "Use between 8 and 72 characters."
	msgPasswordMismatch = "The password and confirmation do not match."
)

// accountPage serves /account to any authenticated session, whatever its role.
func (h *Handler) accountPage(w http.ResponseWriter, r *http.Request) {
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
	st := accountState{Card: "identity", Notice: h.takeNotice(identity.SessionID, 0, i18n.ViewOf(w))}
	h.renderAccount(w, r, identity, http.StatusOK, st, "")
}

// renderAccount renders the full document or, when fragment names a card, only that card.
func (h *Handler) renderAccount(w http.ResponseWriter, r *http.Request, id admin.Identity, status int, st accountState, fragment string) {
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
	if st.Username == "" {
		st.Username = id.User.Username
	}
	v := view{User: id.User, CSRF: token, Submission: ticket, Section: "account", AccountPage: &st,
		Nav: h.navigation(r, id, nil, "account")}
	if fragment == "" {
		h.render(w, status, v)
		return
	}
	v.Locale = i18n.ViewOf(w)
	tmpl, err := h.adminTemplate(v.Locale)
	var out strings.Builder
	if err == nil {
		err = tmpl.ExecuteTemplate(&out, "admin/account-"+fragment, v)
	}
	if err != nil {
		textError(w, "operation failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-TuneTrail-Submission", ticket)
	w.Header().Set("X-TuneTrail-Fragment", "account-"+fragment)
	w.WriteHeader(status)
	_, _ = io.WriteString(w, out.String())
}

// accountAction applies a username or password change with the existing service
// validation; only the presentation of errors and success is card-local.
func (h *Handler) accountAction(w http.ResponseWriter, r *http.Request, identity admin.Identity, f *adminweb.PreparedForm, path string) {
	w.Header().Add("Vary", "X-TuneTrail-Partial")
	card, names := "identity", []string{"username"}
	if path == "account/password" {
		card, names = "password", []string{"password", "confirm_password"}
	}
	if !fields(f, names...) {
		h.fail(w, db.ErrInvalidInput)
		return
	}
	fragment := ""
	if adminweb.FragmentRequested(r) {
		fragment = card
	}
	var username, password *string
	st := accountState{Card: card}
	if card == "identity" {
		v := f.SingleValue("username")
		username, st.Username = &v, v
	} else {
		v := f.SingleValue("password")
		if v != f.SingleValue("confirm_password") {
			st.Error, st.Field = msgPasswordMismatch, "confirm_password"
			h.accountFail(w, r, identity, http.StatusBadRequest, st, fragment)
			return
		}
		password = &v
	}
	changed, err := admin.ChangeAccount(r.Context(), h.store, identity.User, username, password)
	if err != nil {
		switch {
		case errors.Is(err, db.ErrInvalidInput) && card == "identity":
			st.Error, st.Field = msgUsernameInvalid, "username"
		case errors.Is(err, db.ErrInvalidInput):
			st.Error, st.Field = msgPasswordInvalid, "password"
		case errors.Is(err, db.ErrConflict) && card == "identity":
			st.Error, st.Field = msgUsernameTaken, "username"
		default:
			h.fail(w, err)
			return
		}
		h.accountFail(w, r, identity, adminweb.ErrorStatus(err), st, fragment)
		return
	}
	if changed {
		// The store revoked every session: the redirect lands on the sign-in form.
		adminweb.ClearSessionCookie(w, r)
		adminweb.ClearPreauth(w, r)
		http.Redirect(w, r, webassets.AccountDestination, http.StatusSeeOther)
		return
	}
	if fragment != "" {
		st.Notice = localMessage(i18n.ViewOf(w), "Account updated.")
		h.renderAccount(w, r, identity, http.StatusOK, st, fragment)
		return
	}
	h.putNotice(identity.SessionID, 0, "Account updated.")
	http.Redirect(w, r, webassets.AccountDestination, http.StatusSeeOther)
}

// accountFail shows the error inside the affected card, keeping the typed username.
func (h *Handler) accountFail(w http.ResponseWriter, r *http.Request, id admin.Identity, status int, st accountState, fragment string) {
	st.Error = localMessage(i18n.ViewOf(w), st.Error)
	h.renderAccount(w, r, id, status, st, fragment)
}
