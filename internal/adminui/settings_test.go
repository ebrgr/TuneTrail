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
	store, err := sqlite.NewInMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close(context.Background())
	snap := cfg.SettingsSnapshot{Revision: strings.Repeat("c", 64), Saved: map[string]string{}, Effective: map[string]string{}, Active: map[string]string{}, Origins: origins, SavedOrigins: map[string]cfg.Origin{}}
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
