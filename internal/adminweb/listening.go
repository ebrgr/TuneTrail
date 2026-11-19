package adminweb

import (
	"net/http"

	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/models"
)

// CanReadListening resolves a human session only when anonymous access is denied.
func CanReadListening(r *http.Request, sessions admin.SessionReader, profile *models.ListeningProfile) bool {
	if admin.AuthorizeAction(nil, admin.ReadListening, profile) == nil {
		return true
	}
	if sessions == nil {
		return false
	}
	actor, err := SessionIdentity(r, sessions)
	return err == nil && admin.AuthorizeAction(actor.User, admin.ReadListening, profile) == nil
}
