package adminui

import (
	"errors"
	"net/http"

	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/adminweb"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/i18n"
)

// redeemAccount accepts credentials only in a CSRF-protected POST body.
func (h *Handler) redeemAccount(w http.ResponseWriter, r *http.Request) {
	if r.URL.RawQuery != "" {
		h.fail(w, db.ErrInvalidInput)
		return
	}
	status := http.StatusOK
	message := ""
	if r.Method == http.MethodPost {
		f, err := h.csrf.ParseLoginForm(w, r, adminweb.FormLimits{MaxBytes: 8192, TempDir: h.tempDir})
		if err != nil {
			h.fail(w, err)
			return
		}
		defer f.Close()
		if !fields(f, "username", "account_token", "password") {
			h.fail(w, db.ErrInvalidInput)
			return
		}
		err = admin.RedeemProfileAccountToken(r.Context(), h.store, f.SingleValue("username"), f.SingleValue("account_token"), f.SingleValue("password"))
		switch {
		case err == nil:
			http.Redirect(w, r, "/admin", http.StatusSeeOther)
			return
		case errors.Is(err, db.ErrAccountToken):
			status = http.StatusBadRequest
			message = adminText(i18n.ViewOf(w), "admin.account.invalid_token", "Invalid or expired account token.")
		case errors.Is(err, db.ErrInvalidInput):
			status = http.StatusBadRequest
			message = adminText(i18n.ViewOf(w), "admin.account.invalid_password", "Use a password of at least 8 characters and at most 72 bytes.")
		default:
			h.fail(w, err)
			return
		}
	}
	csrf, err := h.csrf.LoginToken(w, r)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.render(w, status, view{Redeem: true, CSRF: csrf, Error: message})
}
