package adminui

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/adminweb"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/i18n"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/google/uuid"
)

// reviewProfileDelete issues a scoped, short-lived plan and a required one-use submission ticket.
func (h *Handler) reviewProfileDelete(w http.ResponseWriter, r *http.Request, identity admin.Identity, p *models.ListeningProfile, kind string, id int32) {
	// Authorize before exposing the personal deletion preview.
	if err := h.authorizePersonalPlan(r, identity, p.ID); err != nil {
		h.fail(w, err)
		return
	}
	partial := r.URL.Query().Get("partial") == "1"
	if partial && !dialogRequested(r) {
		q := r.URL.Query()
		q.Del("partial")
		http.Redirect(w, r, (&url.URL{Path: r.URL.Path, RawQuery: q.Encode()}).RequestURI(), http.StatusSeeOther)
		return
	}
	store, ok := h.store.(interface {
		db.ProfileStore
		db.ProfileDeleteStore
	})
	if !ok {
		h.fail(w, fmt.Errorf("personal deletion unavailable"))
		return
	}
	plan, err := admin.PreviewPersonalDelete(r.Context(), store, identity.User, p.ID, kind, id)
	if err != nil {
		h.fail(w, err)
		return
	}
	token, err := uuid.NewRandom()
	if err != nil {
		h.fail(w, err)
		return
	}
	csrf, err := h.csrf.SessionToken(identity)
	if err != nil {
		h.fail(w, err)
		return
	}
	submission, err := h.newSubmission(identity.SessionID)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.mu.Lock()
	now := time.Now()
	for key, value := range h.catalogPlans {
		if now.After(value.Expires) {
			delete(h.catalogPlans, key)
		}
	}
	if len(h.catalogPlans) >= 256 {
		h.mu.Unlock()
		h.fail(w, fmt.Errorf("too many pending reviews"))
		return
	}
	h.catalogPlans[token] = pendingCatalogPlan{Session: identity.SessionID, ProfileID: p.ID, ProfileSlug: p.Slug, RouteID: id, Kind: kind, Personal: &plan, Expires: now.Add(10 * time.Minute)}
	h.mu.Unlock()
	view := catalogReviewView{Locale: i18n.ViewOf(w), Profile: p, RouteID: id, Kind: kind, CSRF: csrf, Token: token.String(), Submission: submission, Personal: &plan, Dialog: partial, ReturnPath: fmt.Sprintf("/%s/%s/%d", url.PathEscape(p.Slug), kind, id)}
	tmpl, err := h.adminTemplate(view.Locale)
	var output strings.Builder
	name := "admin/catalog-review"
	if partial {
		name = "manage/profile-delete"
	}
	if err == nil {
		err = tmpl.ExecuteTemplate(&output, name, view)
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-TuneTrail-Submission", submission)
	if partial {
		w.Header().Set("X-TuneTrail-Fragment", "task")
	}
	// A disconnected reader cannot recover this rendered response; no mutation has been sent.
	_, _ = io.WriteString(w, output.String())
}

// confirmProfileDelete consumes the ticket even on failure; uncertain results are never retried.
func (h *Handler) confirmProfileDelete(w http.ResponseWriter, r *http.Request, identity admin.Identity, p *models.ListeningProfile, form *adminweb.PreparedForm, plan db.ProfileDeletePlan) {
	// Revalidate ownership when applying a previously reviewed plan.
	if plan.ProfileID != p.ID {
		h.fail(w, db.ErrConflict)
		return
	}
	if err := h.authorizePersonalPlan(r, identity, p.ID); err != nil {
		h.fail(w, err)
		return
	}
	if err := h.consumeSubmission(identity.SessionID, form.SingleValue("submission_token")); err != nil {
		h.fail(w, err)
		return
	}
	store, ok := h.store.(interface {
		db.ProfileStore
		db.ProfileDeleteStore
	})
	if !ok {
		h.fail(w, fmt.Errorf("personal deletion unavailable"))
		return
	}
	count, err := admin.ApplyPersonalDelete(r.Context(), store, identity.User, plan)
	if err != nil {
		h.fail(w, err)
		return
	}
	w.Header().Set("X-TuneTrail-Deleted-Listens", strconv.FormatInt(count, 10))
	target := "/" + url.PathEscape(p.Slug)
	if dialogRequested(r) {
		w.Header().Set("X-TuneTrail-Navigate", target)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}
