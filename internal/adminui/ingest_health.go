package adminui

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/adminweb"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/i18n"
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
	// Apply the Inbox filters only after authorizing the selected profile.
	days := 1
	switch r.URL.Query().Get("days") {
	case "", "1":
	case "7":
		days = 7
	case "30":
		days = 30
	default:
		h.fail(w, db.ErrInvalidInput)
		return
	}
	key := r.URL.Query().Get("key")
	var selected int64
	if key != "" {
		selected, err = strconv.ParseInt(key, 10, 64)
		if err != nil || selected < -1 || strconv.FormatInt(selected, 10) != key {
			h.fail(w, db.ErrInvalidInput)
			return
		}
	}
	now := h.now()
	groups, err := h.store.IngestInbox(r.Context(), profile.ID, nil, now.Add(-time.Duration(days)*24*time.Hour), now)
	if err != nil {
		h.fail(w, err)
		return
	}
	var inbox, keys []db.IngestInboxGroup
	var metadataIssues int64
	for _, group := range groups {
		if len(keys) == 0 || keys[len(keys)-1].Key != group.Key {
			keys = append(keys, group)
		}
		if key == "" || selected == group.Key {
			inbox = append(inbox, group)
			// Match the Inbox window and key filter using the existing group totals.
			metadataIssues += group.Missing + group.Incomplete
		}
	}
	// A receipt link from another section opens its individual confirmation.
	discardTarget := r.URL.Query().Get("discard_receipt")
	discardPage := 1
	if raw := r.URL.Query().Get("discard_page"); raw != "" && discardTarget == "" {
		discardPage, err = strconv.Atoi(raw)
		if err != nil {
			h.fail(w, db.ErrInvalidInput)
			return
		}
	}
	discard, err := h.store.IngestDiscardPreview(r.Context(), profile.ID, discardTarget, discardPage)
	if err != nil {
		h.failHealthDiscard(w, r, identity, err)
		return
	}
	if admin.AuthorizeAction(identity.User, admin.ManageAccounts, nil) != nil {
		profiles = nil
	}
	h.render(w, http.StatusOK, view{
		Nav: h.navigation(r, identity, profile, "health"), Health: &health,
		Profile: profile, Profiles: profiles, User: identity.User, CSRF: token,
		Discard: discard, DiscardTarget: discardTarget,
		Inbox: inbox, InboxKeys: keys, InboxDays: days, InboxKey: key, InboxMetadataIssues: metadataIssues,
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
	case "/admin/health/discard":
		if !fields(form, "receipt", "digest", "confirm") || form.SingleValue("confirm") != "1" {
			h.fail(w, db.ErrInvalidInput)
			return
		}
		anchor = "health-discard"
		// The store revalidates the confirmed versions against this authorized profile.
		receipt := form.SingleValue("receipt")
		err = h.store.DiscardIngestPending(r.Context(), profile, identity.User.ID, receipt, form.SingleValue("digest"))
		if receipt == "" && errors.Is(err, db.ErrConflict) {
			const message = "The eligible receipts changed. Completed batches stay discarded. Reload the page to review the remaining receipts."
			textError(w, adminFormat(i18n.ViewOf(w), "admin.health.discard_conflict", message, nil), http.StatusConflict)
			return
		}
		if err != nil {
			h.failHealthDiscard(w, r, identity, err)
			return
		}
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
		err = h.store.ReprocessIngest(r.Context(), profile, identity.User.ID, receipt.String())
	case "/admin/health/reprocess-all":
		requested, parseErr := parseID(form.SingleValue("profile"))
		if parseErr != nil || !fields(form, "profile") {
			h.fail(w, db.ErrInvalidInput)
			return
		}
		anchor = "health-failures"
		// A form rendered for another profile must not reprocess this queue.
		if requested == profile {
			err = h.store.ReprocessAllIngest(r.Context(), profile, identity.User.ID)
		}
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

// failHealthDiscard gives linked receipts the same guidance as the confirmation.
func (h *Handler) failHealthDiscard(w http.ResponseWriter, r *http.Request, identity admin.Identity, err error) {
	if errors.Is(err, db.ErrIngestDiscardListen) {
		const message = "Use listen deletion for receipts linked to a listen."
		textError(w, adminFormat(i18n.ViewOf(w), "admin.health.discard_linked", message, nil), http.StatusConflict)
		return
	}
	h.failForSession(w, r, identity, err)
}
