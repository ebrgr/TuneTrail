package adminweb

import (
	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/google/uuid"
	"net/http"
)

// SecureRequest preserves the existing TLS/proxy deployment contract.
func SecureRequest(r *http.Request) bool {
	return r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
}

// SetSessionCookie uses the expiration actually committed by the authentication store.
func SetSessionCookie(w http.ResponseWriter, r *http.Request, s *models.Session) {
	http.SetCookie(w, &http.Cookie{Name: "koito_session", Value: s.ID.String(), Expires: s.ExpiresAt, Path: "/", HttpOnly: true, Secure: SecureRequest(r), SameSite: http.SameSiteStrictMode})
}

// ClearSessionCookie mirrors the session cookie's security attributes.
func ClearSessionCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: "koito_session", Value: "", Path: "/", HttpOnly: true, Secure: SecureRequest(r), SameSite: http.SameSiteStrictMode, MaxAge: -1})
}

// SessionIdentity rejects ambiguous cookies and resolves the actor again on every request.
func SessionIdentity(r *http.Request, store admin.SessionReader) (admin.Identity, error) {
	value, ok := oneCookie(r, "koito_session")
	if !ok {
		return admin.Identity{}, admin.ErrUnauthorized
	}
	id, err := uuid.Parse(value)
	if err != nil {
		return admin.Identity{}, admin.ErrUnauthorized
	}
	return admin.Authenticate(r.Context(), store, id)
}
