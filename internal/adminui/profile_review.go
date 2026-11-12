package adminui

import (
	"io"
	"net/http"
	"strings"

	"github.com/ebrgr/TuneTrail/internal/adminweb"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/i18n"
	"github.com/ebrgr/TuneTrail/internal/models"
)

type profileReviewView struct {
	Locale     *i18n.View
	Submission string
	Profile    *models.ListeningProfile
	Count      int64
	CSRF       string
}

// reviewProfile shows the exact profile and listen count without modifying data.
func (h *Handler) reviewProfile(w http.ResponseWriter, r *http.Request) {
	identity, err := adminweb.SessionIdentity(r, h.store)
	if err != nil {
		h.fail(w, err)
		return
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 4 || parts[0] != "admin" || parts[1] != "profile" {
		textError(w, "item not found", http.StatusNotFound)
		return
	}
	id, err := parseID(parts[2])
	if err != nil {
		h.fail(w, err)
		return
	}
	if r.URL.RawQuery != "" {
		h.fail(w, db.ErrInvalidInput)
		return
	}
	p, err := h.store.GetListeningProfileByID(r.Context(), id)
	if err != nil {
		h.fail(w, err)
		return
	}
	count, err := h.store.ProfileListenCount(r.Context(), id)
	if err != nil {
		h.fail(w, err)
		return
	}
	token, err := h.csrf.SessionToken(identity)
	if err != nil {
		h.fail(w, err)
		return
	}
	ticket, err := h.newSubmission(identity.SessionID)
	if err != nil {
		h.fail(w, err)
		return
	}
	var out strings.Builder
	view := profileReviewView{Locale: i18n.ViewOf(w), Profile: p, Count: count, CSRF: token, Submission: ticket}
	tmpl, err := h.adminTemplate(view.Locale)
	if err == nil {
		err = tmpl.ExecuteTemplate(&out, "profile-review.html", view)
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, out.String())
}
