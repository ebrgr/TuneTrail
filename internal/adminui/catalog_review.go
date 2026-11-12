package adminui

import (
	"errors"
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

type catalogReviewView struct {
	Locale                        *i18n.View
	Profile                       *models.ListeningProfile
	RouteID                       int32
	Kind, CSRF, Token, ReturnPath string
	Plan                          db.CatalogPlan
}

// reviewCatalog only reads data and issues a short-lived confirmation bound to the session and route.
func (h *Handler) reviewCatalog(w http.ResponseWriter, r *http.Request, identity admin.Identity, p *models.ListeningProfile, kind string, id int32) {
	q, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		h.fail(w, db.ErrInvalidInput)
		return
	}
	for k, v := range q {
		if len(v) != 1 || (k != "action" && k != "to" && k != "reverse" && k != "replace_image") {
			h.fail(w, db.ErrInvalidInput)
			return
		}
	}
	request := db.CatalogPlan{Action: q.Get("action"), Kind: kind, From: id}
	if request.Action == "merge" {
		request.To, err = parseID(q.Get("to"))
		if err != nil {
			h.fail(w, err)
			return
		}
		reverse := q.Get("reverse")
		if reverse != "" && reverse != "1" {
			h.fail(w, db.ErrInvalidInput)
			return
		}
		if reverse == "1" {
			request.From, request.To = request.To, request.From
		}
		replace := q.Get("replace_image")
		if replace != "" && replace != "1" {
			h.fail(w, db.ErrInvalidInput)
			return
		}
		request.ReplaceImage = replace == "1"
	} else if request.Action != "delete" || len(q) != 1 {
		h.fail(w, db.ErrInvalidInput)
		return
	}
	store, ok := h.store.(admin.CatalogMutationStore)
	if !ok {
		h.fail(w, fmt.Errorf("catalog operations unavailable"))
		return
	}
	plan, err := admin.PreviewCatalogMutation(r.Context(), store, identity.User, request)
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
	h.mu.Lock()
	now := time.Now()
	for k, v := range h.catalogPlans {
		if now.After(v.Expires) {
			delete(h.catalogPlans, k)
		}
	}
	if len(h.catalogPlans) >= 256 {
		h.mu.Unlock()
		h.fail(w, fmt.Errorf("too many pending reviews"))
		return
	}
	h.catalogPlans[token] = pendingCatalogPlan{Session: identity.SessionID, ProfileID: p.ID, ProfileSlug: p.Slug, RouteID: id, Kind: kind, Plan: plan, Expires: now.Add(10 * time.Minute)}
	h.mu.Unlock()
	view := catalogReviewView{Locale: i18n.ViewOf(w), Profile: p, RouteID: id, Kind: kind, CSRF: csrf, Token: token.String(), Plan: plan, ReturnPath: fmt.Sprintf("/%s/manage/%s/%d", url.PathEscape(p.Slug), kind, id)}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var out strings.Builder
	tmpl, err := h.adminTemplate(view.Locale)
	if err == nil {
		err = tmpl.ExecuteTemplate(&out, "catalog-review.html", view)
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	_, _ = io.WriteString(w, out.String())
}

// confirmCatalog consumes a reviewed plan and revalidates it in the same database transaction as the mutation.
func (h *Handler) confirmCatalog(w http.ResponseWriter, r *http.Request, identity admin.Identity, p *models.ListeningProfile, kind string, id int32) {
	form, err := h.csrf.ParseSessionForm(w, r, identity, adminweb.FormLimits{MaxBytes: 8192, TempDir: h.tempDir})
	if err != nil {
		h.fail(w, err)
		return
	}
	defer form.Close()
	if !fields(form, "profile_id", "profile_slug", "plan", "confirm") || form.SingleValue("confirm") != "1" || form.SingleValue("profile_id") != strconv.FormatInt(int64(p.ID), 10) || form.SingleValue("profile_slug") != p.Slug {
		h.fail(w, db.ErrInvalidInput)
		return
	}
	token, err := uuid.Parse(form.SingleValue("plan"))
	if err != nil {
		h.fail(w, db.ErrInvalidInput)
		return
	}
	h.mu.Lock()
	review, ok := h.catalogPlans[token]
	if ok && review.Session == identity.SessionID && review.ProfileID == p.ID && review.ProfileSlug == p.Slug && review.RouteID == id && review.Kind == kind {
		delete(h.catalogPlans, token)
	} else {
		ok = false
	}
	h.mu.Unlock()
	if !ok || time.Now().After(review.Expires) {
		h.fail(w, db.ErrConflict)
		return
	}
	store, ok := h.store.(admin.CatalogMutationStore)
	if !ok {
		h.fail(w, fmt.Errorf("catalog operations unavailable"))
		return
	}
	if err = admin.ApplyCatalogMutation(r.Context(), store, identity.User, review.Plan); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			err = db.ErrConflict
		}
		h.fail(w, err)
		return
	}
	// The profile home is always a valid personal destination, even for an unplayed survivor.
	http.Redirect(w, r, "/"+url.PathEscape(p.Slug), http.StatusSeeOther)
}
