package cfg

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var (
	ErrSettingsConflict          = errors.New("configuration revision conflict")
	ErrSettingsOverridden        = errors.New("configuration field controlled by environment")
	ErrInvalidSettings           = errors.New("invalid configuration")
	ErrUnsupportedSettingsFormat = errors.New("configuration format cannot be edited without losing text")
	ErrSettingsReadOnly          = errors.New("configuration file is read-only")
	ErrSettingsPath              = errors.New("configuration path is not a regular file")
	ErrSettingsDurability        = errors.New("configuration published but durability is uncertain")
)

// SettingsSnapshot describes persisted, currently effective and startup values.
type SettingsSnapshot struct {
	Revision     string
	Exists       bool
	Saved        map[string]string
	Effective    map[string]string
	Active       map[string]string
	Origins      map[string]Origin
	SavedOrigins map[string]Origin
	Overridden   []string
	// Sources names the environment variable that overrides each key, e.g.
	// TUNETRAIL_LOG_LEVEL or KOITO_DISABLE_DEEZER_FILE (ISSUE-111).
	Sources         map[string]string
	RestartRequired bool
}

// SettingsStore uses the directory discovered at startup; callers cannot select a file path per request.
type SettingsStore struct {
	path          string
	getenv        func(string) string
	active        map[string]string
	syncFile      func(*os.File) error
	rename        func(string, string) error
	syncDir       func(string) error
	beforePublish func() error // test hook for external modifications before publication
}

// DurabilityError reports that rename succeeded but the directory sync failed.
type DurabilityError struct{ Cause error }

func (e *DurabilityError) Error() string {
	return fmt.Sprintf("%v: %v", ErrSettingsDurability, e.Cause)
}
func (e *DurabilityError) Unwrap() error { return ErrSettingsDurability }

var settingsGate = make(chan struct{}, 1)

// NewSettingsStore attaches to the immutable directory and values from a successful startup.
func NewSettingsStore(getenv func(string) string) (*SettingsStore, error) {
	lock.RLock()
	defer lock.RUnlock()
	if globalConfig == nil || startupDocument.Path == "" {
		return nil, fmt.Errorf("configuration not initialized")
	}
	if getenv == nil {
		getenv = os.Getenv
	}
	return newSettingsStore(startupDocument.Path, getenv, managedFromConfig(globalConfig)), nil
}
func newSettingsStore(path string, getenv func(string) string, active map[string]string) *SettingsStore {
	return &SettingsStore{path: path, getenv: getenv, active: copySettings(active)}
}
func copySettings(source map[string]string) map[string]string {
	out := make(map[string]string, len(source))
	for k, v := range source {
		out[k] = v
	}
	return out
}

// managedFromConfig returns the actual values loaded at process startup.
func managedFromConfig(c *config) map[string]string {
	level := "info"
	switch c.logLevel {
	case 0:
		level = "debug"
	case 2:
		level = "warn"
	case 3:
		level = "error"
	case 4:
		level = "fatal"
	}
	tz := ""
	if c.forceTZ != nil {
		tz = c.forceTZ.String()
	}
	return map[string]string{LOG_LEVEL_ENV: level, MUSICBRAINZ_URL_ENV: c.musicBrainzUrl, MUSICBRAINZ_RATE_LIMIT_ENV: strconv.Itoa(c.musicBrainzRateLimit), DISABLE_MUSICBRAINZ_ENV: strconv.FormatBool(c.disableMusicBrainz), DISABLE_DEEZER_ENV: strconv.FormatBool(c.disableDeezer), DISABLE_COVER_ART_ARCHIVE_ENV: strconv.FormatBool(c.disableCAA), LASTFM_API_KEY_ENV: c.lastfmApiKey, SUBSONIC_URL_ENV: c.subsonicUrl, SUBSONIC_PARAMS_ENV: c.subsonicParams, FORCE_TZ: tz}
}

// settingsRevision distinguishes a missing file from every possible file body.
func settingsRevision(raw []byte, present bool) string {
	h := sha256.New()
	if present {
		h.Write([]byte{1})
	} else {
		h.Write([]byte{0})
	}
	h.Write(raw)
	return hex.EncodeToString(h.Sum(nil))
}

// makeSnapshot validates saved and effective compositions while retaining legacy behavior without TOML.
func (s *SettingsStore) makeSnapshot(raw []byte, present bool, r *envReader) (SettingsSnapshot, error) {
	state := SettingsSnapshot{Revision: settingsRevision(raw, present), Exists: present, Saved: copySettings(managedDefaults), Active: copySettings(s.active), Origins: map[string]Origin{}, SavedOrigins: map[string]Origin{}, Sources: map[string]string{}}
	for k := range managedDefaults {
		state.Origins[k] = OriginDefault
		state.SavedOrigins[k] = OriginDefault
	}
	if present {
		parsed, err := decodeTOML(raw)
		if err != nil {
			return SettingsSnapshot{}, fmt.Errorf("%w: %v", ErrInvalidSettings, err)
		}
		for k, v := range documentValues(parsed) {
			state.Saved[k] = v
			state.Origins[k] = OriginTOML
			state.SavedOrigins[k] = OriginTOML
		}
	}
	if err := validateManaged(state.Saved); err != nil {
		return SettingsSnapshot{}, fmt.Errorf("%w: %v", ErrInvalidSettings, err)
	}
	state.Effective = copySettings(state.Saved)
	for k := range managedDefaults {
		value, origin := r.get(k)
		if r.err != nil {
			return SettingsSnapshot{}, r.err
		}
		if origin != OriginDefault {
			state.Effective[k] = value
			state.Origins[k] = origin
			state.Sources[k] = r.source(k)
			state.Overridden = append(state.Overridden, k)
		}
	}
	sort.Strings(state.Overridden)
	if present {
		if err := validateManaged(state.Effective); err != nil {
			return SettingsSnapshot{}, fmt.Errorf("%w: effective configuration: %v", ErrInvalidSettings, err)
		}
	} else {
		// The legacy parser intentionally keeps its old fallback behavior until a TOML file is created.
		legacy, err := loadConfig(func(k string) string { v, _ := r.get(k); return v }, "")
		if r.err != nil {
			return SettingsSnapshot{}, r.err
		}
		if err != nil {
			return SettingsSnapshot{}, err
		}
		state.Effective = managedFromConfig(legacy)
	}
	state.RestartRequired = !reflect.DeepEqual(state.Active, state.Effective)
	return state, nil
}

// readRawSettings reads a bounded snapshot and leaves reading existing symlinks to the legacy reader.
func readRawSettings(path string) ([]byte, bool, error) {
	// O_NONBLOCK prevents hanging when an unexpected path names a FIFO.
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0)
	if errors.Is(err, syscall.ENOENT) {
		if _, e := os.Lstat(path); e == nil {
			return nil, false, ErrSettingsPath
		}
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	f := os.NewFile(uintptr(fd), path)
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, false, err
	}
	if !info.Mode().IsRegular() {
		return nil, false, ErrSettingsPath
	}
	raw, err := io.ReadAll(io.LimitReader(f, (64<<10)+1))
	if err != nil {
		return nil, false, err
	}
	if len(raw) > 64<<10 {
		return nil, false, fmt.Errorf("%w: file exceeds 64 KiB", ErrInvalidSettings)
	}
	return raw, true, nil
}

// Read returns the latest persisted state without changing the file or startup configuration.
func (s *SettingsStore) Read(ctx context.Context) (SettingsSnapshot, error) {
	if err := acquireSettingsGate(ctx); err != nil {
		return SettingsSnapshot{}, err
	}
	defer releaseSettingsGate()
	// Honor an existing cooperative writer lock without requiring writes on readonly volumes.
	lk, err := lockSettingsSharedIfExists(ctx, s.path)
	if err != nil {
		return SettingsSnapshot{}, err
	}
	if lk != nil {
		defer unlockSettings(lk)
	}
	raw, present, err := readRawSettings(s.path)
	if err != nil {
		return SettingsSnapshot{}, err
	}
	return s.makeSnapshot(raw, present, &envReader{getenv: s.getenv})
}

func acquireSettingsGate(ctx context.Context) error {
	// Honor an already-canceled request even when the lock is immediately available.
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case settingsGate <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func releaseSettingsGate() { <-settingsGate }

// lockSettings serializes writers that cooperate on the stable sibling lock file.
func lockSettings(ctx context.Context, path string) (*os.File, error) {
	fd, err := syscall.Open(path+".lock", syscall.O_CREAT|syscall.O_RDWR|syscall.O_NOFOLLOW|syscall.O_CLOEXEC|syscall.O_NONBLOCK, 0600)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), path+".lock")
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		f.Close()
		if err != nil {
			return nil, err
		}
		return nil, ErrSettingsPath
	}
	for {
		err = syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			return f, nil
		}
		if err != syscall.EWOULDBLOCK && err != syscall.EAGAIN {
			f.Close()
			return nil, err
		}
		select {
		case <-ctx.Done():
			f.Close()
			return nil, ctx.Err()
		case <-time.After(25 * time.Millisecond):
		}
	}
}

// lockSettingsSharedIfExists coordinates readers without creating a lock on read-only volumes.
func lockSettingsSharedIfExists(ctx context.Context, path string) (*os.File, error) {
	fd, err := syscall.Open(path+".lock", syscall.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC|syscall.O_NONBLOCK, 0)
	if errors.Is(err, syscall.ENOENT) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), path+".lock")
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		f.Close()
		if err != nil {
			return nil, err
		}
		return nil, ErrSettingsPath
	}
	for {
		err = syscall.Flock(fd, syscall.LOCK_SH|syscall.LOCK_NB)
		if err == nil {
			return f, nil
		}
		if err != syscall.EWOULDBLOCK && err != syscall.EAGAIN {
			f.Close()
			return nil, err
		}
		select {
		case <-ctx.Done():
			f.Close()
			return nil, ctx.Err()
		case <-time.After(25 * time.Millisecond):
		}
	}
}

func unlockSettings(f *os.File) { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN); _ = f.Close() }

// writableSettingsInfo rejects symlinks, special files and explicitly read-only files.
func writableSettingsInfo(path string) (os.FileInfo, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, ErrSettingsPath
	}
	if info.Mode().Perm()&0222 == 0 {
		return nil, ErrSettingsReadOnly
	}
	return info, nil
}

// Save validates a partial patch and atomically publishes it only for the expected revision.
// Keys in updates are the ten KOITO_* constants; no path or _FILE key is accepted.
func (s *SettingsStore) Save(ctx context.Context, expectedRevision string, updates map[string]string) (SettingsSnapshot, error) {
	// Bound direct service calls before constructing an edited document.
	total := 0
	for k, value := range updates {
		if _, ok := settingFields[k]; !ok {
			return SettingsSnapshot{}, fmt.Errorf("%w: unsupported field %s", ErrInvalidSettings, k)
		}
		total += len(value)
		if total > 64<<10 {
			return SettingsSnapshot{}, fmt.Errorf("%w: values exceed 64 KiB", ErrInvalidSettings)
		}
	}
	if err := acquireSettingsGate(ctx); err != nil {
		return SettingsSnapshot{}, err
	}
	defer releaseSettingsGate()
	lk, err := lockSettings(ctx, s.path)
	if err != nil {
		return SettingsSnapshot{}, err
	}
	defer unlockSettings(lk)
	info, err := writableSettingsInfo(s.path)
	if err != nil {
		return SettingsSnapshot{}, err
	}
	raw, present, err := readRawSettings(s.path)
	if err != nil {
		return SettingsSnapshot{}, err
	}
	if settingsRevision(raw, present) != expectedRevision {
		return SettingsSnapshot{}, ErrSettingsConflict
	}
	reader := &envReader{getenv: s.getenv}
	// Environment-controlled fields cannot be modified through the settings store.
	for name := range updates {
		_, origin := reader.get(name)
		if reader.err != nil {
			return SettingsSnapshot{}, reader.err
		}
		if origin == OriginEnvironment || origin == OriginFile {
			return SettingsSnapshot{}, ErrSettingsOverridden
		}
	}
	// Validate the candidate so an edit can repair an invalid current override.
	if len(updates) == 0 {
		return s.makeSnapshot(raw, present, reader)
	}
	candidate, err := editSettingsDocument(raw, updates)
	if err != nil {
		return SettingsSnapshot{}, err
	}
	parsed, err := decodeTOML(candidate)
	if err != nil {
		return SettingsSnapshot{}, fmt.Errorf("%w: %v", ErrInvalidSettings, err)
	}
	// Reparse and compare every field so unsupported layouts never lose semantic data.
	expected := map[string]string{}
	if present {
		original, e := decodeTOML(raw)
		if e != nil {
			return SettingsSnapshot{}, fmt.Errorf("%w: %v", ErrInvalidSettings, e)
		}
		expected = documentValues(original)
	}
	for k, v := range updates {
		// Integers are serialized canonically before comparing parsed values.
		if settingFields[k].kind == "int" {
			n, err := strconv.Atoi(v)
			if err != nil {
				return SettingsSnapshot{}, ErrInvalidSettings
			}
			v = strconv.Itoa(n)
		}
		expected[k] = v
	}
	if !reflect.DeepEqual(expected, documentValues(parsed)) {
		return SettingsSnapshot{}, ErrUnsupportedSettingsFormat
	}
	candidateState, err := s.makeSnapshot(candidate, true, reader)
	if err != nil {
		return SettingsSnapshot{}, err
	}
	if err = ctx.Err(); err != nil {
		return SettingsSnapshot{}, err
	}
	dir := filepath.Dir(s.path)
	tmp, err := os.CreateTemp(dir, ".conf.toml-*.tmp")
	if err != nil {
		return SettingsSnapshot{}, err
	}
	tempPath := tmp.Name()
	published := false
	defer func() {
		_ = tmp.Close()
		if !published {
			_ = os.Remove(tempPath)
		}
	}()
	if info != nil {
		// Preserve owner/group when a replacement would otherwise change them.
		current, statErr := tmp.Stat()
		if statErr != nil {
			return SettingsSnapshot{}, statErr
		}
		oldStat, oldOK := info.Sys().(*syscall.Stat_t)
		newStat, newOK := current.Sys().(*syscall.Stat_t)
		if !oldOK || !newOK {
			return SettingsSnapshot{}, fmt.Errorf("cannot verify configuration ownership")
		}
		if oldStat.Uid != newStat.Uid || oldStat.Gid != newStat.Gid {
			if err = tmp.Chown(int(oldStat.Uid), int(oldStat.Gid)); err != nil {
				return SettingsSnapshot{}, err
			}
		}
		if err = tmp.Chmod(info.Mode().Perm()); err != nil {
			return SettingsSnapshot{}, err
		}
	}
	if _, err = tmp.Write(candidate); err != nil {
		return SettingsSnapshot{}, err
	}
	if s.syncFile != nil {
		err = s.syncFile(tmp)
	} else {
		err = tmp.Sync()
	}
	if err != nil {
		return SettingsSnapshot{}, err
	}
	if err = tmp.Close(); err != nil {
		return SettingsSnapshot{}, err
	}
	if s.beforePublish != nil {
		if err = s.beforePublish(); err != nil {
			return SettingsSnapshot{}, err
		}
	}
	if err = ctx.Err(); err != nil {
		return SettingsSnapshot{}, err
	}
	currentInfo, err := writableSettingsInfo(s.path)
	if err != nil {
		return SettingsSnapshot{}, err
	}
	currentRaw, currentPresent, err := readRawSettings(s.path)
	if err != nil {
		return SettingsSnapshot{}, err
	}
	if settingsRevision(currentRaw, currentPresent) != expectedRevision {
		return SettingsSnapshot{}, ErrSettingsConflict
	}
	if (info == nil) != (currentInfo == nil) || (info != nil && (!os.SameFile(info, currentInfo) || info.Mode().Perm() != currentInfo.Mode().Perm())) {
		return SettingsSnapshot{}, ErrSettingsConflict
	}
	if info != nil {
		// Do not reverse an external owner/group change observed before publication.
		previous, ok1 := info.Sys().(*syscall.Stat_t)
		current, ok2 := currentInfo.Sys().(*syscall.Stat_t)
		if !ok1 || !ok2 || previous.Uid != current.Uid || previous.Gid != current.Gid {
			return SettingsSnapshot{}, ErrSettingsConflict
		}
	}
	// Recheck sources immediately before publication with a fresh environment snapshot.
	latestSources := &envReader{getenv: s.getenv}
	for name := range updates {
		_, origin := latestSources.get(name)
		if latestSources.err != nil {
			return SettingsSnapshot{}, latestSources.err
		}
		if origin == OriginEnvironment || origin == OriginFile {
			return SettingsSnapshot{}, ErrSettingsOverridden
		}
	}
	if s.rename != nil {
		err = s.rename(tempPath, s.path)
	} else {
		err = os.Rename(tempPath, s.path)
	}
	if err != nil {
		return SettingsSnapshot{}, err
	}
	published = true
	if s.syncDir != nil {
		err = s.syncDir(dir)
	} else {
		err = syncSettingsDir(dir)
	}
	if err != nil {
		// Publication has occurred: reread if possible and report uncertain durability.
		latest, exists, readErr := readRawSettings(s.path)
		if readErr == nil {
			if observed, stateErr := s.makeSnapshot(latest, exists, &envReader{getenv: s.getenv}); stateErr == nil {
				candidateState = observed
			}
		}
		return candidateState, &DurabilityError{Cause: err}
	}
	return candidateState, nil
}

// syncSettingsDir persists the directory entry after atomic replacement.
func syncSettingsDir(dir string) error {
	f, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer f.Close()
	return f.Sync()
}

// SettingsPath is intended for diagnostics, not for accepting user-selected paths.
func (s *SettingsStore) SettingsPath() string { return strings.Clone(s.path) }
