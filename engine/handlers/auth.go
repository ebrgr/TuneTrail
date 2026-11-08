package handlers

import (
	"github.com/ebrgr/TuneTrail/engine/middleware"
	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/adminweb"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/logger"
	"github.com/ebrgr/TuneTrail/internal/utils"
	"github.com/google/uuid"
	"net/http"
)

// LoginHandler keeps the JSON contract while sharing authentication with future HTML.
func LoginHandler(store db.UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		r.Body = http.MaxBytesReader(w, r.Body, 4096)
		body, err := utils.DecodeBody[struct {
			Username   string `json:"username"`
			Password   string `json:"password"`
			RememberMe bool   `json:"remember_me"`
		}](r)
		if err != nil {
			utils.WriteError(w, "invalid request", 400)
			return
		}
		session, err := admin.Login(r.Context(), store, body.Username, body.Password, body.RememberMe)
		if err != nil {
			logger.FromContext(r.Context()).Err(err).Msg("Administrative login failed")
			message := adminweb.PublicError(err)
			if adminweb.ErrorStatus(err) == 401 {
				message = "invalid credentials"
			}
			utils.WriteError(w, message, adminweb.ErrorStatus(err))
			return
		}
		adminweb.SetSessionCookie(w, r, session)
		adminweb.ClearPreauth(w, r)
		w.WriteHeader(204)
	}
}

// LogoutHandler reports a storage failure instead of pretending revocation succeeded.
func LogoutHandler(store db.UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		// Reject ambiguous session cookies before choosing or revoking any session.
		count := 0
		for _, cookie := range r.Cookies() {
			if cookie.Name == "koito_session" {
				count++
			}
		}
		if count > 1 {
			utils.WriteError(w, "ambiguous session cookie", http.StatusBadRequest)
			return
		}
		if cookie, err := r.Cookie("koito_session"); err == nil {
			if id, err := uuid.Parse(cookie.Value); err == nil {
				if err := admin.Logout(r.Context(), store, id); err != nil {
					logger.FromContext(r.Context()).Err(err).Msg("Administrative logout failed")
					utils.WriteError(w, "operation failed", 500)
					return
				}
			}
		}
		adminweb.ClearSessionCookie(w, r)
		adminweb.ClearPreauth(w, r)
		w.WriteHeader(204)
	}
}

// MeHandler exposes only the administrator resolved by the session middleware.
func MeHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		user := middleware.GetUserFromContext(r.Context())
		if user == nil {
			utils.WriteError(w, "administrator session required", 401)
			return
		}
		utils.WriteJSON(w, 200, user)
	}
}

// UpdateUserHandler clears the browser session after an atomic password change.
func UpdateUserHandler(store db.UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		r.Body = http.MaxBytesReader(w, r.Body, 4096)
		body, err := utils.DecodeBody[struct {
			Username *string `json:"username"`
			Password *string `json:"password"`
		}](r)
		if err != nil {
			utils.WriteError(w, "invalid request", 400)
			return
		}
		changed, err := admin.ChangeAccount(r.Context(), store, middleware.GetUserFromContext(r.Context()), body.Username, body.Password)
		if err != nil {
			logger.FromContext(r.Context()).Err(err).Msg("Administrative account update failed")
			utils.WriteError(w, adminweb.PublicError(err), adminweb.ErrorStatus(err))
			return
		}
		if changed {
			adminweb.ClearSessionCookie(w, r)
			adminweb.ClearPreauth(w, r)
		}
		w.WriteHeader(204)
	}
}
