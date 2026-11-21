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
	"time"
)

type profileRelay struct {
	Enabled bool   `json:"enabled"`
	URL     string `json:"url"`
	Token   string `json:"token,omitempty"`
}

// decodeProfileRelay retains the JSON size and destination validation contract.
func decodeProfileRelay(w http.ResponseWriter, r *http.Request, diagnostic bool) (profileRelay, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, 8192)
	body, err := utils.DecodeBody[profileRelay](r)
	if err != nil {
		utils.WriteError(w, "valid relay base URL and token required", 400)
		return profileRelay{}, false
	}
	// A blank token may reuse the saved credential for an unchanged endpoint.
	validationToken := body.Token
	if validationToken == "" {
		validationToken = "placeholder"
	}
	rawURL := body.URL
	body.URL, _, err = relay.Normalize(body.URL, validationToken)
	if err != nil {
		if diagnostic {
			code := "token_invalid"
			if _, _, baseErr := relay.Normalize(rawURL, "placeholder"); baseErr != nil {
				code = "url_invalid"
			}
			writeRelayDiagnostic(w, r, &relay.ValidationError{Class: relay.PolicyBlocked, Diagnostic: relay.Describe(rawURL, code, "")}, http.StatusBadRequest)
		} else {
			utils.WriteError(w, "valid relay base URL and token required", 400)
		}
		return profileRelay{}, false
	}
	return body, true
}

// GetProfileRelayHandler returns saved state without the credential.
func GetProfileRelayHandler(store db.ProfileStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		p := middleware.ProfileFromContext(r.Context())
		url, token, err := admin.Relay(r.Context(), store, middleware.GetUserFromContext(r.Context()), p.ID, p.Slug)
		if err != nil {
			utils.WriteError(w, adminweb.PublicError(err), adminweb.ErrorStatus(err))
			return
		}
		utils.WriteJSON(w, 200, profileRelay{Enabled: url != "" && token != "", URL: url})
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
			body, ok := decodeProfileRelay(w, r, false)
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
		body, ok := decodeProfileRelay(w, r, true)
		if !ok {
			return
		}
		p := middleware.ProfileFromContext(r.Context())
		name, err := admin.TestRelay(r.Context(), store, middleware.GetUserFromContext(r.Context()), p.ID, p.Slug, body.URL, body.Token, relay.Client{})
		if err != nil {
			var diagnostic *relay.ValidationError
			if errors.As(err, &diagnostic) {
				writeRelayDiagnostic(w, r, diagnostic, http.StatusBadGateway)
			} else if errors.Is(err, relay.ErrUpstream) {
				utils.WriteJSON(w, http.StatusBadGateway, map[string]string{"class": "network"})
			} else {
				utils.WriteError(w, adminweb.PublicError(err), adminweb.ErrorStatus(err))
			}
			return
		}
		utils.WriteJSON(w, 200, map[string]any{"valid": true, "user_name": name})
	}
}

// writeRelayDiagnostic gives only a broad class to callers who are not instance administrators.
func writeRelayDiagnostic(w http.ResponseWriter, r *http.Request, d *relay.ValidationError, status int) {
	ctx := r.Context()
	if p := middleware.ProfileFromContext(ctx); p != nil {
		relay.LogDiagnostic(ctx, p.ID, d.Diagnostic, time.Now())
	}
	body := map[string]string{"class": d.PublicClass()}
	if admin.AuthorizeAction(middleware.GetUserFromContext(ctx), admin.ManageServer, nil) == nil {
		body["reason"] = d.Diagnostic.Code
		body["instruction"] = relay.Remedy(d.Diagnostic.Code)
		body["scheme"] = d.Diagnostic.Scheme
		body["host_port"] = d.Diagnostic.HostPort()
		body["range_category"] = d.Diagnostic.Category
	}
	utils.WriteJSON(w, status, body)
}
