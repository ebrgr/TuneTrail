package cfg

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

func testSettingsStore(t *testing.T, content string, env map[string]string) (*SettingsStore, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "conf.toml")
	if content != "" {
		if err := os.WriteFile(path, []byte(content), 0640); err != nil {
			t.Fatal(err)
		}
	}
	if env == nil {
		env = map[string]string{}
	}
	return newSettingsStore(path, func(k string) string { return env[k] }, copySettings(managedDefaults)), path
}

// TestSettingsCreate verifies partial updates, defaults, permissions and restart state.
func TestSettingsCreate(t *testing.T) {
	s, path := testSettingsStore(t, "", nil)
	ctx := context.Background()
	before, err := s.Read(ctx)
	if err != nil || before.Exists || before.RestartRequired {
		t.Fatalf("initial state: %+v %v", before, err)
	}
	after, err := s.Save(ctx, before.Revision, map[string]string{LOG_LEVEL_ENV: "debug", DISABLE_DEEZER_ENV: "true"})
	if err != nil {
		t.Fatal(err)
	}
	if !after.Exists || !after.RestartRequired || after.Saved[LOG_LEVEL_ENV] != "debug" || after.Active[LOG_LEVEL_ENV] != "info" {
		t.Fatalf("unexpected state: %+v", after)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("new mode: %v %v", info, err)
	}
	doc, _, present, err := readTOML(path)
	if err != nil || !present || documentValues(doc)[DISABLE_DEEZER_ENV] != "true" {
		t.Fatalf("saved TOML invalid: %v", err)
	}
	if _, err = s.Save(ctx, before.Revision, map[string]string{LOG_LEVEL_ENV: "warn"}); !errors.Is(err, ErrSettingsConflict) {
		t.Fatalf("stale revision: %v", err)
	}
}

// TestSettingsCommentsAndFormatting checks only edited scalar values and inserted fields change.
func TestSettingsCommentsAndFormatting(t *testing.T) {
	original := "schema_version = 1\r\n# global comment\r\n[logging] # section\r\nlevel = 'info'  # log comment\r\n[images]\r\ndisable_deezer = false # keep\r\nlastfm_api_key = 'old # value' # key comment\r\n"
	s, path := testSettingsStore(t, original, nil)
	before, err := s.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	after, err := s.Save(context.Background(), before.Revision, map[string]string{LOG_LEVEL_ENV: "warn", LASTFM_API_KEY_ENV: "new # value", DISABLE_COVER_ART_ARCHIVE_ENV: "true"})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, part := range []string{"# global comment\r\n", "[logging] # section\r\n", `level = "warn"  # log comment` + "\r\n", "disable_deezer = false # keep\r\n", `lastfm_api_key = "new # value" # key comment` + "\r\n", "disable_cover_art_archive = true\r\n"} {
		if !strings.Contains(text, part) {
			t.Fatalf("lost text %q: %s", part, text)
		}
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0640 {
		t.Fatalf("mode changed: %v %v", info, err)
	}
	if after.Saved[LASTFM_API_KEY_ENV] != "new # value" {
		t.Fatal("saved value changed")
	}
}

// TestSettingsOverride verifies that a saved value can remain overridden without a false restart indication.
func TestSettingsOverride(t *testing.T) {
	env := map[string]string{LOG_LEVEL_ENV: "warn"}
	s, _ := testSettingsStore(t, "schema_version=1\n[logging]\nlevel=\"info\"\n", env)
	s.active[LOG_LEVEL_ENV] = "warn"
	before, err := s.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Save(context.Background(), before.Revision, map[string]string{LOG_LEVEL_ENV: "debug"})
	if !errors.Is(err, ErrSettingsOverridden) {
		t.Fatalf("override edit accepted: %v", err)
	}
	after, err := s.Save(context.Background(), before.Revision, map[string]string{LASTFM_API_KEY_ENV: "fixture"})
	if err != nil {
		t.Fatal(err)
	}
	if after.Saved[LOG_LEVEL_ENV] != "info" || after.Effective[LOG_LEVEL_ENV] != "warn" || after.Origins[LOG_LEVEL_ENV] != OriginEnvironment || after.SavedOrigins[LOG_LEVEL_ENV] != OriginTOML || after.SavedOrigins[FORCE_TZ] != OriginDefault {
		t.Fatalf("override not represented: %+v", after)
	}
}

// TestSettingsConcurrentWriters ensures a single form revision cannot overwrite another save.
func TestSettingsConcurrentWriters(t *testing.T) {
	s, path := testSettingsStore(t, "schema_version=1\n", nil)
	before, err := s.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	other := newSettingsStore(path, s.getenv, copySettings(managedDefaults))
	var wg sync.WaitGroup
	results := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			target := s
			if i%2 == 0 {
				target = other
			}
			_, err := target.Save(context.Background(), before.Revision, map[string]string{LOG_LEVEL_ENV: "warn"})
			results <- err
		}(i)
	}
	wg.Wait()
	close(results)
	wins, conflicts := 0, 0
	for err := range results {
		if err == nil {
			wins++
		} else if errors.Is(err, ErrSettingsConflict) {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	if wins != 1 || conflicts != 7 {
		t.Fatalf("wins=%d conflicts=%d", wins, conflicts)
	}
}

// TestSettingsUnsupportedSyntaxNeverOverwrites covers valid layouts the editor cannot preserve.
func TestSettingsUnsupportedSyntaxNeverOverwrites(t *testing.T) {
	for name, content := range map[string]string{
		"multiline": "schema_version=1\n[images]\nlastfm_api_key = \"\"\"line one\nline two\"\"\"\n",
		"dotted":    "schema_version=1\nlogging.level=\"info\"\n",
		"inline":    "schema_version=1\nimages={ disable_deezer=true }\n",
		"quoted":    "schema_version=1\n[logging]\n\"level\" = \"info\"\n",
	} {
		t.Run(name, func(t *testing.T) {
			s, path := testSettingsStore(t, content, nil)
			state, err := s.Read(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if _, err = s.Save(context.Background(), state.Revision, map[string]string{LOG_LEVEL_ENV: "warn"}); !errors.Is(err, ErrUnsupportedSettingsFormat) {
				t.Fatalf("expected unsupported format: %v", err)
			}
			after, err := os.ReadFile(path)
			if err != nil || string(after) != content {
				t.Fatal("unsupported document changed")
			}
		})
	}
}

// TestSettingsInvalidDoesNotPublish rejects stored and effective validation errors.
func TestSettingsInvalidDoesNotPublish(t *testing.T) {
	env := map[string]string{MUSICBRAINZ_RATE_LIMIT_ENV: "1"}
	s, path := testSettingsStore(t, "schema_version=1\n", env)
	before, err := s.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Save(context.Background(), before.Revision, map[string]string{MUSICBRAINZ_RATE_LIMIT_ENV: "2"}); !errors.Is(err, ErrSettingsOverridden) {
		t.Fatalf("invalid stored configuration accepted: %v", err)
	}
	env[MUSICBRAINZ_RATE_LIMIT_ENV] = "2"
	if _, err = s.Save(context.Background(), before.Revision, map[string]string{LOG_LEVEL_ENV: "warn"}); !errors.Is(err, ErrInvalidSettings) {
		t.Fatalf("invalid effective configuration accepted: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil || string(raw) != "schema_version=1\n" {
		t.Fatal("failed validation changed file")
	}
}

// TestSettingsFailureStages distinguishes errors before and after publication.
func TestSettingsFailureStages(t *testing.T) {
	for _, phase := range []string{"sync-file", "rename", "sync-directory"} {
		t.Run(phase, func(t *testing.T) {
			original := "schema_version=1\n"
			s, path := testSettingsStore(t, original, nil)
			before, err := s.Read(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			failure := errors.New("injected failure")
			switch phase {
			case "sync-file":
				s.syncFile = func(*os.File) error { return failure }
			case "rename":
				s.rename = func(string, string) error { return failure }
			case "sync-directory":
				s.syncDir = func(string) error { return failure }
			}
			after, err := s.Save(context.Background(), before.Revision, map[string]string{LOG_LEVEL_ENV: "warn"})
			if err == nil {
				t.Fatal("failure ignored")
			}
			raw, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if phase == "sync-directory" {
				if !errors.Is(err, ErrSettingsDurability) || string(raw) == original || after.Revision != settingsRevision(raw, true) {
					t.Fatal("published-but-uncertain state incorrect")
				}
			} else if string(raw) != original || errors.Is(err, ErrSettingsDurability) {
				t.Fatal("pre-publication failure changed file")
			}
			matches, globErr := filepath.Glob(filepath.Join(filepath.Dir(path), ".conf.toml-*.tmp"))
			if globErr != nil || len(matches) != 0 {
				t.Fatalf("temporary file left behind: %v", matches)
			}
		})
	}
}

// TestSettingsExternalModification is checked immediately before the rename.
func TestSettingsExternalModification(t *testing.T) {
	s, path := testSettingsStore(t, "schema_version=1\n", nil)
	before, err := s.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	external := "schema_version=1\n# edited by another process\n"
	s.beforePublish = func() error { return os.WriteFile(path, []byte(external), 0640) }
	if _, err = s.Save(context.Background(), before.Revision, map[string]string{LOG_LEVEL_ENV: "warn"}); !errors.Is(err, ErrSettingsConflict) {
		t.Fatalf("external edit not detected: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil || string(raw) != external {
		t.Fatal("external edit overwritten")
	}
}

// TestSettingsReadOnlyAndLinks ensures writes refuse read-only files and symlink targets.
func TestSettingsReadOnlyAndLinks(t *testing.T) {
	s, path := testSettingsStore(t, "schema_version=1\n", nil)
	before, err := s.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Chmod(path, 0444); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Save(context.Background(), before.Revision, map[string]string{LOG_LEVEL_ENV: "warn"}); !errors.Is(err, ErrSettingsReadOnly) {
		t.Fatalf("read-only file accepted: %v", err)
	}
	if err = os.Chmod(path, 0640); err != nil {
		t.Fatal(err)
	}
	target := path + ".target"
	if err = os.Rename(path, target); err != nil {
		t.Fatal(err)
	}
	if err = os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
	state, err := s.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Save(context.Background(), state.Revision, map[string]string{LOG_LEVEL_ENV: "warn"}); !errors.Is(err, ErrSettingsPath) {
		t.Fatalf("symlink accepted: %v", err)
	}
	raw, err := os.ReadFile(target)
	if err != nil || string(raw) != "schema_version=1\n" {
		t.Fatal("symlink target modified")
	}
}

// TestSettingsLockCancellation checks cancellation while another writer holds the cooperative lock.
func TestSettingsLockCancellation(t *testing.T) {
	s, _ := testSettingsStore(t, "schema_version=1\n", nil)
	before, err := s.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	held, err := lockSettings(context.Background(), s.path)
	if err != nil {
		t.Fatal(err)
	}
	defer unlockSettings(held)
	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()
	if _, err = s.Save(ctx, before.Revision, map[string]string{LOG_LEVEL_ENV: "warn"}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("lock wait did not cancel: %v", err)
	}
}

// TestSettingsAppendToExistingFinalSectionBeforeNewSection covers mixed insertions at EOF.
func TestSettingsAppendToExistingFinalSectionBeforeNewSection(t *testing.T) {
	s, _ := testSettingsStore(t, "schema_version=1\n[time]\n# clock\n", nil)
	state, err := s.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	after, err := s.Save(context.Background(), state.Revision, map[string]string{LOG_LEVEL_ENV: "debug", FORCE_TZ: "UTC"})
	if err != nil {
		t.Fatal(err)
	}
	if after.Saved[LOG_LEVEL_ENV] != "debug" || after.Saved[FORCE_TZ] != "UTC" {
		t.Fatal("fields assigned to incorrect sections")
	}
}

// TestSettingsCanRepairEffectiveConflict validates the candidate, not stale effective values.
func TestSettingsCanRepairEffectiveConflict(t *testing.T) {
	env := map[string]string{}
	s, _ := testSettingsStore(t, "schema_version=1\n", env)
	state, err := s.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	env[MUSICBRAINZ_RATE_LIMIT_ENV] = "2"
	after, err := s.Save(context.Background(), state.Revision, map[string]string{MUSICBRAINZ_URL_ENV: "http://localhost:5000"})
	if err != nil {
		t.Fatal(err)
	}
	if after.Effective[MUSICBRAINZ_RATE_LIMIT_ENV] != "2" || after.Effective[MUSICBRAINZ_URL_ENV] != "http://localhost:5000" {
		t.Fatal("valid repair rejected")
	}
}

// TestSettingsNormalizesIntegerInput checks equivalent supported integer forms.
func TestSettingsNormalizesIntegerInput(t *testing.T) {
	s, _ := testSettingsStore(t, "schema_version=1\n", nil)
	state, err := s.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	after, err := s.Save(context.Background(), state.Revision, map[string]string{MUSICBRAINZ_RATE_LIMIT_ENV: "+1"})
	if err != nil || after.Saved[MUSICBRAINZ_RATE_LIMIT_ENV] != "1" {
		t.Fatalf("integer normalization: %v", err)
	}
}

// TestSettingsReadWaitsForCooperativeEditor avoids observing an intermediate external write.
func TestSettingsReadWaitsForCooperativeEditor(t *testing.T) {
	s, path := testSettingsStore(t, "schema_version=1\n", nil)
	held, err := lockSettings(context.Background(), s.path)
	if err != nil {
		t.Fatal(err)
	}
	unlocked := false
	defer func() {
		if !unlocked {
			unlockSettings(held)
		}
	}()
	if err = os.WriteFile(path, []byte("schema_version=["), 0640); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { _, err := s.Read(context.Background()); done <- err }()
	select {
	case err := <-done:
		t.Fatalf("read did not wait for writer: %v", err)
	case <-time.After(80 * time.Millisecond):
	}
	if err = os.WriteFile(path, []byte("schema_version=1\n"), 0640); err != nil {
		t.Fatal(err)
	}
	unlockSettings(held)
	unlocked = true
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("read did not resume")
	}
}

// TestSettingsRejectFIFO ensures a special config path cannot hang readers or writers.
func TestSettingsRejectFIFO(t *testing.T) {
	s, path := testSettingsStore(t, "", nil)
	if err := syscall.Mkfifo(path, 0600); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { _, err := s.Read(context.Background()); done <- err }()
	select {
	case err := <-done:
		if !errors.Is(err, ErrSettingsPath) {
			t.Fatalf("FIFO accepted: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("reading FIFO blocked")
	}
	if _, _, _, err := readTOML(path); !errors.Is(err, ErrSettingsPath) {
		t.Fatalf("startup accepted FIFO: %v", err)
	}
}

// TestSettingsSaveAllFields verifies the complete supported configuration round trip.
func TestSettingsSaveAllFields(t *testing.T) {
	s, _ := testSettingsStore(t, "", nil)
	before, err := s.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	values := map[string]string{
		LOG_LEVEL_ENV: "debug", MUSICBRAINZ_URL_ENV: "http://localhost:5000/base", MUSICBRAINZ_RATE_LIMIT_ENV: "3",
		DISABLE_MUSICBRAINZ_ENV: "true", DISABLE_DEEZER_ENV: "true", DISABLE_COVER_ART_ARCHIVE_ENV: "true",
		LASTFM_API_KEY_ENV: "visible-value", SUBSONIC_URL_ENV: "http://localhost:4533/base", SUBSONIC_PARAMS_ENV: "u=test&t=value&s=value", FORCE_TZ: "America/Sao_Paulo",
	}
	after, err := s.Save(context.Background(), before.Revision, values)
	if err != nil {
		t.Fatal(err)
	}
	for k, want := range values {
		if after.Saved[k] != want || after.Effective[k] != want || after.Origins[k] != OriginTOML {
			t.Fatalf("field %s did not round-trip", k)
		}
	}
	reread, err := s.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if reread.Revision != after.Revision || !reread.RestartRequired {
		t.Fatal("persisted revision or restart state lost")
	}
}

// TestSettingsRuntimeStorePinsStartupDirectory exercises the public constructor in an isolated process.
func TestSettingsRuntimeStorePinsStartupDirectory(t *testing.T) {
	if os.Getenv("TUNETRAIL_TEST_SETTINGS_RUNTIME") == "1" {
		first := t.TempDir()
		second := t.TempDir()
		env := map[string]string{CONFIG_DIR_ENV: first}
		get := func(k string) string { return env[k] }
		if err := Load(get, "test"); err != nil {
			t.Fatal(err)
		}
		env[CONFIG_DIR_ENV] = second
		s, err := NewSettingsStore(get)
		if err != nil {
			t.Fatal(err)
		}
		if s.SettingsPath() != filepath.Join(first, "conf.toml") {
			t.Fatal("startup directory was not pinned")
		}
		before, err := s.Read(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		after, err := s.Save(context.Background(), before.Revision, map[string]string{LOG_LEVEL_ENV: "warn"})
		if err != nil {
			t.Fatal(err)
		}
		if !after.RestartRequired || after.Active[LOG_LEVEL_ENV] != "info" || LogLevel() != 1 {
			t.Fatal("save changed active process configuration")
		}
		if _, err = os.Stat(filepath.Join(first, "conf.toml")); err != nil {
			t.Fatal(err)
		}
		if _, err = os.Stat(filepath.Join(second, "conf.toml")); !os.IsNotExist(err) {
			t.Fatal("unexpected file created in alternate directory")
		}
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestSettingsRuntimeStorePinsStartupDirectory$")
	cmd.Env = append(os.Environ(), "TUNETRAIL_TEST_SETTINGS_RUNTIME=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("child failed: %v %s", err, string(out))
	}
}

// TestSettingsRejectFIFOLock prevents a special lock path from blocking reads or writes.
func TestSettingsRejectFIFOLock(t *testing.T) {
	s, path := testSettingsStore(t, "schema_version=1\n", nil)
	before, err := s.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err = syscall.Mkfifo(path+".lock", 0600); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { _, err := s.Read(context.Background()); done <- err }()
	select {
	case err := <-done:
		if !errors.Is(err, ErrSettingsPath) {
			t.Fatalf("FIFO lock accepted: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("reading FIFO lock blocked")
	}
	if _, err = s.Save(context.Background(), before.Revision, map[string]string{LOG_LEVEL_ENV: "warn"}); !errors.Is(err, ErrSettingsPath) {
		t.Fatalf("writing with FIFO lock accepted: %v", err)
	}
}

// TestSettingsRejectSymlinkLock keeps the cooperative lock from following a link.
func TestSettingsRejectSymlinkLock(t *testing.T) {
	s, path := testSettingsStore(t, "schema_version=1\n", nil)
	state, err := s.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(filepath.Dir(path), "other")
	if err = os.WriteFile(target, []byte("untouched"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.Symlink(target, path+".lock"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Save(context.Background(), state.Revision, map[string]string{LOG_LEVEL_ENV: "warn"}); err == nil {
		t.Fatal("symlink lock accepted")
	}
	data, err := os.ReadFile(target)
	if err != nil || string(data) != "untouched" {
		t.Fatal("lock target modified")
	}
}

// TestSettingsRejectOversizedValues avoids constructing an unbounded candidate document.
func TestSettingsRejectOversizedValues(t *testing.T) {
	s, path := testSettingsStore(t, "schema_version=1\n", nil)
	state, err := s.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Save(context.Background(), state.Revision, map[string]string{LASTFM_API_KEY_ENV: strings.Repeat("a", (64<<10)+1)}); !errors.Is(err, ErrInvalidSettings) {
		t.Fatalf("oversized input accepted: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil || string(raw) != "schema_version=1\n" {
		t.Fatal("oversized input changed document")
	}
}

// TestSettingsReadDoesNotCreateLock verifies that viewing a readonly directory does not require writes.
func TestSettingsReadDoesNotCreateLock(t *testing.T) {
	s, path := testSettingsStore(t, "", nil)
	dir := filepath.Dir(path)
	if err := os.Chmod(dir, 0555); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(dir, 0700)
	if _, err := s.Read(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(path + ".lock"); !os.IsNotExist(err) {
		t.Fatal("read unexpectedly created lock file")
	}
}

// TestSettingsRejectsOverrideAppearingBeforePublish keeps the original file when the environment changes.
func TestSettingsRejectsOverrideAppearingBeforePublish(t *testing.T) {
	env := map[string]string{}
	s, path := testSettingsStore(t, "schema_version=1\n", env)
	before, err := s.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	s.beforePublish = func() error { env[LOG_LEVEL_ENV] = "warn"; return nil }
	_, err = s.Save(context.Background(), before.Revision, map[string]string{LOG_LEVEL_ENV: "debug"})
	if !errors.Is(err, ErrSettingsOverridden) {
		t.Fatalf("late override accepted: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil || string(raw) != "schema_version=1\n" {
		t.Fatal("late override changed the file")
	}
}
