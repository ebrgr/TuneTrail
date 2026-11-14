package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/cfg"
)

// TestSettingsHTTPJourney verifies real login, CSRF, persistence, conflicts and a non-saving Subsonic test.
func TestSettingsHTTPJourney(t *testing.T) {
	if os.Getenv("TUNETRAIL_TEST_041_SERVER") == "1" {
		os.Args = []string{"tunetrail"}
		main()
		return
	}
	dir := t.TempDir()
	config := `schema_version=1
# preserved by the settings form
[logging]
level='info' # keep original quotes
[musicbrainz]
disabled=true
[images]
disable_deezer=true
disable_cover_art_archive=true
`
	conf := filepath.Join(dir, "conf.toml")
	if err := os.WriteFile(conf, []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	envFile := filepath.Join(dir, "deezer-override")
	if err := os.WriteFile(envFile, []byte("true\n"), 0600); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	env := []string{}
	// Drop every configuration variable in both prefixes (ISSUE-111): a developer's
	// .env exported by the Makefile must never reach the fixture server.
	configNames := map[string]bool{}
	for _, name := range cfg.EnvironmentNames() {
		configNames[name] = true
	}
	for _, v := range os.Environ() {
		name, _, _ := strings.Cut(v, "=")
		if !configNames[name] && !strings.HasPrefix(v, "TUNETRAIL_TEST_041_SERVER=") {
			env = append(env, v)
		}
	}
	env = append(env, "TUNETRAIL_TEST_041_SERVER=1", "KOITO_CONFIG_DIR="+dir, "KOITO_BIND_ADDR=127.0.0.1", fmt.Sprintf("KOITO_LISTEN_PORT=%d", port), "KOITO_DEFAULT_PASSWORD=fixture-only", "KOITO_SKIP_IMPORT=true", "TUNETRAIL_LOG_LEVEL=error", "KOITO_DISABLE_DEEZER_FILE="+envFile)
	cmd := exec.Command(os.Args[0], "-test.run=^TestSettingsHTTPJourney$")
	cmd.Env = env
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
	base := fmt.Sprintf("http://127.0.0.1:%d", port)
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar, Timeout: 3 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	var login string
	for until := time.Now().Add(15 * time.Second); time.Now().Before(until); time.Sleep(100 * time.Millisecond) {
		resp, e := client.Get(base + "/admin/settings")
		if e == nil {
			b, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if resp.StatusCode == 200 {
				login = string(b)
				break
			}
		}
	}
	if !strings.Contains(login, `name="next" value="/admin/settings"`) {
		t.Fatal("login did not preserve settings return")
	}
	extract := func(body, name string) string {
		t.Helper()
		m := regexp.MustCompile(`name="` + regexp.QuoteMeta(name) + `" value="([^"]*)"`).FindStringSubmatch(body)
		if len(m) != 2 {
			t.Fatalf("missing %s", name)
		}
		return m[1]
	}
	call := func(method, path string, values url.Values, origin string) (int, string, http.Header) {
		t.Helper()
		var body io.Reader
		if values != nil {
			body = strings.NewReader(values.Encode())
		}
		req, e := http.NewRequest(method, base+path, body)
		if e != nil {
			t.Fatal(e)
		}
		if values != nil {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		resp, e := client.Do(req)
		if e != nil {
			t.Fatal(e)
		}
		b, e := io.ReadAll(resp.Body)
		resp.Body.Close()
		if e != nil {
			t.Fatal(e)
		}
		return resp.StatusCode, string(b), resp.Header
	}
	// Seed a real profile key in the disposable database, and prove it works for ingestion auth.
	fixtureDB, err := sql.Open("sqlite", filepath.Join(dir, "tunetrail.db")+"?_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatal(err)
	}
	defer fixtureDB.Close()
	if _, err = fixtureDB.Exec(`INSERT INTO listening_profiles(slug,display_name,created_at) VALUES ('fixture','Fixture',123)`); err != nil {
		t.Fatal(err)
	}
	if _, err = fixtureDB.Exec(`INSERT INTO api_keys(key,profile_id,created_at,label) SELECT 'fixture-profile-key',id,123,'fixture' FROM listening_profiles WHERE slug='fixture'`); err != nil {
		t.Fatal(err)
	}
	anonymous := &http.Client{Timeout: 3 * time.Second}
	validReq, _ := http.NewRequest("GET", base+"/apis/listenbrainz/1/validate-token", nil)
	validReq.Header.Set("Authorization", "Token fixture-profile-key")
	validResp, err := anonymous.Do(validReq)
	if err != nil {
		t.Fatal(err)
	}
	validBody, _ := io.ReadAll(validResp.Body)
	validResp.Body.Close()
	if validResp.StatusCode != http.StatusOK || !strings.Contains(string(validBody), `"valid":true`) {
		t.Fatal("fixture profile key was not valid")
	}
	req, _ := http.NewRequest("GET", base+"/admin/settings", nil)
	req.Header.Set("Authorization", "Token fixture-profile-key")
	response, err := anonymous.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("API key accessed settings: %d", response.StatusCode)
	}
	for _, endpoint := range []string{"/admin/settings/save", "/admin/settings/subsonic/test"} {
		deniedReq, _ := http.NewRequest("POST", base+endpoint, strings.NewReader(""))
		deniedReq.Header.Set("Authorization", "Token fixture-profile-key")
		deniedReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		denied, e := anonymous.Do(deniedReq)
		if e != nil {
			t.Fatal(e)
		}
		denied.Body.Close()
		if denied.StatusCode != http.StatusUnauthorized {
			t.Fatalf("API key POST was accepted: %s %d", endpoint, denied.StatusCode)
		}
	}
	code, _, headers := call("POST", "/admin/login", url.Values{"csrf_token": {extract(login, "csrf_token")}, "username": {"admin"}, "password": {"fixture-only"}, "next": {"/admin/settings"}}, "")
	if code != http.StatusSeeOther || headers.Get("Location") != "/admin/settings" {
		t.Fatalf("login status %d", code)
	}
	getPage := func() string {
		t.Helper()
		code, body, headers := call("GET", "/admin/settings", nil, "")
		if code != 200 || !strings.Contains(headers.Get("Cache-Control"), "no-store") {
			t.Fatalf("settings GET status %d", code)
		}
		return body
	}
	page1 := getPage()
	page2 := getPage()
	for _, name := range []string{cfg.LOG_LEVEL_ENV, cfg.MUSICBRAINZ_URL_ENV, cfg.MUSICBRAINZ_RATE_LIMIT_ENV, cfg.DISABLE_MUSICBRAINZ_ENV, cfg.DISABLE_DEEZER_ENV, cfg.DISABLE_COVER_ART_ARCHIVE_ENV, cfg.LASTFM_API_KEY_ENV, cfg.SUBSONIC_URL_ENV, cfg.SUBSONIC_PARAMS_ENV, cfg.FORCE_TZ} {
		if !strings.Contains(page1, `name="`+name+`"`) {
			t.Fatalf("field missing: %s", name)
		}
	}
	if strings.Contains(page1, `name="KOITO_SUBSONIC_PARAMS" type="password"`) {
		t.Fatal("parameters must be visible")
	}
	values := func(page string) url.Values {
		return url.Values{"csrf_token": {extract(page, "csrf_token")}, "submission_token": {extract(page, "submission_token")}, "revision": {extract(page, "revision")}, cfg.MUSICBRAINZ_URL_ENV: {"https://musicbrainz.org"}, cfg.MUSICBRAINZ_RATE_LIMIT_ENV: {"1"}, cfg.DISABLE_MUSICBRAINZ_ENV: {"true"}, cfg.DISABLE_COVER_ART_ARCHIVE_ENV: {"true"}, cfg.LASTFM_API_KEY_ENV: {""}, cfg.SUBSONIC_URL_ENV: {""}, cfg.SUBSONIC_PARAMS_ENV: {""}, cfg.FORCE_TZ: {""}}
	}
	for _, name := range []string{cfg.LOG_LEVEL_ENV, cfg.DISABLE_DEEZER_ENV} {
		pattern := `(?s)<select[^>]*name="` + regexp.QuoteMeta(name) + `"[^>]* disabled`
		if !regexp.MustCompile(pattern).MatchString(page1) {
			t.Fatalf("override field %s is not disabled", name)
		}
	}
	forged := values(getPage())
	forged.Set(cfg.LOG_LEVEL_ENV, "debug")
	code, _, _ = call("POST", "/admin/settings/save", forged, "")
	if code != http.StatusBadRequest {
		t.Fatalf("forged override edit: %d", code)
	}
	original, err := os.ReadFile(conf)
	if err != nil {
		t.Fatal(err)
	}
	bad := values(page1)
	bad.Del("csrf_token")
	code, _, _ = call("POST", "/admin/settings/save", bad, "")
	if code != 403 {
		t.Fatalf("missing csrf: %d", code)
	}
	bad = values(page1)
	code, _, _ = call("POST", "/admin/settings/save", bad, "http://attacker.invalid")
	if code != 403 {
		t.Fatalf("external origin: %d", code)
	}
	current, err := os.ReadFile(conf)
	if err != nil || string(current) != string(original) {
		t.Fatal("unauthorized request changed settings")
	}
	first := values(page1)
	first.Set(cfg.LASTFM_API_KEY_ENV, "draft-key-one")
	code, _, headers = call("POST", "/admin/settings/save", first, "")
	if code != 303 || headers.Get("Location") != "/admin/settings" {
		t.Fatalf("save status %d", code)
	}
	saved, err := os.ReadFile(conf)
	if err != nil || !strings.Contains(string(saved), "draft-key-one") || !strings.Contains(string(saved), "# preserved by the settings form") || !strings.Contains(string(saved), "level='info' # keep original quotes") {
		t.Fatal("save lost value or comment")
	}
	second := values(page2)
	second.Set(cfg.LASTFM_API_KEY_ENV, "draft-key-two")
	code, body, _ := call("POST", "/admin/settings/save", second, "")
	if code != 409 || !strings.Contains(body, `value="draft-key-two"`) || !strings.Contains(body, `name="revision" value="`+extract(page2, "revision")+`"`) {
		t.Fatalf("conflict did not preserve draft and revision: %d", code)
	}
	refreshed := getPage()
	if !strings.Contains(refreshed, "Restart to apply") || !strings.Contains(refreshed, "Set by environment variable <code>TUNETRAIL_LOG_LEVEL</code>") || !strings.Contains(refreshed, "Set by environment variable <code>KOITO_DISABLE_DEEZER_FILE</code>") || !strings.Contains(refreshed, `value="draft-key-one"`) {
		t.Fatal("saved, overridden or restart state not shown")
	}
	// A KOITO_* override is flagged DEPRECATED with its TUNETRAIL_* name; a TUNETRAIL_* one is not (ISSUE-111).
	if !strings.Contains(refreshed, `class="server-setting-override server-setting-override-deprecated">Set by environment variable <code>KOITO_DISABLE_DEEZER_FILE</code> <strong class="setting-deprecated">(DEPRECATED)</strong> Rename it to <code>TUNETRAIL_DISABLE_DEEZER_FILE</code>.<br>To edit`) {
		t.Fatal("deprecated KOITO_ override not flagged with its new name")
	}
	if !strings.Contains(refreshed, "<code>TUNETRAIL_LOG_LEVEL</code>. ") || strings.Count(refreshed, `class="setting-deprecated"`) != 1 || strings.Count(refreshed, "server-setting-override-deprecated") != 1 {
		t.Fatal("TUNETRAIL_ override must not be flagged as deprecated")
	}
	public, err := client.Get(base + "/apis/web/v1/config")
	if err != nil {
		t.Fatal(err)
	}
	publicBody, _ := io.ReadAll(public.Body)
	public.Body.Close()
	var publicFields map[string]json.RawMessage
	if err := json.Unmarshal(publicBody, &publicFields); err != nil || len(publicFields) != 2 || publicFields["web_ui"] == nil || publicFields["default_theme"] == nil {
		t.Fatal("public configuration contains unexpected fields")
	}
	if strings.Contains(string(publicBody), "draft-key-one") {
		t.Fatal("administrative settings exposed publicly")
	}
	invalidPage := getPage()
	invalidValues := values(invalidPage)
	invalidValues.Set(cfg.LASTFM_API_KEY_ENV, "preserve-invalid-draft")
	invalidValues.Set(cfg.FORCE_TZ, "Not/A_Timezone")
	code, invalidBody, _ := call("POST", "/admin/settings/save", invalidValues, "")
	if code != http.StatusBadRequest || !strings.Contains(invalidBody, "valid IANA") || !strings.Contains(invalidBody, `value="preserve-invalid-draft"`) || extract(invalidBody, "submission_token") == extract(invalidPage, "submission_token") {
		t.Fatalf("invalid draft not preserved with actionable error: %d", code)
	}
	afterInvalid, err := os.ReadFile(conf)
	if err != nil || string(afterInvalid) != string(saved) {
		t.Fatal("invalid request changed configuration")
	}
	headCode, headBody, _ := call("HEAD", "/admin/settings", nil, "")
	if headCode != 200 || headBody != "" {
		t.Fatal("HEAD did not preserve headers without body")
	}
	duplicatePage := getPage()
	duplicate := values(duplicatePage)
	duplicate.Add(cfg.LOG_LEVEL_ENV, "debug")
	code, _, _ = call("POST", "/admin/settings/save", duplicate, "")
	if code != http.StatusBadRequest {
		t.Fatalf("duplicate field status %d", code)
	}
	pathPage := getPage()
	pathInput := values(pathPage)
	pathInput.Set("KOITO_LOG_LEVEL_FILE", "/unexpected/path")
	code, _, _ = call("POST", "/admin/settings/save", pathInput, "")
	if code != http.StatusBadRequest {
		t.Fatalf("unexpected _FILE input status %d", code)
	}
	oversizedPage := getPage()
	oversized := values(oversizedPage)
	oversized.Set(cfg.LASTFM_API_KEY_ENV, strings.Repeat("x", 65<<10))
	code, _, _ = call("POST", "/admin/settings/save", oversized, "")
	if code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized form status %d", code)
	}
	var calls atomic.Int32
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != "/rest/ping.view" {
			t.Errorf("unexpected probe path")
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"subsonic-response":{"status":"ok"}}`)
	}))
	defer mock.Close()
	draft := values(refreshed)
	draft.Set(cfg.LASTFM_API_KEY_ENV, "draft-not-saved")
	draft.Set(cfg.SUBSONIC_URL_ENV, mock.URL)
	draft.Set(cfg.SUBSONIC_PARAMS_ENV, "u=demo&p=demo")
	code, body, _ = call("POST", "/admin/settings/subsonic/test", draft, "")
	if code != 200 || calls.Load() != 1 || !strings.Contains(body, "draft has not been saved") || !strings.Contains(body, `value="draft-not-saved"`) || !strings.Contains(body, `value="u=demo&amp;p=demo"`) {
		t.Fatalf("test draft failed: %d", code)
	}
	after, err := os.ReadFile(conf)
	if err != nil || string(after) != string(saved) {
		t.Fatal("Subsonic test modified configuration")
	}
	testAgain := values(getPage())
	testAgain.Set(cfg.SUBSONIC_URL_ENV, mock.URL)
	testAgain.Set(cfg.SUBSONIC_PARAMS_ENV, "u=demo&p=demo&f=evil")
	code, _, _ = call("POST", "/admin/settings/subsonic/test", testAgain, "")
	if code != 400 || calls.Load() != 1 {
		t.Fatalf("invalid draft reached server: %d calls %d", code, calls.Load())
	}
	saveDraft := values(getPage())
	saveDraft.Set(cfg.MUSICBRAINZ_URL_ENV, "http://localhost:5000/base")
	saveDraft.Set(cfg.MUSICBRAINZ_RATE_LIMIT_ENV, "3")
	saveDraft.Set(cfg.DISABLE_COVER_ART_ARCHIVE_ENV, "false")
	saveDraft.Set(cfg.LASTFM_API_KEY_ENV, "fixture-after-restart")
	saveDraft.Set(cfg.FORCE_TZ, "America/Sao_Paulo")
	saveDraft.Set(cfg.SUBSONIC_URL_ENV, mock.URL)
	saveDraft.Set(cfg.SUBSONIC_PARAMS_ENV, "u=demo&p=demo")
	code, _, _ = call("POST", "/admin/settings/save", saveDraft, "")
	if code != http.StatusSeeOther || calls.Load() != 1 {
		t.Fatalf("saving unexpectedly tested remote: %d calls %d", code, calls.Load())
	}
	// Restart the same disposable installation and compare all running fields.
	_ = cmd.Process.Kill()
	_ = cmd.Wait()
	cmd = exec.Command(os.Args[0], "-test.run=^TestSettingsHTTPJourney$")
	cmd.Env, cmd.Stdout, cmd.Stderr = env, io.Discard, io.Discard
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	var restarted string
	for until := time.Now().Add(15 * time.Second); time.Now().Before(until); time.Sleep(100 * time.Millisecond) {
		resp, e := client.Get(base + "/admin/settings")
		if e != nil {
			continue
		}
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			restarted = string(b)
			break
		}
	}
	if strings.Contains(restarted, `name="username"`) {
		code, _, _ = call("POST", "/admin/login", url.Values{"csrf_token": {extract(restarted, "csrf_token")}, "username": {"admin"}, "password": {"fixture-only"}, "next": {"/admin/settings"}}, "")
		if code != http.StatusSeeOther {
			t.Fatalf("restart login: %d", code)
		}
		restarted = getPage()
	}
	expected := map[string]string{
		cfg.LOG_LEVEL_ENV: "error", cfg.MUSICBRAINZ_URL_ENV: "http://localhost:5000/base", cfg.MUSICBRAINZ_RATE_LIMIT_ENV: "3",
		cfg.DISABLE_MUSICBRAINZ_ENV: "true", cfg.DISABLE_DEEZER_ENV: "true", cfg.DISABLE_COVER_ART_ARCHIVE_ENV: "false",
		cfg.LASTFM_API_KEY_ENV: "fixture-after-restart", cfg.SUBSONIC_URL_ENV: mock.URL, cfg.SUBSONIC_PARAMS_ENV: "u=demo&p=demo", cfg.FORCE_TZ: "America/Sao_Paulo",
	}
	for name, value := range expected {
		section := ""
		for _, candidate := range strings.Split(restarted, `<section class="server-setting">`) {
			candidate = strings.SplitN(candidate, "</section>", 2)[0]
			if strings.Contains(candidate, `name="`+name+`"`) {
				section = candidate
				break
			}
		}
		if !strings.Contains(section, "<dt>Running</dt><dd>"+html.EscapeString(value)+"</dd>") {
			t.Fatalf("running value for %s after restart missing or incorrect", name)
		}
	}
	if strings.Contains(restarted, "Restart to apply it") || !strings.Contains(restarted, "Set by environment variable <code>TUNETRAIL_LOG_LEVEL</code>") || calls.Load() != 2 {
		t.Fatal("restart state, environment override or Subsonic startup ping incorrect")
	}
	// Removing the external override activates the saved value on the next restart.
	_ = cmd.Process.Kill()
	_ = cmd.Wait()
	withoutOverride := make([]string, 0, len(env))
	for _, item := range env {
		if !strings.HasPrefix(item, "TUNETRAIL_LOG_LEVEL=") {
			withoutOverride = append(withoutOverride, item)
		}
	}
	cmd = exec.Command(os.Args[0], "-test.run=^TestSettingsHTTPJourney$")
	cmd.Env, cmd.Stdout, cmd.Stderr = withoutOverride, io.Discard, io.Discard
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	restarted = ""
	for until := time.Now().Add(15 * time.Second); time.Now().Before(until); time.Sleep(100 * time.Millisecond) {
		resp, e := client.Get(base + "/admin/settings")
		if e != nil {
			continue
		}
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			restarted = string(b)
			break
		}
	}
	if strings.Contains(restarted, `name="username"`) {
		code, _, _ = call("POST", "/admin/login", url.Values{"csrf_token": {extract(restarted, "csrf_token")}, "username": {"admin"}, "password": {"fixture-only"}, "next": {"/admin/settings"}}, "")
		if code != http.StatusSeeOther {
			t.Fatalf("second restart login: %d", code)
		}
		restarted = getPage()
	}
	if !strings.Contains(restarted, "<dt>Running</dt><dd>info</dd>") || strings.Contains(restarted, "Set by environment variable <code>TUNETRAIL_LOG_LEVEL</code>") || calls.Load() != 3 {
		t.Fatal("saved value did not become active after removing override")
	}
}
