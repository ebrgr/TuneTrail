package adminui

import (
	"context"
	"errors"
	"net/http"
	"net/netip"
	"time"

	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/adminweb"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/relay"
	"github.com/google/uuid"
)

type relayPanelStore interface {
	RelayPolicyProfiles(context.Context, uuid.UUID) ([]relay.ProfileView, error)
	SetRelayPermissionAsRoot(context.Context, uuid.UUID, int32, relay.Permission) error
	EditRelayExceptionAsRoot(context.Context, uuid.UUID, int32, relay.Exception, relay.Permission) error
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
	if !fields(f, "profile", "operation", "host_port", "cidr", "allow_http", "allow_private", "old_host", "old_cidr", "old_origin") {
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
	case "add", "edit":
		permission := relay.Permission{HostPort: f.SingleValue("host_port"), HTTP: f.SingleValue("allow_http") == "1", Private: f.SingleValue("allow_private") == "1"}
		if cidr := f.SingleValue("cidr"); cidr != "" {
			permission.Prefix, err = netip.ParsePrefix(cidr)
			if err != nil {
				err = db.ErrInvalidInput
			}
		}
		if err == nil {
			if f.SingleValue("operation") == "edit" {
				err = store.EditRelayExceptionAsRoot(ctx, identity.SessionID, profile, old, permission)
			} else {
				err = store.SetRelayPermissionAsRoot(ctx, identity.SessionID, profile, permission)
			}
		}
	default:
		err = db.ErrInvalidInput
	}
	var diagnostic *relay.ValidationError
	switch {
	case errors.As(err, &diagnostic):
		relay.LogDiagnostic(ctx, profile, diagnostic.Diagnostic, time.Now())
		code := diagnostic.Diagnostic.Code
		h.putNoticeData(identity.SessionID, 0, "admin.relay.reason_"+code, relay.Remedy(code), nil)
	case errors.Is(err, db.ErrInvalidInput):
		h.putNoticeData(identity.SessionID, 0, "admin.relay.invalid_exception", "Check the host and port, the permission flags and the reserved CIDR.", nil)
	case err != nil:
		h.fail(w, err)
		return
	default:
		h.putNoticeData(identity.SessionID, 0, "admin.relay.saved", "Relay policy updated.", nil)
	}
	http.Redirect(w, r, "/admin/settings#relay-exceptions", http.StatusSeeOther)
}
