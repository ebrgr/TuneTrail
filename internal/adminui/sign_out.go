package adminui

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/adminweb"
	"github.com/ebrgr/TuneTrail/internal/webassets"
)

// signOutPage admits any active session and issues a ticket only for the confirmation document.
func (h *Handler) signOutPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	if strings.Contains(r.Header.Get("Sec-Purpose"), "prefetch") || r.Header.Get("Purpose") == "prefetch" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	identity, err := adminweb.SessionIdentity(r, h.store)
	if errors.Is(err, admin.ErrUnauthorized) {
		h.renderLogin(w, r, http.StatusUnauthorized, "Sign in to continue.")
		return
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	if r.Method == http.MethodHead {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		return
	}
	csrf, err := h.csrf.SessionToken(identity)
	if err != nil {
		h.fail(w, err)
		return
	}
	ticket, err := h.issueSubmission(identity.SessionID, true)
	if err != nil {
		h.fail(w, err)
		return
	}
	next := webassets.AccountDestination
	if admin.AuthorizeAction(identity.User, admin.ManageAccount, nil) != nil {
		next = "/"
	}
	if candidate := adminweb.SafeReturn(r.URL.Query().Get("return")); candidate != "" {
		if dest, err := url.Parse(candidate); err == nil && h.canReturnToRead(r, identity.User, dest) {
			next = candidate
		}
	}
	h.render(w, http.StatusOK, view{
		Nav:     h.navigation(r.Context(), identity, nil, "sign-out"),
		SignOut: true, CSRF: csrf, Submission: ticket, Next: next,
	})
}
