package cfg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// envMap is a fake environment for the precedence tests (ISSUE-111).
func envMap(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func secretFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestResolveEnvPrecedence covers every row of the table in env_alias.go for a
// plain variable and its _FILE form, in both prefixes.
func TestResolveEnvPrecedence(t *testing.T) {
	key := LOG_LEVEL_ENV
	file := func(v string) string { return secretFile(t, v+"\n") }
	for _, tc := range []struct {
		name       string
		env        map[string]string
		value      string
		source     string
		origin     Origin
		notice     string
		errContain string
	}{
		{"none", nil, "", "", OriginDefault, "", ""},
		{"new only", map[string]string{"TUNETRAIL_LOG_LEVEL": "debug"}, "debug", "TUNETRAIL_LOG_LEVEL", OriginEnvironment, "", ""},
		{"new file only", map[string]string{"TUNETRAIL_LOG_LEVEL_FILE": file("warn")}, "warn", "TUNETRAIL_LOG_LEVEL_FILE", OriginFile, "", ""},
		{"old only", map[string]string{"KOITO_LOG_LEVEL": "debug"}, "debug", "KOITO_LOG_LEVEL", OriginEnvironment, "rename it to TUNETRAIL_LOG_LEVEL", ""},
		{"old file only", map[string]string{"KOITO_LOG_LEVEL_FILE": file("warn")}, "warn", "KOITO_LOG_LEVEL_FILE", OriginFile, "rename it to TUNETRAIL_LOG_LEVEL_FILE", ""},
		{"both equal", map[string]string{"TUNETRAIL_LOG_LEVEL": "info", "KOITO_LOG_LEVEL": "info"}, "info", "TUNETRAIL_LOG_LEVEL", OriginEnvironment, "remove the deprecated KOITO_LOG_LEVEL", ""},
		{"new file equals old", map[string]string{"TUNETRAIL_LOG_LEVEL_FILE": file("info"), "KOITO_LOG_LEVEL": "info"}, "info", "TUNETRAIL_LOG_LEVEL_FILE", OriginFile, "remove the deprecated KOITO_LOG_LEVEL", ""},
		{"both differ", map[string]string{"TUNETRAIL_LOG_LEVEL": "info", "KOITO_LOG_LEVEL": "debug"}, "", "", "", "", "TUNETRAIL_LOG_LEVEL and KOITO_LOG_LEVEL differ"},
		{"file and plain differ", map[string]string{"TUNETRAIL_LOG_LEVEL_FILE": file("warn"), "KOITO_LOG_LEVEL": "debug"}, "", "", "", "", "TUNETRAIL_LOG_LEVEL_FILE and KOITO_LOG_LEVEL differ"},
		{"comparison is exact", map[string]string{"TUNETRAIL_LOG_LEVEL": "INFO", "KOITO_LOG_LEVEL": "info"}, "", "", "", "", "differ"},
		{"empty counts as absent", map[string]string{"TUNETRAIL_LOG_LEVEL": "", "KOITO_LOG_LEVEL": "debug"}, "debug", "KOITO_LOG_LEVEL", OriginEnvironment, "rename", ""},
		{"empty file content is present", map[string]string{"TUNETRAIL_LOG_LEVEL_FILE": file("")}, "", "TUNETRAIL_LOG_LEVEL_FILE", OriginFile, "", ""},
		{"unreadable old file fails closed", map[string]string{"TUNETRAIL_LOG_LEVEL": "info", "KOITO_LOG_LEVEL_FILE": "/nonexistent/secret"}, "", "", "", "", "cannot read KOITO_LOG_LEVEL_FILE"},
		{"unreadable new file", map[string]string{"TUNETRAIL_LOG_LEVEL_FILE": "/nonexistent/secret"}, "", "", "", "", "cannot read TUNETRAIL_LOG_LEVEL_FILE"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, notice, err := resolveEnv(envMap(tc.env), key)
			if tc.errContain != "" {
				if err == nil || !strings.Contains(err.Error(), tc.errContain) {
					t.Fatalf("error %v, want %q", err, tc.errContain)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.value != tc.value || got.source != tc.source || got.origin != tc.origin {
				t.Fatalf("got %+v", got)
			}
			if (tc.notice == "") != (notice == "") || !strings.Contains(notice, tc.notice) {
				t.Fatalf("notice %q, want %q", notice, tc.notice)
			}
		})
	}
}

// TestConflictNeverRevealsValues: secrets must not reach stderr through the error.
func TestConflictNeverRevealsValues(t *testing.T) {
	for _, env := range []map[string]string{
		{"TUNETRAIL_DEFAULT_PASSWORD": "new-secret-value", "KOITO_DEFAULT_PASSWORD": "old-secret-value"},
		{"TUNETRAIL_DEFAULT_PASSWORD_FILE": secretFile(t, "new-secret-value"), "KOITO_DEFAULT_PASSWORD": "old-secret-value"},
		{"TUNETRAIL_SUBSONIC_PARAMS": "u=a&p=new-secret-value", "KOITO_SUBSONIC_PARAMS": "u=a&p=old-secret-value"},
	} {
		_, err := scanEnvironment(envMap(env))
		if err == nil {
			t.Fatal("conflict accepted")
		}
		if strings.Contains(err.Error(), "secret-value") {
			t.Fatalf("error reveals a value: %v", err)
		}
	}
}

// TestScanDetectsConflictInUnreadKey: the relay token is only read when the relay
// is enabled, but a conflict in it must still stop startup.
func TestScanDetectsConflictInUnreadKey(t *testing.T) {
	_, err := scanEnvironment(envMap(map[string]string{"TUNETRAIL_LBZ_RELAY_TOKEN": "a", "KOITO_LBZ_RELAY_TOKEN": "b"}))
	if err == nil || !strings.Contains(err.Error(), "LBZ_RELAY_TOKEN") {
		t.Fatalf("conflict in an unread key not detected: %v", err)
	}
}

// TestScanCollectsOneNoticePerVariable: notices come from the startup scan only.
func TestScanCollectsOneNoticePerVariable(t *testing.T) {
	reader, err := scanEnvironment(envMap(map[string]string{"KOITO_LOG_LEVEL": "debug", "KOITO_SKIP_IMPORT": "true", "TUNETRAIL_BIND_ADDR": "127.0.0.1"}))
	if err != nil {
		t.Fatal(err)
	}
	// Reading the same keys again (as configuration loading does) adds nothing.
	reader.get(LOG_LEVEL_ENV)
	reader.get(SKIP_IMPORT_ENV)
	if len(reader.notices) != 2 {
		t.Fatalf("notices %q", reader.notices)
	}
}

// TestLegacyOnlyKeysHaveNoAlias: obsolete keys are not given TUNETRAIL_* names.
func TestLegacyOnlyKeysHaveNoAlias(t *testing.T) {
	for _, key := range []string{WEB_UI_ENV, DATABASE_URL_ENV, SQLITE_ENABLED, ENABLE_FULL_IMAGE_CACHE_ENV} {
		if CurrentName(key) != key {
			t.Fatalf("%s has an alias", key)
		}
		got, _, _ := resolveEnv(envMap(map[string]string{"TUNETRAIL_" + strings.TrimPrefix(key, "KOITO_"): "x"}), key)
		if got.origin != OriginDefault {
			t.Fatalf("%s accepted the TUNETRAIL_ name", key)
		}
	}
	if len(aliasedKeys) != 30 {
		t.Fatalf("%d aliased keys, want 30", len(aliasedKeys))
	}
}

// TestLoadConfigAcceptsNewNames: configuration loading sees TUNETRAIL_* values.
func TestLoadConfigAcceptsNewNames(t *testing.T) {
	reader, err := scanEnvironment(envMap(map[string]string{"TUNETRAIL_LISTEN_PORT": "4999", "TUNETRAIL_LOG_LEVEL": "warn"}))
	if err != nil {
		t.Fatal(err)
	}
	c, err := loadConfig(func(k string) string { v, _ := reader.get(k); return v }, "test")
	if err != nil {
		t.Fatal(err)
	}
	if c.listenPort != 4999 || c.logLevel != 2 {
		t.Fatalf("port %d level %d", c.listenPort, c.logLevel)
	}
}
