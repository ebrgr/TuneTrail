package adminui

import (
	"errors"
	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/adminweb"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/i18n"
	"github.com/ebrgr/TuneTrail/internal/importer"
	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/ebrgr/TuneTrail/internal/logger"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/ebrgr/TuneTrail/internal/relay"
	"github.com/ebrgr/TuneTrail/internal/webassets"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// action validates the complete body, including CSRF, before calling any service.
func (h *Handler) action(w http.ResponseWriter, r *http.Request, identity admin.Identity) {
	path := strings.TrimPrefix(r.URL.Path, "/admin/")
	if path == r.URL.Path || path == "" {
		textError(w, "item not found", http.StatusNotFound)
		return
	}
	// Bound image admission before multipart parsing can buffer a request.
	if path == "profile/create" || strings.HasSuffix(path, "/avatar") {
		ctx, release, err := admin.AcquireImageAdmission(r.Context())
		if err != nil {
			h.fail(w, err)
			return
		}
		defer release()
		r = r.WithContext(ctx)
	}
	max := int64(8192)
	if path == "profile/create" || strings.HasSuffix(path, "/avatar") {
		max = 5 << 20
	}
	if strings.HasSuffix(path, "/import") {
		max = 100 << 20
	}
	if path == "settings/save" || path == "settings/subsonic/test" {
		max = 64 << 10
	}
	f, err := h.csrf.ParseSessionForm(w, r, identity, adminweb.FormLimits{MaxBytes: max, TempDir: h.tempDir})
	if err != nil {
		h.fail(w, err)
		return
	}
	defer f.Close()
	if err = h.consumeTicket(identity.SessionID, f.SingleValue("submission_token"), path == "logout"); err != nil {
		h.fail(w, err)
		return
	}
	switch path {
	case "settings/relay":
		h.relaySettingsAction(w, r, identity, f)
		return
	case "settings/save", "settings/subsonic/test":
		h.settingsAction(w, r, identity, f, path)
		return
	case "logout":
		if !fields(f) {
			h.fail(w, db.ErrInvalidInput)
			return
		}
		if err = admin.Logout(r.Context(), h.store, identity.SessionID); err != nil {
			logger.FromContext(r.Context()).Error().Str("reason", "session revocation failed").Msg("Logout failed")
			h.fail(w, err)
			return
		}
		adminweb.ClearSessionCookie(w, r)
		adminweb.ClearPreauth(w, r)
		// Signing out lands on the public directory, which offers Sign in.
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	case "account/username", "account/password":
		if (path == "account/username" && !fields(f, "username")) || (path == "account/password" && !fields(f, "password", "confirm_password")) {
			h.fail(w, db.ErrInvalidInput)
			return
		}
		var username, password *string
		if path == "account/username" {
			v := f.SingleValue("username")
			username = &v
		} else {
			v := f.SingleValue("password")
			if v != f.SingleValue("confirm_password") {
				h.fail(w, db.ErrInvalidInput)
				return
			}
			password = &v
		}
		changed, e := admin.ChangeAccount(r.Context(), h.store, identity.User, username, password)
		if e != nil {
			h.fail(w, e)
			return
		}
		if changed {
			adminweb.ClearSessionCookie(w, r)
			adminweb.ClearPreauth(w, r)
		} else {
			h.putNotice(identity.SessionID, 0, "Account updated.")
		}
		http.Redirect(w, r, webassets.AccountDestination, 303)
		return
	case "profile/create":
		if !fields(f, "display_name", "slug", "icon", "image") {
			h.fail(w, db.ErrInvalidInput)
			return
		}
		p, e := admin.CreateProfile(r.Context(), h.store, identity.User, f.SingleValue("display_name"), f.SingleValue("slug"), f.SingleValue("icon"))
		if e != nil {
			h.fail(w, e)
			return
		}
		if image, e := openImage(f); e != nil {
			h.render(w, 500, view{Error: "Profile created, but the photo could not be saved. Retry in Avatar.", Profile: p})
			return
		} else if image != nil {
			defer image.Close()
			if _, e = admin.SetAvatar(r.Context(), h.store, identity.User, p.ID, p.Slug, f.SingleValue("icon"), image); e != nil {
				h.render(w, 200, view{Error: "Profile created, but the photo could not be saved. Retry in Avatar.", Profile: p})
				return
			}
		}
		h.putNotice(identity.SessionID, p.ID, "Profile created with an initial key.")
		http.Redirect(w, r, profilePath(p.ID), 303)
		return
	}
	// All profile actions carry both a stable ID in the route and the slug from the form.
	parts := strings.Split(path, "/")
	if len(parts) < 3 || parts[0] != "profile" {
		textError(w, "item not found", http.StatusNotFound)
		return
	}
	id, err := parseID(parts[1])
	if err != nil {
		h.fail(w, err)
		return
	}
	slug := f.SingleValue("profile_slug")
	operation := strings.Join(parts[2:], "/")
	target := profilePath(id)
	profile, err := admin.Profile(r.Context(), h.store, identity.User, id, slug)
	if err != nil {
		h.fail(w, err)
		return
	}
	switch operation {
	case "account-token":
		if !fields(f, "profile_slug") {
			err = db.ErrInvalidInput
			break
		}
		var token string
		token, err = admin.IssueProfileAccountToken(r.Context(), h.store, identity.SessionID, id)
		if err == nil {
			// Only this response receives the plaintext. Never include it in a redirect or notice.
			w.Header().Set("Referrer-Policy", "no-referrer")
			h.render(w, http.StatusOK, view{IssuedToken: token, Profile: profile})
			return
		}
	case "account-role":
		if !fields(f, "profile_slug", "role") {
			err = db.ErrInvalidInput
			break
		}
		err = admin.ChangeProfileAccountRole(r.Context(), h.store, identity.SessionID, id, models.UserRole(f.SingleValue("role")))
	case "import":
		if !fields(f, "profile_slug", "profile_name", "confirm", "format", "file") || f.SingleValue("confirm") != "1" {
			err = db.ErrInvalidInput
			break
		}
		current, e := admin.Profile(r.Context(), h.store, identity.User, id, slug)
		if e != nil {
			err = e
			break
		}
		if current.DisplayName != f.SingleValue("profile_name") {
			err = db.ErrConflict
			break
		}
		upload, ok := f.Files["file"]
		if !ok {
			err = db.ErrInvalidInput
			break
		}
		result, e := admin.ImportHistory(r.Context(), h.store, identity.User, h.musicBrainz, id, f.SingleValue("format"), upload.Path)
		summaryKey, importSummary := "admin.import.summary", "Processed: {{.Processed}}. Imported: {{.Imported}}. Duplicates: {{.Duplicates}}. Ignored: {{.Ignored}}. Errors: {{.Errors}}."
		values := map[string]any{"Processed": result.Processed, "Imported": result.Imported, "Duplicates": result.Duplicates, "Ignored": result.Ignored, "Errors": result.Errors}
		// Active imports acknowledge durable receipts before the worker writes history.
		if result.Active {
			summaryKey = "admin.import.activeSummary"
			importSummary = "Processed: {{.Processed}}. Pending: {{.Pending}}. Duplicates: {{.Duplicates}}. Ignored: {{.Ignored}}. Errors: {{.Errors}}."
			values["Pending"] = result.Pending
		}
		if e != nil {
			status := http.StatusInternalServerError
			// Active receipt storage failures can be retried after the writer recovers.
			var unavailable *ingest.ErrUnavailable
			switch {
			case errors.As(e, &unavailable):
				w.Header().Set("Retry-After", "1")
				status = http.StatusServiceUnavailable
			case errors.Is(e, importer.ErrInvalidFile), errors.Is(e, db.ErrInvalidInput):
				status = http.StatusBadRequest
			}
			h.render(w, status, view{Error: adminText(i18n.ViewOf(w), "admin.import.failed", "Import failed; some records may already be saved.") + " " + adminFormat(i18n.ViewOf(w), summaryKey, importSummary, values), Profile: current})
			return
		}
		h.putNoticeData(identity.SessionID, id, summaryKey, importSummary, values)
		http.Redirect(w, r, target, http.StatusSeeOther)
		return
	case "delete":
		if !fields(f, "profile_slug", "display_name", "listen_count", "confirm") || f.SingleValue("confirm") != "1" {
			err = db.ErrInvalidInput
			break
		}
		var count int64
		count, err = strconv.ParseInt(f.SingleValue("listen_count"), 10, 64)
		if err != nil || count < 0 {
			err = db.ErrInvalidInput
			break
		}
		err = admin.DeleteProfile(r.Context(), h.store, identity.User, id, slug, f.SingleValue("display_name"), count)
		if err == nil {
			// A confirmed self-deletion revokes the account's sessions in the store.
			if profile.UserID == identity.User.ID {
				adminweb.ClearSessionCookie(w, r)
				adminweb.ClearPreauth(w, r)
			} else {
				h.putNotice(identity.SessionID, 0, "Profile deleted; the shared catalog remains.")
			}
			http.Redirect(w, r, "/admin", http.StatusSeeOther)
			return
		}
	case "save":
		if !fields(f, "profile_slug", "display_name", "slug", "require_login_present", "require_login") {
			err = db.ErrInvalidInput
			break
		}
		// The marker distinguishes an unchecked box from a legacy identity edit.
		var visibility *bool
		marker, value := f.Values["require_login_present"], f.Values["require_login"]
		if len(marker) != 0 || len(value) != 0 {
			if len(marker) != 1 || marker[0] != "1" || (len(value) > 0 && (len(value) != 1 || value[0] != "1")) {
				err = db.ErrInvalidInput
				break
			}
			enabled := len(value) == 1
			visibility = &enabled
		}
		_, err = admin.UpdateProfile(r.Context(), h.store, identity.User, id, slug, f.SingleValue("display_name"), f.SingleValue("slug"), visibility)
	case "avatar", "avatar-clear":
		if !fields(f, "profile_slug", "icon", "image") {
			err = db.ErrInvalidInput
			break
		}
		imageFile, e := openImage(f)
		err = e
		if err != nil {
			break
		}
		if imageFile != nil {
			defer imageFile.Close()
		}
		icon := f.SingleValue("icon")
		if operation == "avatar-clear" {
			if imageFile != nil {
				err = db.ErrInvalidInput
				break
			}
			icon = ""
		}
		var image io.ReadSeeker
		if imageFile != nil {
			image = imageFile
		}
		_, err = admin.SetAvatar(r.Context(), h.store, identity.User, id, slug, icon, image)
	case "keys/create":
		if !fields(f, "profile_slug", "label") {
			err = db.ErrInvalidInput
			break
		}
		_, err = admin.CreateKey(r.Context(), h.store, identity.User, id, slug, f.SingleValue("label"))
	case "relay/save", "relay/test", "relay/remove":
		if !fields(f, "profile_slug", "url", "token") {
			err = db.ErrInvalidInput
			break
		}
		raw, token := f.SingleValue("url"), f.SingleValue("token")
		if operation == "relay/save" {
			err = admin.SaveRelay(r.Context(), h.store, identity.User, id, slug, raw, token)
		} else if operation == "relay/remove" {
			err = admin.RemoveRelay(r.Context(), h.store, identity.User, id, slug)
		} else {
			_, err = admin.TestRelay(r.Context(), h.store, identity.User, id, slug, raw, token, h.relayClient)
			if err == nil {
				h.putNotice(identity.SessionID, id, "Relay connection successful; configuration was not changed.")
			}
			var diagnostic *relay.ValidationError
			if errors.As(err, &diagnostic) {
				relay.LogDiagnostic(r.Context(), id, diagnostic.Diagnostic, time.Now())
				if admin.AuthorizeAction(identity.User, admin.ManageServer, nil) == nil {
					code := diagnostic.Diagnostic.Code
					h.putNoticeData(identity.SessionID, id, "admin.relay.reason_"+code, relay.Remedy(code), nil)
				} else {
					class := diagnostic.PublicClass()
					h.putNoticeData(identity.SessionID, id, "admin.relay.class_"+class, class, nil)
				}
				err = nil
			}
		}
	default:
		kp := strings.Split(operation, "/")
		if len(kp) != 3 || kp[0] != "keys" {
			textError(w, "item not found", http.StatusNotFound)
			return
		}
		var key int32
		key, err = parseID(kp[1])
		if err != nil {
			break
		}
		if kp[2] == "label" && fields(f, "profile_slug", "label") {
			err = admin.RenameKey(r.Context(), h.store, identity.User, id, slug, key, f.SingleValue("label"))
		} else if kp[2] == "revoke" && fields(f, "profile_slug", "confirm") && f.SingleValue("confirm") == "1" {
			err = admin.RevokeKey(r.Context(), h.store, identity.User, id, slug, key)
		} else {
			err = db.ErrInvalidInput
		}
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	if operation != "relay/test" {
		h.putNotice(identity.SessionID, id, "Changes saved.")
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}
