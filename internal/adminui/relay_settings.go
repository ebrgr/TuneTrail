package adminui

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/adminweb"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/i18n"
	"github.com/ebrgr/TuneTrail/internal/relay"
	"github.com/google/uuid"
)

type relayPanelStore interface {
	RelayPolicyProfiles(context.Context, uuid.UUID, int32) ([]relay.ProfileView, error)
	RelayGrant(context.Context, uuid.UUID, int32) (relay.Grant, error)
	ConfirmRelayGrant(context.Context, uuid.UUID, int32, string) error
	RemoveRelayExceptionAsRoot(context.Context, uuid.UUID, int32, relay.Exception) error
	ResumeRelay(context.Context, uuid.UUID, int32) error
}

// relaySettingsAction handles native forms after the shared CSRF and submission checks.
func (h *Handler) relaySettingsAction(w http.ResponseWriter, r *http.Request, identity admin.Identity, f *adminweb.PreparedForm) {
	if err := admin.AuthorizeAction(identity.User, admin.ManageServer, nil); err != nil {
		h.fail(w, err)
		return
	}
	store, ok := h.store.(relayPanelStore)
	if !ok {
		h.fail(w, db.ErrNotFound)
		return
	}
	if !fields(f, "profile", "operation", "fence", "old_host", "old_cidr", "old_origin") {
		h.fail(w, db.ErrInvalidInput)
		return
	}
	profile, err := parseID(f.SingleValue("profile"))
	if err != nil {
		h.fail(w, db.ErrInvalidInput)
		return
	}
	ctx := r.Context()
	old := relay.Exception{HostPort: f.SingleValue("old_host"), CIDR: f.SingleValue("old_cidr"), Origin: f.SingleValue("old_origin")}
	switch f.SingleValue("operation") {
	case "resume":
		err = store.ResumeRelay(ctx, identity.SessionID, profile)
	case "remove":
		err = store.RemoveRelayExceptionAsRoot(ctx, identity.SessionID, profile, old)
	case "preview":
		var grant relay.Grant
		grant, err = store.RelayGrant(ctx, identity.SessionID, profile)
		if err == nil && (grant.Available() || grant.Draft) {
			h.renderRelayGrant(w, r, identity, grant)
			return
		}
		if err == nil {
			err = db.ErrConflict
		}
	case "allow":
		err = store.ConfirmRelayGrant(ctx, identity.SessionID, profile, f.SingleValue("fence"))
	default:
		err = db.ErrInvalidInput
	}
	var diagnostic *relay.ValidationError
	switch {
	case errors.As(err, &diagnostic):
		relay.LogDiagnostic(ctx, profile, diagnostic.Diagnostic, time.Now())
		code := diagnostic.Diagnostic.Code
		h.putNoticeData(identity.SessionID, 0, "admin.relay.reason_"+code, relay.Remedy(code), nil)
	case errors.Is(err, db.ErrConflict):
		h.putNoticeData(identity.SessionID, 0, "admin.relay.grant_changed", "Destination or policy changed. Review the destination and confirm again.", nil)
	case errors.Is(err, db.ErrForbidden):
		h.putNoticeData(identity.SessionID, 0, "admin.relay.reason_forbidden_range", "This destination is forbidden.", nil)
	case errors.Is(err, db.ErrInvalidInput):
		h.putNoticeData(identity.SessionID, 0, "admin.relay.invalid_exception", "Check the saved relay destination.", nil)
	case err != nil:
		h.fail(w, err)
		return
	default:
		h.putNoticeData(identity.SessionID, 0, "admin.relay.saved", "Relay policy updated.", nil)
	}
	http.Redirect(w, r, "/admin/settings#relay-exceptions", http.StatusSeeOther)
}

// renderRelayGrant issues a fresh form ticket for the root-only confirmation.
func (h *Handler) renderRelayGrant(w http.ResponseWriter, r *http.Request, identity admin.Identity, grant relay.Grant) {
	csrf, err := h.csrf.SessionToken(identity)
	if err != nil {
		h.fail(w, err)
		return
	}
	ticket, err := h.newSubmission(identity.SessionID)
	if err != nil {
		h.fail(w, err)
		return
	}
	view := settingsView{Nav: h.navigation(r, identity, nil, "settings"), Locale: i18n.ViewOf(w), CSRF: csrf, Submission: ticket, RelayGrant: grant}
	tmpl, err := h.adminTemplate(view.Locale)
	var output bytes.Buffer
	if err == nil {
		err = tmpl.ExecuteTemplate(&output, "admin/relay-grant", view)
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.Copy(w, &output)
}
