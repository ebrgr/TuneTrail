package cfg

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestTOMLPrecedence covers false, empty _FILE, raw comments and external overrides.
func TestTOMLPrecedence(t *testing.T) {
	dir := t.TempDir()
	content := `schema_version = 1
# Keep this comment for future editing.
[logging]
level = "debug"
[musicbrainz]
url = "http://localhost:9999/mbz/"
rate_limit = 4
[images]
disable_deezer = true
lastfm_api_key = "from-file"
[time]
force_tz = "America/Sao_Paulo"
`
	if err := os.WriteFile(filepath.Join(dir, "conf.toml"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	empty := filepath.Join(dir, "empty")
	if err := os.WriteFile(empty, nil, 0600); err != nil {
		t.Fatal(err)
	}
	env := map[string]string{CONFIG_DIR_ENV: dir, LOG_LEVEL_ENV: "warn", DISABLE_DEEZER_ENV: "false", LASTFM_API_KEY_ENV + "_FILE": empty}
	r := &envReader{getenv: func(k string) string { return env[k] }}
	doc, values, err := prepareTOML(r)
	if err != nil {
		t.Fatal(err)
	}
	if !doc.Present || string(doc.Raw) != content {
		t.Fatal("original document not retained")
	}
	for key, want := range map[string]string{LOG_LEVEL_ENV: "warn", DISABLE_DEEZER_ENV: "false", LASTFM_API_KEY_ENV: "", MUSICBRAINZ_URL_ENV: "http://localhost:9999/mbz", MUSICBRAINZ_RATE_LIMIT_ENV: "4", FORCE_TZ: "America/Sao_Paulo"} {
		if values[key] != want {
			t.Errorf("%s got %q want %q", key, values[key], want)
		}
	}
	for key, want := range map[string]Origin{LOG_LEVEL_ENV: OriginEnvironment, DISABLE_DEEZER_ENV: OriginEnvironment, LASTFM_API_KEY_ENV: OriginFile, FORCE_TZ: OriginTOML, DISABLE_MUSICBRAINZ_ENV: OriginDefault} {
		if doc.Origins[key] != want {
			t.Errorf("%s origin %q want %q", key, doc.Origins[key], want)
		}
	}
}

// TestTOMLValidation rejects invalid saved and effective configurations independently.
func TestTOMLValidation(t *testing.T) {
	cases := []struct {
		name, content string
		env           map[string]string
	}{
		{"no schema", "[logging]\nlevel=\"info\"", nil},
		{"new schema", `schema_version=2`, nil},
		{"syntax", `schema_version = [`, nil},
		{"unknown nested", "schema_version=1\n[images]\nunknown = true", nil},
		{"wrong type", "schema_version=1\n[images]\ndisable_deezer=\"true\"", nil},
		{"invalid stored hidden by env", "schema_version=1\n[logging]\nlevel=\"bogus\"", map[string]string{LOG_LEVEL_ENV: "info"}},
		{"official url rate", "schema_version=1\n[musicbrainz]\nurl=\"https://MUSICBRAINZ.ORG:443/\"\nrate_limit=2", nil},
		{"bad override", "schema_version=1", map[string]string{MUSICBRAINZ_RATE_LIMIT_ENV: "2"}},
		{"negative rate", "schema_version=1\n[musicbrainz]\nrate_limit=-1", nil},
		{"invalid timezone", "schema_version=1\n[time]\nforce_tz=\"Not/AZone\"", nil},
		{"subsonic mixed auth", "schema_version=1\n[subsonic]\nurl=\"http://localhost:4533\"\nparams=\"u=a&p=b&t=c&s=d\"", nil},
		{"subsonic partial", "schema_version=1\n[subsonic]\nurl=\"http://localhost:4533\"", nil},
		{"subsonic reserved", "schema_version=1\n[subsonic]\nurl=\"http://localhost:4533\"\nparams=\"u=a&p=b&query=x\"", nil},
		{"subsonic duplicate", "schema_version=1\n[subsonic]\nurl=\"http://localhost:4533\"\nparams=\"u=a&u=b&p=c\"", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "conf.toml"), []byte(tc.content), 0600); err != nil {
				t.Fatal(err)
			}
			env := map[string]string{CONFIG_DIR_ENV: dir}
			for k, v := range tc.env {
				env[k] = v
			}
			_, _, err := prepareTOML(&envReader{getenv: func(k string) string { return env[k] }})
			if err == nil {
				t.Fatal("invalid configuration accepted")
			}
		})
	}
}

// TestLegacyAndFileErrors verifies that missing TOML retains legacy parsing and missing _FILE fails.
func TestLegacyAndFileErrors(t *testing.T) {
	dir := t.TempDir()
	env := map[string]string{CONFIG_DIR_ENV: dir, LOG_LEVEL_ENV: "unknown", MUSICBRAINZ_RATE_LIMIT_ENV: "bad", SUBSONIC_URL_ENV: "http://localhost"}
	r := &envReader{getenv: func(k string) string { return env[k] }}
	doc, values, err := prepareTOML(r)
	if err != nil || doc.Present || values != nil {
		t.Fatalf("legacy preparation: %v", err)
	}
	c, err := loadConfig(func(k string) string { v, _ := r.get(k); return v }, "test")
	if err != nil || c.logLevel != 1 || c.musicBrainzRateLimit != 1 || c.subsonicEnabled {
		t.Fatalf("legacy behavior changed: %+v %v", c, err)
	}
	env[LASTFM_API_KEY_ENV+"_FILE"] = filepath.Join(dir, "missing")
	if _, _, err = prepareTOML(&envReader{getenv: r.getenv}); err == nil {
		t.Fatal("missing _FILE accepted")
	}
}

// TestLoadFailurePersists verifies sync.Once cannot turn an initial error into success.
func TestLoadFailurePersists(t *testing.T) {
	if os.Getenv("TUNETRAIL_TEST_LOAD_FAILURE") == "1" {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "conf.toml"), []byte("broken = ["), 0600); err != nil {
			t.Fatal(err)
		}
		get := func(k string) string {
			if k == CONFIG_DIR_ENV {
				return dir
			}
			return ""
		}
		for i := 0; i < 2; i++ {
			if err := Load(get, "test"); err == nil {
				t.Fatal("failed startup returned success")
			}
		}
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestLoadFailurePersists$")
	cmd.Env = append(os.Environ(), "TUNETRAIL_TEST_LOAD_FAILURE=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("child failed: %v %s", err, strings.TrimSpace(string(out)))
	}
}

// TestConfigDirectorySnapshot prevents a changed _FILE from switching the database directory.
func TestConfigDirectorySnapshot(t *testing.T) {
	first := t.TempDir()
	second := t.TempDir()
	path := filepath.Join(t.TempDir(), "directory")
	if err := os.WriteFile(path, []byte(first), 0600); err != nil {
		t.Fatal(err)
	}
	env := map[string]string{CONFIG_DIR_ENV + "_FILE": path}
	r := &envReader{getenv: func(k string) string { return env[k] }}
	doc, _, err := prepareTOML(r)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(second), 0600); err != nil {
		t.Fatal(err)
	}
	got, _ := r.get(CONFIG_DIR_ENV)
	if got != first || doc.Path != filepath.Join(first, "conf.toml") {
		t.Fatalf("configuration directory changed during startup")
	}
}

// TestLoadTOML publishes effective values and preserves a copy of the input document.
func TestLoadTOML(t *testing.T) {
	if os.Getenv("TUNETRAIL_TEST_LOAD_SUCCESS") == "1" {
		dir := t.TempDir()
		cfg := `schema_version=1
[logging]
level="warn"
[musicbrainz]
url="http://localhost:1234/"
rate_limit=2
disabled=true
[images]
disable_deezer=true
lastfm_api_key="example"
[time]
force_tz="America/Sao_Paulo"
`
		if err := os.WriteFile(filepath.Join(dir, "conf.toml"), []byte(cfg), 0600); err != nil {
			t.Fatal(err)
		}
		dirFile := filepath.Join(t.TempDir(), "dir")
		if err := os.WriteFile(dirFile, []byte(dir), 0600); err != nil {
			t.Fatal(err)
		}
		env := map[string]string{CONFIG_DIR_ENV + "_FILE": dirFile, LOG_LEVEL_ENV: "error"}
		if err := Load(func(k string) string { return env[k] }, "test"); err != nil {
			t.Fatal(err)
		}
		if ConfigDir() != dir || LogLevel() != 3 || !MusicBrainzDisabled() || !DeezerDisabled() || LastFMApiKey() != "example" || ForceTZ().String() != "America/Sao_Paulo" {
			t.Fatal("effective settings not applied")
		}
		d := ConfigurationDocument()
		if !d.Present || d.Origins[LOG_LEVEL_ENV] != OriginEnvironment || d.Origins[FORCE_TZ] != OriginTOML {
			t.Fatal("incorrect configuration origins")
		}
		d.Raw[0] = 'x'
		d.Origins[FORCE_TZ] = OriginDefault
		if ConfigurationDocument().Raw[0] == 'x' || ConfigurationDocument().Origins[FORCE_TZ] != OriginTOML {
			t.Fatal("configuration snapshot is mutable")
		}
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestLoadTOML$")
	cmd.Env = append(os.Environ(), "TUNETRAIL_TEST_LOAD_SUCCESS=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("child failed: %v %s", err, strings.TrimSpace(string(out)))
	}
}

// TestSubsonicParameterExtensions retains compatible non-reserved query values.
func TestSubsonicParameterExtensions(t *testing.T) {
	for _, params := range []string{"u=a&t=b&s=c&custom=value", "u=a&p=b"} {
		values := make(map[string]string, len(managedDefaults))
		for k, v := range managedDefaults {
			values[k] = v
		}
		values[SUBSONIC_URL_ENV] = "http://localhost:4533/base/"
		values[SUBSONIC_PARAMS_ENV] = params
		if err := validateManaged(values); err != nil {
			t.Fatalf("valid params %q rejected: %v", params, err)
		}
		if values[SUBSONIC_URL_ENV] != "http://localhost:4533/base" {
			t.Fatal("base path not preserved")
		}
	}
}

// TestDirectEnvironmentWinsWithoutReadingMissingFile preserves established precedence.
func TestDirectEnvironmentWinsWithoutReadingMissingFile(t *testing.T) {
	env := map[string]string{LOG_LEVEL_ENV: "warn", LOG_LEVEL_ENV + "_FILE": filepath.Join(t.TempDir(), "absent")}
	r := &envReader{getenv: func(k string) string { return env[k] }}
	value, source := r.get(LOG_LEVEL_ENV)
	if value != "warn" || source != OriginEnvironment || r.err != nil {
		t.Fatal("direct environment did not take precedence")
	}
}

// TestEmptyFileOverrideDistinguishesLegacyAndStrictTOML verifies an empty boolean _FILE never falls back to TOML.
func TestEmptyFileOverrideDistinguishesLegacyAndStrictTOML(t *testing.T) {
	dir := t.TempDir()
	empty := filepath.Join(dir, "empty")
	if err := os.WriteFile(empty, nil, 0600); err != nil {
		t.Fatal(err)
	}
	env := map[string]string{CONFIG_DIR_ENV: dir, DISABLE_DEEZER_ENV + "_FILE": empty}
	r := &envReader{getenv: func(k string) string { return env[k] }}
	doc, _, err := prepareTOML(r)
	if err != nil || doc.Present || doc.Origins[DISABLE_DEEZER_ENV] != OriginFile {
		t.Fatal("legacy empty file origin lost")
	}
	c, err := loadConfig(func(k string) string { v, _ := r.get(k); return v }, "test")
	if err != nil || c.disableDeezer {
		t.Fatal("legacy empty boolean must remain false")
	}
	if err := os.WriteFile(filepath.Join(dir, "conf.toml"), []byte("schema_version=1\n[images]\ndisable_deezer=true\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := prepareTOML(&envReader{getenv: r.getenv}); err == nil {
		t.Fatal("empty file fell back to a TOML boolean")
	}
}
