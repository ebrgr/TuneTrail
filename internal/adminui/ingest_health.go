package adminui

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/adminweb"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/ebrgr/TuneTrail/internal/webassets"
	"github.com/google/uuid"
)

// healthProfile resolves the explicit profile or defaults to the session owner's profile.
func (h *Handler) healthProfile(r *http.Request, actor *models.User, profiles []models.ListeningProfile) (*models.ListeningProfile, error) {
	if raw := r.URL.Query().Get("profile"); raw != "" {
		id, err := parseID(raw)
		if err != nil {
			return nil, err
		}
		return admin.AuthorizeProfile(r.Context(), h.store, actor, id)
	}
	profile := webassets.OwnProfile(actor, profiles)
	if profile == nil && len(profiles) > 0 && admin.AuthorizeAction(actor, admin.ManageAccounts, nil) == nil {
		profile = &profiles[0]
	}
	if profile == nil {
		return nil, db.ErrNotFound
	}
	return admin.AuthorizeProfile(r.Context(), h.store, actor, profile.ID)
}

// healthPage authenticates and authorizes the profile before reading any diagnostics.
func (h *Handler) healthPage(w http.ResponseWriter, r *http.Request) {
	identity, err := adminweb.SessionIdentity(r, h.store)
	if r.Method == http.MethodGet && errors.Is(err, admin.ErrUnauthorized) && r.Header.Get("Authorization") == "" {
		h.loginPage(w, r, "")
		return
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	profiles, err := h.store.ListListeningProfiles(r.Context())
	if err != nil {
		h.fail(w, err)
		return
	}
	profile, err := h.healthProfile(r, identity.User, profiles)
	if err != nil {
		h.failForSession(w, r, identity, err)
		return
	}
	if r.Method == http.MethodPost {
		h.mutateHealth(w, r, identity, profile.ID)
		return
	}
	token, err := h.csrf.SessionToken(identity)
	if err != nil {
		h.fail(w, err)
		return
	}
	// The store clamps the page to the current queue after concurrent changes.
	page := 1
	if raw := r.URL.Query().Get("review_page"); raw != "" {
		page, err = strconv.Atoi(raw)
		if err != nil || page < 1 {
			h.fail(w, db.ErrInvalidInput)
			return
		}
	}
	sourcePage := 1
	if raw := r.URL.Query().Get("source_page"); raw != "" {
		// Parse into 32 bits so converting a page to an offset cannot overflow.
		requested, parseErr := strconv.ParseInt(raw, 10, 32)
		if parseErr != nil || requested < 1 {
			h.fail(w, db.ErrInvalidInput)
			return
		}
		sourcePage = int(requested)
	}
	health, err := h.store.IngestHealth(r.Context(), profile.ID, page, sourcePage)
	if err != nil {
		h.fail(w, err)
		return
	}
	if admin.AuthorizeAction(identity.User, admin.ManageAccounts, nil) != nil {
		profiles = nil
	}
	h.render(w, http.StatusOK, view{
		Nav: h.navigation(r, identity, profile, "health"), Health: &health,
		Profile: profile, Profiles: profiles, User: identity.User, CSRF: token,
	})
}

// mutateHealth accepts session-bound forms for the already authorized profile.
func (h *Handler) mutateHealth(w http.ResponseWriter, r *http.Request, identity admin.Identity, profile int32) {
	form, err := h.csrf.ParseSessionForm(w, r, identity, adminweb.FormLimits{MaxBytes: 8192, TempDir: h.tempDir})
	if err != nil {
		h.fail(w, err)
		return
	}
	defer form.Close()
	anchor := "health-review"
	switch r.URL.Path {
	case "/admin/health/review-setting":
		enabled := form.SingleValue("enabled")
		if !fields(form, "enabled") || (enabled != "1" && enabled != "0") {
			h.fail(w, db.ErrInvalidInput)
			return
		}
		err = h.store.SetIngestReview(r.Context(), profile, enabled == "1")
	case "/admin/health/review":
		receipt, receiptErr := uuid.Parse(form.SingleValue("receipt"))
		version, versionErr := strconv.ParseInt(form.SingleValue("version"), 10, 64)
		if receiptErr != nil || versionErr != nil || !fields(form, "receipt", "version", "decision", "candidate") {
			h.fail(w, db.ErrInvalidInput)
			return
		}
		err = h.store.ResolveIngestReview(r.Context(), profile, identity.User.ID, db.IngestReviewDecision{
			Receipt: receipt.String(), Version: version, Value: form.SingleValue("decision"), Candidate: form.SingleValue("candidate"),
		})
	case "/admin/health/review-batch":
		if !fields(form, "decision", "digest") {
			h.fail(w, db.ErrInvalidInput)
			return
		}
		// The account ID always comes from the authorized session.
		err = h.store.ResolveIngestReviewBatch(r.Context(), profile, identity.User.ID, form.SingleValue("decision"), form.SingleValue("digest"))
	case "/admin/health/reprocess":
		receipt, parseErr := uuid.Parse(form.SingleValue("receipt"))
		if parseErr != nil || !fields(form, "receipt") {
			h.fail(w, db.ErrInvalidInput)
			return
		}
		anchor = "health-failures"
		err = h.store.ReprocessIngest(r.Context(), profile, receipt.String())
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if err != nil {
		h.failForSession(w, r, identity, err)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/admin/health?profile=%d#%s", profile, anchor), http.StatusSeeOther)
}
