// Package adminui serves protected administration documents and form actions.
package adminui

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/adminweb"
	"github.com/ebrgr/TuneTrail/internal/cfg"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/export"
	"github.com/ebrgr/TuneTrail/internal/i18n"
	"github.com/ebrgr/TuneTrail/internal/images"
	"github.com/ebrgr/TuneTrail/internal/importer"
	"github.com/ebrgr/TuneTrail/internal/logger"
	"github.com/ebrgr/TuneTrail/internal/mbz"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/ebrgr/TuneTrail/internal/relay"
	"github.com/go-chi/httprate"
	"github.com/google/uuid"
)

//go:embed templates/*.html
var templates embed.FS

// Store contains the existing persistence contracts; no implementation-specific database is needed.
type Store interface {
	db.UserStore
	db.ProfileStore
	db.ImageStore
	db.ExportStore
	importer.UploadStore
}

// SettingsStore provides configuration without coupling it to the SQLite store.
type SettingsStore interface {
	Read(context.Context) (cfg.SettingsSnapshot, error)
	Save(context.Context, string, map[string]string) (cfg.SettingsSnapshot, error)
}
type Options struct {
	Languages              *i18n.Catalog
	Settings               SettingsStore
	SubsonicTest           func(context.Context, string, string) error
	RequireSubmissionToken bool
	DefaultTheme           string
	CSRF                   *adminweb.CSRF
	TempDir                string
	DisableRateLimit       bool
	RelayClient            relay.Client
	MusicBrainz            mbz.MusicBrainzCaller
	LoginMiddleware        func(http.Handler) http.Handler
}
type pendingCatalogPlan struct {
	Session     uuid.UUID
	ProfileID   int32
	ProfileSlug string
	RouteID     int32
	Kind        string
	Plan        db.CatalogPlan
	Expires     time.Time
}

type notice struct {
	profile   int32
	key, text string
	values    map[string]any
	expires   time.Time
}

// Handler serves administration in Go mode with session and form protection.
type Handler struct {
	languages         *i18n.Catalog
	settings          SettingsStore
	subsonicTest      func(context.Context, string, string) error
	requireSubmission bool
	submissions       map[uuid.UUID]submissionTicket
	store             Store
	csrf              *adminweb.CSRF
	tempDir           string
	relayClient       relay.Client
	musicBrainz       mbz.MusicBrainzCaller
	tmpl              *template.Template
	entry             http.Handler
	login             http.Handler
	mu                sync.Mutex
	notices           map[uuid.UUID]notice
	catalogPlans      map[uuid.UUID]pendingCatalogPlan
}
type view struct {
	Locale                                          *i18n.View
	User                                            *models.User
	Profiles                                        []models.ListeningProfile
	Profile                                         *models.ListeningProfile
	Keys                                            []models.ApiKey
	RelayURL, RelayToken, CSRF, Next, Notice, Error string
	Login                                           bool
	Section                                         string
	Submission                                      string
}

// New constructs the protected handler and a bounded pre-login endpoint.
func New(store Store, opts Options) (*Handler, error) {
	if store == nil {
		return nil, fmt.Errorf("admin store required")
	}
	csrf := opts.CSRF
	var err error
	if csrf == nil {
		csrf, err = adminweb.NewCSRF()
		if err != nil {
			return nil, err
		}
	}
	if opts.DefaultTheme == "" {
		opts.DefaultTheme = "yuu"
	}
	// Render profile initials using the same two-word rule as public navigation.
	initials := func(name string) string {
		var letters []rune
		for _, word := range strings.Fields(name) {
			letters = append(letters, []rune(word)[0])
			if len(letters) == 2 {
				break
			}
		}
		return strings.ToUpper(string(letters))
	}
	// Limit avatar classes to the same known icons as the public renderer.
	avatarIcon := func(icon *string) string {
		if icon == nil {
			return ""
		}
		return map[string]string{"music": "music", "headphones": "headphones", "disc": "disc", "radio": "radio", "user": "user"}[*icon]
	}
	tmpl, err := template.New("admin").Funcs(adminFuncs(nil)).Funcs(template.FuncMap{"defaultTheme": func() string { return opts.DefaultTheme }, "initials": initials, "avatarIcon": avatarIcon}).ParseFS(templates, "templates/*.html")
	if err != nil {
		return nil, err
	}
	if opts.SubsonicTest == nil {
		opts.SubsonicTest = images.TestSubsonic
	}
	h := &Handler{languages: opts.Languages, settings: opts.Settings, subsonicTest: opts.SubsonicTest, requireSubmission: opts.RequireSubmissionToken, submissions: map[uuid.UUID]submissionTicket{}, store: store, csrf: csrf, tempDir: opts.TempDir, relayClient: opts.RelayClient, musicBrainz: opts.MusicBrainz, tmpl: tmpl, notices: map[uuid.UUID]notice{}, catalogPlans: map[uuid.UUID]pendingCatalogPlan{}}
	h.login = http.HandlerFunc(h.postLogin)
	if opts.LoginMiddleware != nil {
		h.login = opts.LoginMiddleware(h.login)
	} else if !opts.DisableRateLimit {
		h.login = httprate.Limit(10, time.Minute)(h.login)
	}
	limitedLogin := h.login
	h.login = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limitedLogin.ServeHTTP(&loginResponse{ResponseWriter: w, handler: h}, r)
	})
	h.entry = adminweb.NoStore(adminweb.SameOriginWithReject(http.HandlerFunc(h.dispatch), func(w http.ResponseWriter, _ *http.Request) { h.fail(w, adminweb.ErrCSRF) }))
	return h, nil
}

// loginResponse localizes only HTML rate-limit responses; the shared JSON endpoint is unchanged.
type loginResponse struct {
	http.ResponseWriter
	handler  *Handler
	rejected bool
}

func (w *loginResponse) WriteHeader(status int) {
	if status == http.StatusTooManyRequests {
		w.rejected = true
		w.handler.render(w.ResponseWriter, status, view{Error: "too many requests"})
		return
	}
	w.ResponseWriter.WriteHeader(status)
}
func (w *loginResponse) Write(b []byte) (int, error) {
	if w.rejected {
		return len(b), nil
	}
	return w.ResponseWriter.Write(b)
}
func (w *loginResponse) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// ServeHTTP applies common no-store and origin protection before dispatch.
// headWriter preserves response headers while suppressing the document body.
type headWriter struct{ http.ResponseWriter }

func (w headWriter) Write(b []byte) (int, error) { return len(b), nil }
func (w headWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w = h.languages.Wrap(w, r)
	if r.Method == http.MethodHead {
		copy := r.Clone(r.Context())
		copy.Method = http.MethodGet
		h.entry.ServeHTTP(headWriter{w}, copy)
		return
	}
	h.entry.ServeHTTP(w, r)
}

// dispatch distinguishes login from session-only reads and actions on every request.
func (h *Handler) dispatch(w http.ResponseWriter, r *http.Request) {
	if !IsRoute(r.Method, r.URL.Path) {
		textError(w, "item not found", http.StatusNotFound)
		return
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) >= 2 && parts[1] == "manage" {
		h.media(w, r, parts)
		return
	}

	if r.URL.Path == "/admin/login" && r.Method == "POST" {
		h.login.ServeHTTP(w, r)
		return
	}
	if r.URL.Path == "/admin/settings" && r.Method == "GET" {
		// Resolve an administrative session before reading any configuration.
		identity, err := adminweb.SessionIdentity(r, h.store)
		if errors.Is(err, admin.ErrUnauthorized) && r.Header.Get("Authorization") == "" {
			h.loginPage(w, r, "")
			return
		}
		if err != nil {
			h.fail(w, err)
			return
		}
		h.settingsPage(w, r, identity)
		return
	}
	if r.URL.Path == "/admin" && (r.Method == "GET" || r.Method == "HEAD") {
		identity, err := adminweb.SessionIdentity(r, h.store)
		if errors.Is(err, admin.ErrUnauthorized) {
			h.loginPage(w, r, "")
			return
		}
		if err != nil {
			h.fail(w, err)
			return
		}
		h.page(w, r, identity)
		return
	}
	if r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/admin/profile/") && strings.HasSuffix(r.URL.Path, "/delete-review") {
		h.reviewProfile(w, r)
		return
	}
	if r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/admin/profile/") && strings.HasSuffix(r.URL.Path, "/export") {
		identity, err := adminweb.SessionIdentity(r, h.store)
		if err != nil {
			h.fail(w, err)
			return
		}
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) != 4 {
			textError(w, "item not found", http.StatusNotFound)
			return
		}
		id, err := parseID(parts[2])
		if err != nil {
			h.fail(w, err)
			return
		}
		profile, err := h.store.GetListeningProfileByID(r.Context(), id)
		if err != nil {
			h.fail(w, err)
			return
		}
		if _, err = admin.Profile(r.Context(), h.store, identity.User, id, profile.Slug); err != nil {
			h.fail(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", `attachment; filename="koito_export.json"`)
		if err := export.ExportData(r.Context(), profile, h.store, w); err != nil {
			logger.FromContext(r.Context()).Error().Err(err).Msg("Administrative export interrupted")
		}
		return
	}
	if r.Method != "POST" {
		textError(w, "item not found", http.StatusNotFound)
		return
	}
	identity, err := adminweb.SessionIdentity(r, h.store)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.action(w, r, identity)
}

// render buffers the complete template so errors do not leak secrets or partial HTML.
func (h *Handler) render(w http.ResponseWriter, status int, data view) {
	data.Locale = i18n.ViewOf(w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var out strings.Builder
	data.Error = localMessage(data.Locale, data.Error)
	tmpl, err := h.adminTemplate(data.Locale)
	if err == nil {
		err = tmpl.ExecuteTemplate(&out, "admin.html", data)
	}
	if err != nil {
		textError(w, "operation failed", 500)
		return
	}
	w.WriteHeader(status)
	_, _ = io.WriteString(w, out.String())
}

// fail returns generic public errors and never prints request bodies or credentials.
func (h *Handler) fail(w http.ResponseWriter, err error) {
	status := adminweb.ErrorStatus(err)
	if errors.Is(err, relay.ErrUpstream) {
		status = 502
	}
	message := adminweb.PublicError(err)
	if status == 502 {
		message = "relay validation unavailable"
	}
	h.render(w, status, view{Error: message})
}

// loginPage binds a new or existing preauthentication nonce without revealing its cookie.
func (h *Handler) loginPage(w http.ResponseWriter, r *http.Request, message string) {
	token, err := h.csrf.LoginToken(w, r)
	if err != nil {
		h.fail(w, err)
		return
	}
	query := r.URL.Query()
	next := adminweb.SafeReturn(query.Get("next"))
	if next == "" && query.Get("next") == "" {
		next = adminweb.SafeReturn(r.URL.RequestURI())
	}
	if next == "" {
		next = "/admin"
	}
	h.render(w, 200, view{Login: true, CSRF: token, Next: next, Error: message})
}

// postLogin validates preauthentication CSRF before checking credentials.
func (h *Handler) postLogin(w http.ResponseWriter, r *http.Request) {
	f, err := h.csrf.ParseLoginForm(w, r, adminweb.FormLimits{MaxBytes: 8192, TempDir: h.tempDir})
	if err != nil {
		h.fail(w, err)
		return
	}
	defer f.Close()
	if !fields(f, "username", "password", "remember", "next") {
		h.fail(w, db.ErrInvalidInput)
		return
	}
	remember := f.SingleValue("remember")
	if remember != "" && remember != "1" {
		h.fail(w, db.ErrInvalidInput)
		return
	}
	session, err := admin.Login(r.Context(), h.store, f.SingleValue("username"), f.SingleValue("password"), remember == "1")
	if err != nil {
		if errors.Is(err, admin.ErrUnauthorized) {
			token, tokenErr := h.csrf.LoginToken(w, r)
			if tokenErr != nil {
				h.fail(w, tokenErr)
				return
			}
			next := adminweb.SafeReturn(f.SingleValue("next"))
			if next == "" {
				next = "/admin"
			}
			h.render(w, http.StatusUnauthorized, view{Login: true, CSRF: token, Next: next, Error: "Invalid username or password."})
			return
		}
		h.fail(w, err)
		return
	}
	adminweb.SetSessionCookie(w, r, session)
	adminweb.ClearPreauth(w, r)
	next := adminweb.SafeReturn(f.SingleValue("next"))
	if next == "" {
		next = "/admin"
	}
	http.Redirect(w, r, next, http.StatusSeeOther)
}

// page renders only data belonging to the authenticated admin and explicitly selected profile.
func (h *Handler) page(w http.ResponseWriter, r *http.Request, id admin.Identity) {
	q, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		h.fail(w, db.ErrInvalidInput)
		return
	}
	for k, v := range q {
		if len(v) != 1 || (k != "profile" && k != "section" && k != "next") {
			h.fail(w, db.ErrInvalidInput)
			return
		}
	}
	token, err := h.csrf.SessionToken(id)
	if err != nil {
		h.fail(w, err)
		return
	}
	profiles, err := h.store.ListListeningProfiles(r.Context())
	if err != nil {
		h.fail(w, err)
		return
	}
	section := q.Get("section")
	if section == "" {
		section = "profiles"
	}
	switch section {
	case "account", "profiles", "keys", "relay", "import", "export":
	default:
		h.fail(w, db.ErrInvalidInput)
		return
	}
	ticket, err := h.newSubmission(id.SessionID)
	if err != nil {
		h.fail(w, err)
		return
	}
	v := view{User: id.User, Profiles: profiles, CSRF: token, Section: section, Submission: ticket}
	if q.Get("profile") != "" {
		n, e := parseID(q.Get("profile"))
		if e != nil {
			h.fail(w, e)
			return
		}
		v.Profile, err = h.store.GetListeningProfileByID(r.Context(), n)
		if err != nil {
			h.fail(w, err)
			return
		}
		v.Keys, err = admin.Keys(r.Context(), h.store, id.User, n, v.Profile.Slug)
		if err != nil {
			h.fail(w, err)
			return
		}
		v.RelayURL, v.RelayToken, err = admin.Relay(r.Context(), h.store, id.User, n, v.Profile.Slug)
		if err != nil {
			h.fail(w, err)
			return
		}
		v.Notice = h.takeNotice(id.SessionID, n, i18n.ViewOf(w))
	} else {
		v.Notice = h.takeNotice(id.SessionID, 0, i18n.ViewOf(w))
	}
	h.render(w, 200, v)
}

// parseID rejects negative, zero, and overflowing profile or key IDs.
func parseID(raw string) (int32, error) {
	n, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || n <= 0 {
		return 0, db.ErrInvalidInput
	}
	return int32(n), nil
}

// fields enforces an action-specific allowlist for every parsed field and file.
func fields(f *adminweb.PreparedForm, names ...string) bool {
	allowed := map[string]bool{"csrf_token": true, "submission_token": true}
	for _, name := range names {
		allowed[name] = true
	}
	for key := range f.Values {
		if !allowed[key] {
			return false
		}
	}
	for key := range f.Files {
		if !allowed[key] || (key != "image" && key != "file") {
			return false
		}
	}
	return true
}

// profilePath is generated only from a positive, server-resolved ID.
func profilePath(id int32) string { return "/admin?profile=" + strconv.FormatInt(int64(id), 10) }

// putNotice stores only fixed, non-secret feedback bound to the authenticated session.
func (h *Handler) putNotice(id uuid.UUID, profile int32, text string) {
	h.putNoticeData(id, profile, messageID(text), text, nil)
}

// putNoticeData keeps message identity and arguments until the redirected request.
func (h *Handler) putNoticeData(id uuid.UUID, profile int32, key, text string, values map[string]any) {
	h.mu.Lock()
	defer h.mu.Unlock()
	now := time.Now()
	for key, n := range h.notices {
		if n.expires.Before(now) {
			delete(h.notices, key)
		}
	}
	if len(h.notices) >= 1024 {
		for key := range h.notices {
			delete(h.notices, key)
			break
		}
	}
	h.notices[id] = notice{profile: profile, key: key, text: text, values: values, expires: now.Add(time.Minute)}
}

// takeNotice consumes a short-lived message only for its original session and profile.
func (h *Handler) takeNotice(id uuid.UUID, profile int32, locale *i18n.View) string {
	h.mu.Lock()
	defer h.mu.Unlock()
	n, ok := h.notices[id]
	if !ok || n.profile != profile || time.Now().After(n.expires) {
		return ""
	}
	delete(h.notices, id)
	return adminFormat(locale, n.key, n.text, n.values)
}

// openImage opens only the server-generated staging path returned by the verified parser.
func openImage(f *adminweb.PreparedForm) (*os.File, error) {
	u, ok := f.Files["image"]
	if !ok {
		return nil, nil
	}
	return os.Open(u.Path)
}
