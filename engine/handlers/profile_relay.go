package handlers

import (
	"errors"
	"github.com/ebrgr/TuneTrail/engine/middleware"
	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/adminweb"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/relay"
	"github.com/ebrgr/TuneTrail/internal/utils"
	"net/http"
)

type profileRelay struct {
	Enabled bool   `json:"enabled"`
	URL     string `json:"url"`
	Token   string `json:"token"`
}

// decodeProfileRelay retains the JSON size and destination validation contract.
func decodeProfileRelay(w http.ResponseWriter, r *http.Request) (profileRelay, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, 8192)
	body, err := utils.DecodeBody[profileRelay](r)
	if err != nil {
		utils.WriteError(w, "valid relay base URL and token required", 400)
		return profileRelay{}, false
	}
	body.URL, body.Token, err = relay.Normalize(body.URL, body.Token)
	if err != nil {
		utils.WriteError(w, "valid relay base URL and token required", 400)
		return profileRelay{}, false
	}
	return body, true
}

// GetProfileRelayHandler returns secrets only in a no-store admin response.
func GetProfileRelayHandler(store db.ProfileStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		p := middleware.ProfileFromContext(r.Context())
		url, token, err := admin.Relay(r.Context(), store, middleware.GetUserFromContext(r.Context()), p.ID, p.Slug)
		if err != nil {
			utils.WriteError(w, adminweb.PublicError(err), adminweb.ErrorStatus(err))
			return
		}
		utils.WriteJSON(w, 200, profileRelay{Enabled: url != "" && token != "", URL: url, Token: token})
	}
}

// SetProfileRelayHandler shares profile scope and normalization with HTML.
func SetProfileRelayHandler(store db.ProfileStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := middleware.ProfileFromContext(r.Context())
		user := middleware.GetUserFromContext(r.Context())
		var err error
		if r.Method == "DELETE" {
			err = admin.RemoveRelay(r.Context(), store, user, p.ID, p.Slug)
		} else {
			body, ok := decodeProfileRelay(w, r)
			if !ok {
				return
			}
			err = admin.SaveRelay(r.Context(), store, user, p.ID, p.Slug, body.URL, body.Token)
		}
		if err != nil {
			utils.WriteError(w, adminweb.PublicError(err), adminweb.ErrorStatus(err))
			return
		}
		w.WriteHeader(204)
	}
}

// TestProfileRelayHandler uses the same bounded network policy as actual forwarding.
func TestProfileRelayHandler(store db.ProfileStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		body, ok := decodeProfileRelay(w, r)
		if !ok {
			return
		}
		p := middleware.ProfileFromContext(r.Context())
		name, err := admin.TestRelay(r.Context(), store, middleware.GetUserFromContext(r.Context()), p.ID, p.Slug, body.URL, body.Token, relay.Client{})
		if err != nil {
			if errors.Is(err, relay.ErrUpstream) {
				utils.WriteError(w, "relay validation unavailable", 502)
			} else {
				utils.WriteError(w, adminweb.PublicError(err), adminweb.ErrorStatus(err))
			}
			return
		}
		utils.WriteJSON(w, 200, map[string]any{"valid": true, "user_name": name})
	}
}
