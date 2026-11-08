package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/ebrgr/TuneTrail/internal/adminweb"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/logger"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/ebrgr/TuneTrail/internal/utils"
	"github.com/go-chi/chi/v5"
)

type profileKey struct{}

func ProfileFromContext(ctx context.Context) *models.ListeningProfile {
	p, _ := ctx.Value(profileKey{}).(*models.ListeningProfile)
	return p
}
func withProfile(ctx context.Context, p *models.ListeningProfile) context.Context {
	return db.WithProfile(context.WithValue(ctx, profileKey{}, p), p.ID)
}

// ProfileScope resolves an explicit profile without masking store failures as absence.
func ProfileScope(store db.ProfileStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			slug := chi.URLParam(r, "slug")
			var p *models.ListeningProfile
			if slug != "" {
				var err error
				p, err = store.GetListeningProfile(r.Context(), slug)
				if err != nil {
					if errors.Is(err, db.ErrNotFound) {
						utils.WriteError(w, "profile not found", http.StatusNotFound)
					} else {
						logger.FromContext(r.Context()).Err(err).Msg("Profile lookup failed")
						utils.WriteError(w, "failed to resolve profile", http.StatusInternalServerError)
					}
					return
				}
			} else {
				profiles, err := store.ListListeningProfiles(r.Context())
				if err != nil {
					utils.WriteError(w, "failed to resolve profile", 500)
					return
				}
				if len(profiles) != 1 {
					utils.WriteError(w, "select an explicit profile", 409)
					return
				}
				p = &profiles[0]
			}
			next.ServeHTTP(w, r.WithContext(withProfile(r.Context(), p)))
		})
	}
}

// IngestionKey binds a valid player credential to its profile and reports store failures separately.
func IngestionKey(store db.ProfileStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if len(header) < 7 || !strings.EqualFold(header[:6], "Token ") {
				utils.WriteError(w, "invalid API key", 401)
				return
			}
			p, err := store.GetProfileByAPIKey(r.Context(), strings.TrimSpace(header[6:]))
			if err != nil {
				if errors.Is(err, db.ErrNotFound) {
					utils.WriteError(w, "invalid API key", http.StatusUnauthorized)
				} else {
					logger.FromContext(r.Context()).Err(err).Msg("Ingestion key lookup failed")
					utils.WriteError(w, "failed to validate API key", http.StatusInternalServerError)
				}
				return
			}
			next.ServeHTTP(w, r.WithContext(withProfile(r.Context(), p)))
		})
	}
}

// RequireAdmin shares authentication errors and cache policy with HTML adapters.
func RequireAdmin(store db.UserStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-store")
			u, err := validateSession(r.Context(), store, r)
			if err != nil {
				if adminweb.ErrorStatus(err) == 500 {
					logger.FromContext(r.Context()).Err(err).Msg("Administrative session lookup failed")
				}
				utils.WriteError(w, adminweb.PublicError(err), adminweb.ErrorStatus(err))
				return
			}
			if u.Role != models.UserRoleAdmin {
				utils.WriteError(w, "administrator required", 403)
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), UserContextKey, u)))
		})
	}
}

// SameOrigin delegates the shared policy used by JSON and prepared HTML handlers.
func SameOrigin(next http.Handler) http.Handler { return adminweb.SameOrigin(next) }
