package adminui

import (
	"context"
	"errors"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/adminweb"
	"github.com/ebrgr/TuneTrail/internal/cfg"
	"github.com/ebrgr/TuneTrail/internal/db/sqlite"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/google/uuid"
)

type uncertainSettings struct{ state cfg.SettingsSnapshot }

func (s uncertainSettings) Read(context.Context) (cfg.SettingsSnapshot, error) { return s.state, nil }
func (s uncertainSettings) Save(context.Context, string, map[string]string) (cfg.SettingsSnapshot, error) {
	return s.state, &cfg.DurabilityError{Cause: errors.New("disk sync")}
}

// TestSettingsDurabilityWarning confirms a published file is not reported as an unsaved draft.
func TestSettingsDurabilityWarning(t *testing.T) {
	store, err := sqlite.NewInMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close(context.Background())
	revision := strings.Repeat("b", 64)
	snap := cfg.SettingsSnapshot{Revision: revision, Saved: map[string]string{cfg.LASTFM_API_KEY_ENV: "published-value"}, Effective: map[string]string{}, Active: map[string]string{}, Origins: map[string]cfg.Origin{}, SavedOrigins: map[string]cfg.Origin{}}
	h, err := New(store, Options{Settings: uncertainSettings{state: snap}, DisableRateLimit: true})
	if err != nil {
		t.Fatal(err)
	}
	identity := admin.Identity{User: &models.User{ID: 1, Username: "admin", Role: models.UserRoleAdmin}, SessionID: uuid.New()}
	values := url.Values{"revision": {strings.Repeat("a", 64)}}
	for _, spec := range settingsFields {
		values.Set(spec.Name, "")
	}
	f := &adminweb.PreparedForm{Values: values, Files: map[string]adminweb.Upload{}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/admin/settings/save", nil)
	h.settingsAction(w, r, identity, f, "settings/save")
	if w.Code != 500 || !strings.Contains(w.Body.String(), "configuration was saved") || !strings.Contains(w.Body.String(), `name="revision" value="`+revision+`"`) || !strings.Contains(w.Body.String(), `value="published-value"`) {
		t.Fatalf("unexpected durability response: %d", w.Code)
	}
}

type unavailableSettings struct{}

func (unavailableSettings) Read(context.Context) (cfg.SettingsSnapshot, error) {
	return cfg.SettingsSnapshot{}, errors.New("unavailable")
}
func (unavailableSettings) Save(context.Context, string, map[string]string) (cfg.SettingsSnapshot, error) {
	return cfg.SettingsSnapshot{}, cfg.ErrInvalidSettings
}

// TestSettingsPreservesInvalidDraftWhenStateUnavailable avoids inventing current values or changing select inputs.
func TestSettingsPreservesInvalidDraftWhenStateUnavailable(t *testing.T) {
	store, err := sqlite.NewInMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close(context.Background())
	h, err := New(store, Options{Settings: unavailableSettings{}, DisableRateLimit: true})
	if err != nil {
		t.Fatal(err)
	}
	identity := admin.Identity{User: &models.User{ID: 1, Username: "admin", Role: models.UserRoleAdmin}, SessionID: uuid.New()}
	values := url.Values{"revision": {strings.Repeat("a", 64)}}
	for _, spec := range settingsFields {
		values.Set(spec.Name, "")
	}
	values.Set(cfg.LOG_LEVEL_ENV, "verbose")
	values.Set(cfg.DISABLE_DEEZER_ENV, "maybe")
	f := &adminweb.PreparedForm{Values: values, Files: map[string]adminweb.Upload{}}
	w := httptest.NewRecorder()
	h.settingsAction(w, httptest.NewRequest("POST", "/admin/settings/save", nil), identity, f, "settings/save")
	body := w.Body.String()
	if w.Code != 503 || !strings.Contains(body, `<option value="verbose" selected>Invalid: verbose</option>`) || !strings.Contains(body, `<option value="maybe" selected>Invalid: maybe</option>`) || strings.Contains(body, "server-setting-state") || !strings.Contains(body, "temporarily unavailable") {
		t.Fatalf("invalid draft or unavailable state not preserved: %d", w.Code)
	}
}

// subsonicPage renders the settings page for the credentials dialog tests (ISSUE-127).
func subsonicPage(t *testing.T, origins map[string]cfg.Origin) string {
	t.Helper()
	return settingsPage(t, cfg.SettingsSnapshot{Revision: strings.Repeat("c", 64), Saved: map[string]string{}, Effective: map[string]string{}, Active: map[string]string{}, Origins: origins, SavedOrigins: map[string]cfg.Origin{}})
}

// settingsPage renders the server configuration for a given snapshot (ISSUE-127).
func settingsPage(t *testing.T, snap cfg.SettingsSnapshot) string {
	t.Helper()
	store, err := sqlite.NewInMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close(context.Background())
	h, err := New(store, Options{Settings: uncertainSettings{state: snap}, DisableRateLimit: true})
	if err != nil {
		t.Fatal(err)
	}
	identity := admin.Identity{User: &models.User{ID: 1, Username: "admin", Role: models.UserRoleAdmin}, SessionID: uuid.New()}
	w := httptest.NewRecorder()
	h.renderSettings(httptest.NewRequest("GET", "/admin/settings", nil), w, identity, snap, nil, snap.Revision, "", "", 200, nil)
	return w.Body.String()
}

// TestSubsonicCredentialsDialog checks that the password field can never be part of a submission.
func TestSubsonicCredentialsDialog(t *testing.T) {
	body := subsonicPage(t, map[string]cfg.Origin{})
	for _, want := range []string{"data-subsonic-open", `id="subsonic-credentials"`, "never saved or sent to TuneTrail", "subsonic-credentials.js"} {
		if !strings.Contains(body, want) {
			t.Fatalf("settings page lacks %q", want)
		}
	}
	// The dialog sits after the page form closes, so a submission cannot carry its inputs.
	formStart := strings.Index(body, `class="server-settings-form"`)
	formEnd := formStart + strings.Index(body[formStart:], "</form>")
	dialog := strings.Index(body, `id="subsonic-credentials"`)
	if formStart < 0 || dialog < formEnd {
		t.Fatal("the credentials dialog must be outside the settings form")
	}
	section := body[dialog : dialog+strings.Index(body[dialog:], "</dialog>")]
	if strings.Contains(section, "<form") || strings.Contains(section, " name=") {
		t.Fatal("no field of the credentials dialog may be named or inside a form")
	}
	if !strings.Contains(section, `type="password" autocomplete="new-password"`) {
		t.Fatal("the password input must be a password field")
	}
}

// TestSubsonicCredentialsButtonHiddenWhenLocked keeps the generator away from an environment-controlled field.
func TestSubsonicCredentialsButtonHiddenWhenLocked(t *testing.T) {
	body := subsonicPage(t, map[string]cfg.Origin{cfg.SUBSONIC_PARAMS_ENV: cfg.OriginEnvironment})
	if strings.Contains(body, "data-subsonic-open") {
		t.Fatal("the generate button must not appear when the parameters come from the environment")
	}
}

// cardOf returns the markup of one settings card, from its opening tag to the next card or the action bar.
func cardOf(t *testing.T, body, id string) string {
	t.Helper()
	start := strings.Index(body, `id="settings-`+id+`" class="settings-card"`)
	if start < 0 {
		t.Fatalf("card %q is missing", id)
	}
	end := len(body)
	for _, stop := range []string{`<section id="settings-`, `class="settings-actions`} {
		if i := strings.Index(body[start+10:], stop); i >= 0 && start+10+i < end {
			end = start + 10 + i
		}
	}
	return body[start:end]
}

// TestSettingsCardsGroupFields checks the six cards, their order and the field that belongs to each (ISSUE-127).
func TestSettingsCardsGroupFields(t *testing.T) {
	body := settingsPage(t, cfg.SettingsSnapshot{Revision: strings.Repeat("d", 64), Saved: map[string]string{}, Effective: map[string]string{}, Active: map[string]string{}, Origins: map[string]cfg.Origin{}, SavedOrigins: map[string]cfg.Origin{}})
	order := []string{"general", "time", "musicbrainz", "images", "subsonic", "network"}
	last := -1
	for _, id := range order {
		at := strings.Index(body, `id="settings-`+id+`" class="settings-card"`)
		if at <= last {
			t.Fatalf("card %q is missing or out of order", id)
		}
		last = at
	}
	want := map[string][]string{
		"general": {cfg.LOG_LEVEL_ENV}, "time": {cfg.FORCE_TZ}, "musicbrainz": {cfg.MUSICBRAINZ_URL_ENV, cfg.MUSICBRAINZ_RATE_LIMIT_ENV, cfg.DISABLE_MUSICBRAINZ_ENV},
		"images": {cfg.DISABLE_DEEZER_ENV, cfg.DISABLE_COVER_ART_ARCHIVE_ENV, cfg.LASTFM_API_KEY_ENV}, "subsonic": {cfg.SUBSONIC_URL_ENV, cfg.SUBSONIC_PARAMS_ENV},
		"network": {cfg.RelayAllowHTTPKey, cfg.RelayAllowPrivateKey},
	}
	for id, names := range want {
		card := cardOf(t, body, id)
		for _, name := range names {
			if !strings.Contains(card, `name="`+name+`"`) {
				t.Errorf("card %q lacks %s", id, name)
			}
		}
		if got := strings.Count(card, `<section class="server-setting"`); got != len(names) {
			t.Errorf("card %q has %d fields, want %d", id, got, len(names))
		}
	}
	// The save and test actions live in one bar after every card, inside the form.
	if bar := strings.Index(body, `class="settings-actions`); bar < last || !strings.Contains(body[bar:], "/admin/settings/subsonic/test") {
		t.Fatal("the action bar must follow the cards and carry Save and Test")
	}
}

// TestSettingsCardRestartNotice marks only the card whose effective value differs from the running one.
func TestSettingsCardRestartNotice(t *testing.T) {
	snap := cfg.SettingsSnapshot{
		Revision: strings.Repeat("e", 64),
		Saved:    map[string]string{cfg.LOG_LEVEL_ENV: "debug"}, Effective: map[string]string{cfg.LOG_LEVEL_ENV: "debug"}, Active: map[string]string{cfg.LOG_LEVEL_ENV: "info"},
		Origins: map[string]cfg.Origin{cfg.LOG_LEVEL_ENV: cfg.OriginTOML}, SavedOrigins: map[string]cfg.Origin{cfg.LOG_LEVEL_ENV: cfg.OriginTOML}, RestartRequired: true,
	}
	body := settingsPage(t, snap)
	general, mb := cardOf(t, body, "general"), cardOf(t, body, "musicbrainz")
	if !strings.Contains(general, `id="settings-restart-general"`) || !strings.Contains(general, "origin-restart") {
		t.Fatal("the general card must announce the restart")
	}
	if strings.Contains(mb, "settings-restart-") || strings.Contains(mb, "origin-restart") {
		t.Fatal("a card without pending changes must not announce a restart")
	}
	if !strings.Contains(general, `<span class="origin-tag">Configuration file</span>`) {
		t.Fatal("the origin of the effective value must show on the field line")
	}
}

// TestSettingsLockedFieldIsMarked shows the lock beside an environment-controlled field and keeps it disabled.
func TestSettingsLockedFieldIsMarked(t *testing.T) {
	body := subsonicPage(t, map[string]cfg.Origin{cfg.LOG_LEVEL_ENV: cfg.OriginEnvironment})
	general := cardOf(t, body, "general")
	if !strings.Contains(general, "server-setting-override") || !strings.Contains(general, "origin-locked") || !strings.Contains(general, "disabled") {
		t.Fatal("an environment-controlled field must be locked, tagged and disabled")
	}
	if strings.Contains(cardOf(t, body, "musicbrainz"), "origin-locked") {
		t.Fatal("only the environment-controlled field is locked")
	}
}

// TestSettingsPageMatchesManagedSettings checks that the page shows every managed setting once and nothing else,
// and that each one has a conf.toml key (ISSUE-127).
func TestSettingsPageMatchesManagedSettings(t *testing.T) {
	managed := cfg.ManagedSettingNames()
	if len(settingsFields) != len(managed) {
		t.Fatalf("the page has %d fields, the configuration manages %d", len(settingsFields), len(managed))
	}
	onPage := map[string]bool{}
	for _, spec := range settingsFields {
		if onPage[spec.Name] {
			t.Errorf("%s appears twice", spec.Name)
		}
		onPage[spec.Name] = true
		if _, ok := cfg.TOMLKey(spec.Name); !ok {
			t.Errorf("%s has no conf.toml key", spec.Name)
		}
	}
	for _, name := range managed {
		if !onPage[name] {
			t.Errorf("%s is managed but not on the page", name)
		}
	}
}

// TestSettingsButtonsUseCommonStates checks that every button of the page uses the shared hover and pressed
// states, and that the removed legacy and shadow notice is gone (ISSUE-127).
func TestSettingsButtonsUseCommonStates(t *testing.T) {
	body := subsonicPage(t, map[string]cfg.Origin{})
	for _, want := range []string{
		`class="account-action">Save configuration`, `class="account-secondary" formaction="/admin/settings/subsonic/test"`,
		`class="account-secondary" hidden data-subsonic-open`, `class="account-action" data-subsonic-generate`,
		`class="account-secondary" data-subsonic-test`, `class="account-secondary" data-subsonic-cancel`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("button markup %q is missing", want)
		}
	}
	if strings.Contains(body, "Legacy and shadow") || strings.Contains(body, "legacy_warning") {
		t.Fatal("the legacy and shadow forwarding notice must not be shown")
	}
}
