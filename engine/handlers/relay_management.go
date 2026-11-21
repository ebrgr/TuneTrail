package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/netip"

	"github.com/ebrgr/TuneTrail/engine/middleware"
	"github.com/ebrgr/TuneTrail/internal/adminweb"
	"github.com/ebrgr/TuneTrail/internal/cfg"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/relay"
	"github.com/ebrgr/TuneTrail/internal/utils"
	"github.com/google/uuid"
)

// RelayManagementStore binds administrative changes to a fresh root session.
type RelayManagementStore interface {
	SetRelayPermissionAsRoot(context.Context, uuid.UUID, int32, relay.Permission) error
	RemoveRelayPermissionAsRoot(context.Context, uuid.UUID, int32, string, netip.Prefix) error
	ResumeRelay(context.Context, uuid.UUID, int32) error
	ConvertOccurrence(context.Context, uuid.UUID, int32) error
}

// RelayManagementHandler handles root actions scoped to a listening profile.
func RelayManagementHandler(store RelayManagementStore, action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		actor, profile := middleware.GetUserFromContext(ctx), middleware.ProfileFromContext(ctx)
		if actor == nil || profile == nil {
			utils.WriteError(w, "root session and profile required", http.StatusUnauthorized)
			return
		}
		var err error
		switch action {
		case "permission":
			r.Body = http.MaxBytesReader(w, r.Body, 4096)
			body, decodeErr := adminweb.DecodeManagementJSON[struct {
				HostPort string `json:"host_port"`
				CIDR     string `json:"cidr"`
				HTTP     bool   `json:"allow_http"`
				Private  bool   `json:"allow_private"`
			}](r)
			permission := relay.Permission{HostPort: body.HostPort, HTTP: body.HTTP, Private: body.Private}
			if decodeErr == nil && body.CIDR != "" {
				permission.Prefix, decodeErr = netip.ParsePrefix(body.CIDR)
			}
			if decodeErr != nil {
				err = db.ErrInvalidInput
			} else if r.Method == http.MethodDelete {
				err = store.RemoveRelayPermissionAsRoot(ctx, actor.SessionID, profile.ID, permission.HostPort, permission.Prefix)
			} else {
				err = store.SetRelayPermissionAsRoot(ctx, actor.SessionID, profile.ID, permission)
			}
		case "resume":
			err = store.ResumeRelay(ctx, actor.SessionID, profile.ID)
		case "occurrence":
			if !cfg.ActiveIngestEnabled() {
				err = db.ErrConflict
			} else {
				err = store.ConvertOccurrence(ctx, actor.SessionID, profile.ID)
			}
		default:
			err = db.ErrNotFound
		}
		if err != nil {
			var diagnostic *relay.ValidationError
			if errors.As(err, &diagnostic) {
				writeRelayDiagnostic(w, r, diagnostic, http.StatusBadGateway)
				return
			}
			utils.WriteError(w, adminweb.PublicError(err), adminweb.ErrorStatus(err))
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
