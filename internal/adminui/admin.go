// Package adminui serves protected administration documents and form actions.
package adminui

import (
	"context"
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
	"github.com/ebrgr/TuneTrail/internal/i18n"
	"github.com/ebrgr/TuneTrail/internal/images"
	"github.com/ebrgr/TuneTrail/internal/importer"
	"github.com/ebrgr/TuneTrail/internal/logger"
	"github.com/ebrgr/TuneTrail/internal/mbz"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/ebrgr/TuneTrail/internal/relay"
	"github.com/ebrgr/TuneTrail/internal/version"
	"github.com/ebrgr/TuneTrail/internal/webassets"
	"github.com/go-chi/httprate"
	"github.com/google/uuid"
)

// Administration has its own private template namespace within the process snapshot.
var templates = webassets.FS{Prefix: "admin"}

// Store contains the existing persistence contracts; no implementation-specific database is needed.
type Store interface {
	db.UserStore
	admin.ProfileAccountStore
	db.ProfileStore
	db.ImageStore
	db.ExportStore
	db.IngestHealthStore
	importer.UploadStore
}

// SettingsStore provides configuration without coupling it to the SQLite store.
type SettingsStore interface {
	Read(context.Context) (cfg.SettingsSnapshot, error)
	Save(context.Context, string, map[string]string) (cfg.SettingsSnapshot, error)
}
type Options struct {
	Now                    func() time.Time
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
	Personal    *db.ProfileDeletePlan
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
	now               func() time.Time
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
	Nav                                 webassets.Nav
	Locale                              *i18n.View
	User                                *models.User
	Profiles                            []models.ListeningProfile
	Profile                             *models.ListeningProfile
	Keys                                []models.ApiKey
	RelayURL, CSRF, Next, Notice, Error string
	// ISSUE-128: Warning is a neutral instruction ("Sign in to continue."); Status is the HTTP status
	// of the document; Problem selects the neutral 401/403/404 state card and is derived by render.
	Warning                             string
	Status                              int
	Problem                             bool
	Health                              *db.IngestHealth
	Discard                             db.IngestDiscardPreview
	DiscardTarget                       string
	Inbox                               []db.IngestInboxGroup
	InboxKeys                           []db.IngestInboxGroup
	InboxMetadataIssues                 int64
	InboxDays                           int
	InboxKey                            string
	RelayState                          *relay.ProfileView
	Login, SignOut                      bool
	Section                             string
	Submission                          string
	AccountsRoot, CanIssueToken, Redeem bool
	Account                             db.AccountDetails
	IssuedToken                         string
	AccountPage                         *accountState
	RedeemUsername, RedeemField         string
	RedeemSession                       string
	// ISSUE-126: shared account shell. Tab is the current tab, Context is true only while a
	// root operates another account, and Rows feeds the root accounts list.
	Tab     string
	Tabs    []webassets.NavLink
	Context bool
	Rows    []accountRow
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
	asset, err := webassets.AssetFunc("/web-assets")
	if err != nil {
		return nil, err
	}
	tmpl, err := template.New("admin").Funcs(adminFuncs(nil)).Funcs(template.FuncMap{"msg": newMessage, "heading": newHeading, "asset": asset, "head": webassets.HeadFunc, "defaultTheme": func() string { return opts.DefaultTheme }, "initials": webassets.Initials, "strval": func(p *string) string {
		if p == nil {
			return ""
		}
		return *p
	}, "avatarIcon": webassets.AvatarIcon, "icon": webassets.Icon, "themes": webassets.ThemeOptions}).ParseFS(templates, "templates/*.html")
	if err == nil {
		err = webassets.ValidateTemplates(tmpl)
	}
	if err != nil {
		return nil, err
	}
	if opts.SubsonicTest == nil {
		opts.SubsonicTest = images.TestSubsonic
	}
	// Use the same clock for the Inbox bounds throughout a request.
	if opts.Now == nil {
		opts.Now = time.Now
	}
	h := &Handler{now: opts.Now, languages: opts.Languages, settings: opts.Settings, subsonicTest: opts.SubsonicTest, requireSubmission: opts.RequireSubmissionToken, submissions: map[uuid.UUID]submissionTicket{}, store: store, csrf: csrf, tempDir: opts.TempDir, relayClient: opts.RelayClient, musicBrainz: opts.MusicBrainz, tmpl: tmpl, notices: map[uuid.UUID]notice{}, catalogPlans: map[uuid.UUID]pendingCatalogPlan{}}
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
	h.entry = adminweb.NoStore(adminweb.SameOriginWithReject(http.HandlerFunc(h.dispatch), h.rejectOrigin))
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
	// Protect the redemption document and errors produced before dispatch.
	if r.URL.Path == "/admin/redeem" {
		// ISSUE-125: not "no-referrer". Browsers serialize the Origin of a form POST as "null" under
		// that policy, so the same-origin check rejected every real redemption. The page and its URL
		// carry no secret (a query string is refused), and same-origin still sends no Referer elsewhere.
		w.Header().Set("Referrer-Policy", "same-origin")
	}
	w.Header().Add("Vary", "Cookie")
	w = requestWriter{h.languages.Wrap(w, r), r}
	if r.Method == http.MethodHead {
		// Confirmation HEAD requests must not allocate a submission ticket.
		if r.URL.Path == "/admin/sign-out" {
			h.entry.ServeHTTP(headWriter{w}, r)
			return
		}
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
	// Redemption uses preauthentication CSRF and only hashes a password after claiming a valid token.
	if r.URL.Path == "/admin/redeem" {
		h.redeemAccount(w, r)
		return
	}
	if r.URL.Path == "/admin/sign-out" {
		h.signOutPage(w, r)
		return
	}
	if r.URL.Path == webassets.LoginDestination {
		h.loginRoute(w, r)
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
	// Health has its own session and profile authorization before diagnostics.
	if r.URL.Path == "/admin/health" || isHealthAction(r.URL.Path) {
		h.healthPage(w, r)
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
		if err := admin.AuthorizeAction(identity.User, admin.ManageServer, nil); err != nil {
			h.failForSession(w, r, identity, err)
			return
		}
		h.settingsPage(w, r, identity)
		return
	}
	if r.URL.Path == "/account" {
		h.accountPage(w, r)
		return
	}
	// ISSUE-126: the tabs of the session's own profile; IsRoute already restricted the path.
	if strings.HasPrefix(r.URL.Path, "/account/") {
		h.accountTab(w, r)
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
		// A stale redemption redirect for an already signed-in browser lands on the account.
		if r.URL.Query().Has("redeemed") {
			http.Redirect(w, r, webassets.AccountDestination, http.StatusSeeOther)
			return
		}
		if err := h.authorizePage(r, identity); err != nil {
			h.failForSession(w, r, identity, err)
			return
		}
		// ISSUE-125: the account document moved to /account; old links keep working.
		if r.URL.Query().Get("section") == "account" && r.URL.Query().Get("profile") == "" {
			http.Redirect(w, r, webassets.AccountDestination, http.StatusSeeOther)
			return
		}
		// ISSUE-126: the own profile has a single address, /account/<tab>.
		if h.redirectOwnProfile(w, r, identity) {
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
		// Use the listening policy to hide protected exports before management checks.
		if admin.AuthorizeAction(identity.User, admin.ReadListening, profile) != nil {
			h.fail(w, db.ErrNotFound)
			return
		}
		if _, err = admin.Profile(r.Context(), h.store, identity.User, id, profile.Slug); err != nil {
			h.failForSession(w, r, identity, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", `attachment; filename="tunetrail_export.json"`)
		if err := admin.ExportHistory(r.Context(), h.store, identity.User, profile.ID, w); err != nil {
			logger.FromContext(r.Context()).Error().Err(err).Msg("Administrative export interrupted")
		}
		return
	}
	if r.Method != "POST" {
		textError(w, "item not found", http.StatusNotFound)
		return
	}
	// Resolve the session independently of role, then authorize the listed operation.
	identity, err := adminweb.SessionIdentity(r, h.store)
	if err == nil {
		err = h.authorizeAction(r, identity)
	}
	if r.URL.Path == "/admin/logout" && errors.Is(err, admin.ErrUnauthorized) {
		// An expired or revoked session gets a neutral sign-in form instead of a 403.
		h.renderLogin(w, r, http.StatusUnauthorized, "Sign in to continue.")
		return
	}
	if err != nil {
		h.failForSession(w, r, identity, err)
		return
	}

	h.action(w, r, identity)
}

// failForSession renders a forbidden document without loading its denied resource.
func (h *Handler) failForSession(w http.ResponseWriter, r *http.Request, identity admin.Identity, err error) {
	if identity.User == nil || adminweb.ErrorStatus(err) != http.StatusForbidden {
		h.fail(w, err)
		return
	}
	nav := h.navigation(r, identity, nil, "")
	h.render(w, http.StatusForbidden, view{Nav: nav, Error: "Your account does not have permission to access this page."})
}

// render buffers the complete template so errors do not leak secrets or partial HTML.
func (h *Handler) render(w http.ResponseWriter, status int, data view) {
	data.Locale = i18n.ViewOf(w)
	if data.Locale != nil && (status == http.StatusForbidden || status == http.StatusNotFound) {
		data.Locale.Return = "/"
	}
	data.Status = status
	if data.Nav.Kind == "" {
		section := ""
		if data.Login {
			section = "login"
		}
		data.Nav = webassets.Nav{Kind: "admin", Version: version.Current, NavAccess: webassets.AccessNavigation(nil, nil, nil, section)}
		// ISSUE-128: an error document offers the entries of the session that asked, so a signed-in
		// user is not told to sign in. The links never depend on why the request was refused.
		if req := requestOf(w); req != nil && data.Error != "" && !data.Login {
			if identity, err := adminweb.SessionIdentity(req, h.store); err == nil && identity.User != nil {
				data.Nav = h.navigation(req, identity, nil, "")
			}
		}
	}
	// The state card replaces the page body for refusals; forms and tabs keep their own content.
	data.Problem = data.Error != "" && !data.Login && !data.Redeem && !data.SignOut && data.Health == nil && data.User == nil
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// "Sign in to continue." is an instruction, not a failure: same role as the sign-in form's warning.
	if data.Problem && status == http.StatusUnauthorized && data.Warning == "" {
		data.Warning, data.Error = data.Error, ""
	}
	var out strings.Builder
	data.Error = localMessage(data.Locale, data.Error)
	data.Warning = localMessage(data.Locale, data.Warning)
	tmpl, err := h.adminTemplate(data.Locale)
	if err == nil {
		err = tmpl.ExecuteTemplate(&out, "admin/home", data)
	}
	if err != nil {
		textError(w, "operation failed", 500)
		return
	}
	w.WriteHeader(status)
	_, _ = io.WriteString(w, out.String())
}

// errorMessage maps an error to its public status and fixed message.
func errorMessage(err error) (int, string) {
	status := adminweb.ErrorStatus(err)
	if errors.Is(err, relay.ErrUpstream) {
		status = 502
	}
	message := adminweb.PublicError(err)
	switch status {
	case http.StatusUnauthorized:
		message = "Sign in to continue."
	case http.StatusForbidden:
		if message == "request forbidden" {
			message = "Your account does not have permission to access this page."
		}
	case http.StatusNotFound:
		message = "This profile or page is unavailable."
	}
	if status == 502 {
		message = "relay validation unavailable"
	}
	return status, message
}

// fail returns generic public errors and never prints request bodies or credentials.
func (h *Handler) fail(w http.ResponseWriter, err error) {
	status, message := errorMessage(err)
	h.render(w, status, view{Error: message})
}

// loginPage answers a protected document requested without a session with the neutral sign-in form
// and the "Sign in to continue." instruction (ISSUE-128). The status stays 200 as before; only the
// sign-out and logout paths answer 401. The nonce is bound without revealing its cookie.
func (h *Handler) loginPage(w http.ResponseWriter, r *http.Request, message string) {
	if message == "" {
		message = msgSignIn
	}
	h.renderLogin(w, r, http.StatusOK, message)
}

// loginRoute serves GET /login for every role (ISSUE-128). A session that is already open goes to
// the validated return or to its role's home instead of seeing the form; an expired session cookie
// gets the neutral instruction, a plain visit does not.
func (h *Handler) loginRoute(w http.ResponseWriter, r *http.Request) {
	identity, err := adminweb.SessionIdentity(r, h.store)
	switch {
	case err == nil && identity.User != nil:
		http.Redirect(w, r, h.loginDestination(r, identity.User, r.URL.Query().Get("next")), http.StatusSeeOther)
	case errors.Is(err, admin.ErrUnauthorized):
		message := ""
		if _, err := r.Cookie(adminweb.SessionCookie); err == nil {
			// A cookie that no longer resolves is an expired or revoked session.
			message = msgSignIn
		}
		h.renderLogin(w, r, http.StatusOK, message)
	default:
		h.fail(w, err)
	}
}

// renderLogin gives an absent or expired session a usable sign-in form. A non-empty message is a
// neutral warning, never an account-specific diagnostic.
func (h *Handler) renderLogin(w http.ResponseWriter, r *http.Request, status int, message string) {
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
	v := view{Login: true, CSRF: token, Next: next, Warning: message}
	// ISSUE-125: the redirect after a successful token redemption asks for a normal sign-in.
	// ISSUE-128: a password change ends every session and lands here with its own confirmation.
	switch {
	case query.Get("redeemed") == "1":
		v.Notice = adminText(i18n.ViewOf(w), "admin.account.redeemed", "Password set. Sign in with your username and the new password.")
	case query.Get("changed") == "1":
		v.Notice = adminText(i18n.ViewOf(w), "admin.account.password_changed", "Password changed. Sign in with your new password.")
	}
	// A confirmation already tells the user to sign in; the generic instruction would repeat it.
	if v.Notice != "" {
		v.Warning = ""
	}
	h.render(w, status, v)
}

// postLogin validates preauthentication CSRF before checking credentials.
func (h *Handler) postLogin(w http.ResponseWriter, r *http.Request) {
	f, err := h.csrf.ParseLoginForm(w, r, adminweb.FormLimits{MaxBytes: 8192, TempDir: h.tempDir})
	if errors.Is(err, adminweb.ErrCSRF) {
		// ISSUE-125: an expired or foreign preauth cookie gets a fresh form, not a permission error.
		token, tokenErr := h.csrf.LoginToken(w, r)
		if tokenErr != nil {
			h.fail(w, tokenErr)
			return
		}
		h.render(w, http.StatusForbidden, view{Login: true, CSRF: token, Error: "This page expired or was opened in another tab. Try again."})
		return
	}
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
		logger.FromContext(r.Context()).Warn().Int("status", adminweb.ErrorStatus(err)).Msg("Login failed")
		if errors.Is(err, admin.ErrUnauthorized) {
			token, tokenErr := h.csrf.LoginToken(w, r)
			if tokenErr != nil {
				h.fail(w, tokenErr)
				return
			}
			next := adminweb.SafeReturn(f.SingleValue("next"))
			h.render(w, http.StatusUnauthorized, view{Login: true, CSRF: token, Next: next, Error: "Invalid username or password."})
			return
		}
		h.fail(w, err)
		return
	}
	// Resolve the committed session so fallback and return authorization use its current role.
	identity, err := admin.Authenticate(r.Context(), h.store, session.ID)
	if err != nil {
		h.fail(w, err)
		return
	}
	adminweb.SetSessionCookie(w, r, session)
	adminweb.ClearPreauth(w, r)
	http.Redirect(w, r, h.loginDestination(r, identity.User, f.SingleValue("next")), http.StatusSeeOther)
}

// redirectOwnProfile sends /admin?profile=<own> and its legacy sections to /account/<tab>.
// It runs after authorizePage and only acts when the selected profile belongs to the session.
func (h *Handler) redirectOwnProfile(w http.ResponseWriter, r *http.Request, id admin.Identity) bool {
	q := r.URL.Query()
	raw := q.Get("profile")
	if raw == "" {
		return false
	}
	tab, ok := webassets.TabFromSection(q.Get("section"))
	n, err := parseID(raw)
	if !ok || err != nil {
		return false
	}
	profile, err := h.store.GetListeningProfileByID(r.Context(), n)
	if err != nil || profile.UserID != id.User.ID {
		return false
	}
	http.Redirect(w, r, webassets.AccountTabURL(true, profile.ID, tab), http.StatusSeeOther)
	return true
}

// page serves the root accounts list, or a tab of another account in the administration context.
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
	section := q.Get("section")
	if section == "" {
		section = "profiles"
	}
	switch section {
	case "profiles", "keys", "relay", "import", "export", "account":
	default:
		h.fail(w, db.ErrInvalidInput)
		return
	}
	if q.Get("profile") != "" {
		n, e := parseID(q.Get("profile"))
		if e != nil {
			h.fail(w, e)
			return
		}
		profile, e := h.store.GetListeningProfileByID(r.Context(), n)
		if e != nil {
			h.fail(w, e)
			return
		}
		tab, _ := webassets.TabFromSection(section)
		// A profile of the session itself never uses the context mode (redirected before).
		h.renderProfileTab(w, r, id, profile, tab, profile.UserID == id.User.ID, accountState{}, http.StatusOK, "")
		return
	}
	if section == "account" {
		h.fail(w, db.ErrInvalidInput)
		return
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
	// Limit the management document to resources authorized by the same policy.
	allowed := profiles[:0]
	for _, p := range profiles {
		if admin.AuthorizeAction(id.User, admin.ManageProfile, &p) == nil {
			allowed = append(allowed, p)
		}
	}
	profiles = allowed
	ticket, err := h.newSubmission(id.SessionID)
	if err != nil {
		h.fail(w, err)
		return
	}
	v := view{User: id.User, Profiles: profiles, CSRF: token, Section: section, Submission: ticket,
		AccountsRoot: admin.AuthorizeAction(id.User, admin.ManageAccounts, nil) == nil}
	// Account roles and states are loaded only for administrators.
	if v.AccountsRoot {
		v.Rows = h.accountRows(r.Context(), id.User, profiles)
	}
	v.Notice = h.takeNotice(id.SessionID, 0, i18n.ViewOf(w))
	v.Nav = h.navigation(r, id, nil, "profiles")
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

// rejectOrigin answers a request the same-origin policy refused. The pre-session redemption form
// gets a fresh form and the neutral "page expired" text instead of a misleading permission error
// (ISSUE-125); every other path, including sign-in, keeps the generic forbidden answer.
func (h *Handler) rejectOrigin(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/admin/redeem" {
		h.renderRedeem(w, r, http.StatusForbidden, view{Redeem: true}, "This page expired or was opened in another tab. Try again.", "")
		return
	}
	h.fail(w, adminweb.ErrCSRF)
}
