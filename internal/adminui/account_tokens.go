package adminui

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/adminweb"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/i18n"
	"github.com/ebrgr/TuneTrail/internal/webassets"
)

// redeemAccount accepts credentials only in a CSRF-protected POST body.
func (h *Handler) redeemAccount(w http.ResponseWriter, r *http.Request) {
	if r.URL.RawQuery != "" {
		h.fail(w, db.ErrInvalidInput)
		return
	}
	v := view{Redeem: true}
	status := http.StatusOK
	if r.Method == http.MethodPost {
		draft := peekRedeemUsername(r)
		f, err := h.csrf.ParseLoginForm(w, r, adminweb.FormLimits{MaxBytes: 8192, TempDir: h.tempDir})
		if errors.Is(err, adminweb.ErrCSRF) {
			// ISSUE-125: a stale or foreign preauth cookie is not a permission problem. Show the
			// form again with a fresh token; only the username is kept, never the token or password.
			v.RedeemUsername = draft
			h.renderRedeem(w, r, http.StatusForbidden, v, "This page expired or was opened in another tab. Try again.", "")
			return
		}
		if err != nil {
			h.fail(w, err)
			return
		}
		defer f.Close()
		if !fields(f, "username", "account_token", "password", "confirm_password") {
			h.fail(w, db.ErrInvalidInput)
			return
		}
		v.RedeemUsername = f.SingleValue("username")
		password := f.SingleValue("password")
		if password != f.SingleValue("confirm_password") {
			h.renderRedeem(w, r, http.StatusBadRequest, v, msgPasswordMismatch, "confirm_password")
			return
		}
		err = admin.RedeemProfileAccountToken(r.Context(), h.store, v.RedeemUsername, f.SingleValue("account_token"), password)
		switch {
		case err == nil:
			// The sign-in form explains the next step; no session is created here.
			http.Redirect(w, r, webassets.LoginDestination+"?redeemed=1", http.StatusSeeOther)
			return
		case errors.Is(err, db.ErrAccountToken):
			status = http.StatusBadRequest
			h.renderRedeem(w, r, status, v, adminText(i18n.ViewOf(w), "admin.account.invalid_token", "Invalid or expired account token."), "account_token")
			return
		case errors.Is(err, db.ErrInvalidInput):
			status = http.StatusBadRequest
			h.renderRedeem(w, r, status, v, adminText(i18n.ViewOf(w), "admin.account.invalid_password", "Use a password of at least 8 characters and at most 72 bytes."), "password")
			return
		default:
			h.fail(w, err)
			return
		}
	}
	h.renderRedeem(w, r, status, v, "", "")
}

// renderRedeem issues a fresh pre-session token and renders the redemption form.
func (h *Handler) renderRedeem(w http.ResponseWriter, r *http.Request, status int, v view, message, field string) {
	csrf, err := h.csrf.LoginToken(w, r)
	if err != nil {
		h.fail(w, err)
		return
	}
	v.CSRF, v.Error, v.RedeemField = csrf, message, field
	// An open session in this browser only ever contributes its own username to the notice.
	if id, err := adminweb.SessionIdentity(r, h.store); err == nil && id.User != nil {
		v.RedeemSession = id.User.Username
	}
	h.render(w, status, v)
}

// peekRedeemUsername reads a short username from the leading bytes of an urlencoded body and
// restores the body, so the CSRF parser sees the request unchanged. Nothing else is kept.
func peekRedeemUsername(r *http.Request) string {
	if r.Body == nil || !strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
		return ""
	}
	orig := r.Body
	head, _ := io.ReadAll(io.LimitReader(orig, 8193))
	r.Body = struct {
		io.Reader
		io.Closer
	}{io.MultiReader(bytes.NewReader(head), orig), orig}
	if len(head) > 8192 {
		return ""
	}
	values, err := url.ParseQuery(string(head))
	if err != nil || len(values.Get("username")) > 64 {
		return ""
	}
	return values.Get("username")
}
