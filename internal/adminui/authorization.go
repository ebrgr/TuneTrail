package adminui

import (
	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/db"
	"net/http"
	"strings"
)

// authorizePage leaves the common user's landing destination to ISSUE-163 while
// permitting explicit existing account and personal management documents.
func (h *Handler) authorizePage(r *http.Request, id admin.Identity) error {
	switch r.URL.Query().Get("section") {
	case "account":
		return admin.AuthorizeAction(id.User, admin.ManageAccount, nil)
	case "profiles", "keys", "relay", "import", "export":
		if raw := r.URL.Query().Get("profile"); raw != "" {
			n, err := parseID(raw)
			if err != nil {
				return err
			}
			_, err = admin.AuthorizeProfile(r.Context(), h.store, id.User, n)
			return err
		}
	}
	return admin.AuthorizeAction(id.User, admin.ManageServer, nil)
}

// authorizeAction is an explicit capability inventory; future routes fail closed.
func (h *Handler) authorizeAction(r *http.Request, id admin.Identity) error {
	path := strings.TrimPrefix(r.URL.Path, "/admin/")
	switch path {
	case "logout", "account/username", "account/password":
		return admin.AuthorizeAction(id.User, admin.ManageAccount, nil)
	case "profile/create":
		return admin.AuthorizeAction(id.User, admin.ManageAccounts, nil)
	case "settings/save", "settings/subsonic/test":
		return admin.AuthorizeAction(id.User, admin.ManageServer, nil)
	}
	parts := strings.Split(path, "/")
	if len(parts) < 3 || parts[0] != "profile" {
		return admin.ErrForbidden
	}
	n, err := parseID(parts[1])
	if err != nil {
		return err
	}
	op := strings.Join(parts[2:], "/")
	switch op {
	case "delete":
		return admin.AuthorizeAction(id.User, admin.ManageAccounts, nil)
	case "save", "avatar", "avatar-clear", "import", "keys/create", "relay/save", "relay/test", "relay/remove":
	default:
		if len(parts) != 5 || parts[2] != "keys" || (parts[4] != "label" && parts[4] != "revoke") {
			return admin.ErrForbidden
		}
	}
	_, err = admin.AuthorizeProfile(r.Context(), h.store, id.User, n)
	return err
}

// personalMediaRoute permits only the existing personal tasks and reviewed confirmations.
func personalMediaRoute(r *http.Request, parts []string) bool {
	if len(parts) != 5 {
		return false
	}
	if r.Method == "POST" {
		return parts[4] == "listen-add" || parts[4] == "listen-delete" || parts[4] == "confirm"
	}
	switch parts[4] {
	case "listens", "add-listen", "listen":
		return parts[2] == "track"
	case "delete":
		return true
	case "review":
		return r.URL.Query().Get("action") == "delete"
	}
	return false
}

// authorizePersonalPlan rechecks ownership before both preview and application.
func (h *Handler) authorizePersonalPlan(r *http.Request, identity admin.Identity, profileID int32) error {
	if profileID <= 0 {
		return db.ErrInvalidInput
	}
	_, err := admin.AuthorizeProfile(r.Context(), h.store, identity.User, profileID)
	return err
}
