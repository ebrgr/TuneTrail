package adminui

import (
	"errors"
	"fmt"
	"net/http"

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
		h.reprocessHealthReceipt(w, r, identity, profile.ID)
		return
	}
	token, err := h.csrf.SessionToken(identity)
	if err != nil {
		h.fail(w, err)
		return
	}
	health, err := h.store.IngestHealth(r.Context(), profile.ID)
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

// reprocessHealthReceipt accepts a session-bound form for the already authorized profile.
func (h *Handler) reprocessHealthReceipt(w http.ResponseWriter, r *http.Request, identity admin.Identity, profile int32) {
	form, err := h.csrf.ParseSessionForm(w, r, identity, adminweb.FormLimits{MaxBytes: 8192, TempDir: h.tempDir})
	if err != nil {
		h.fail(w, err)
		return
	}
	defer form.Close()
	receipt, err := uuid.Parse(form.SingleValue("receipt"))
	if err != nil || !fields(form, "receipt") {
		h.fail(w, db.ErrInvalidInput)
		return
	}
	if err := h.store.ReprocessIngest(r.Context(), profile, receipt.String()); err != nil {
		h.fail(w, err)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/admin/health?profile=%d#health-failures", profile), http.StatusSeeOther)
}
