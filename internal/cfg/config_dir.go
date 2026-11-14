package cfg

// Default configuration and data directory (ISSUE-111). The default moved from
// /etc/koito to /etc/tunetrail. Containers that still mount data at /etc/koito
// must keep working until 2.0, and a fresh install must never write its database
// into an unmounted path while a volume sits at the old one.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	DefaultConfigDir = "/etc/tunetrail"
	LegacyConfigDir  = "/etc/koito"
)

// hasDatabase reports whether dir holds a TuneTrail or legacy Koito database.
func hasDatabase(dir string) bool {
	for _, name := range []string{"tunetrail.db", "koito.db"} {
		if info, err := os.Stat(filepath.Join(dir, name)); err == nil && info.Mode().IsRegular() {
			return true
		}
	}
	return false
}

func isDir(dir string) bool {
	info, err := os.Stat(dir)
	return err == nil && info.IsDir()
}

// errAmbiguousDataDir stops startup instead of guessing between two databases.
var errAmbiguousDataDir = errors.New("both default directories contain a database")

// resolveDefaultConfigDir chooses the directory when no *_CONFIG_DIR variable is
// set. It returns an optional notice for the startup log.
//
//	current has a database and legacy too  -> error (ambiguous)
//	current has a database                 -> current
//	legacy has a database                  -> legacy + notice
//	legacy exists, current does not exist  -> legacy + notice (a volume mounted
//	                                          at the old path on a fresh install)
//	otherwise                              -> current
func resolveDefaultConfigDir(current, legacy string) (dir, notice string, err error) {
	currentData, legacyData := hasDatabase(current), hasDatabase(legacy)
	switch {
	case currentData && legacyData:
		return "", "", fmt.Errorf("%w: %s and %s; set TUNETRAIL_CONFIG_DIR or remove one of them", errAmbiguousDataDir, current, legacy)
	case currentData:
		return current, "", nil
	case legacyData || (isDir(legacy) && !isDir(current)):
		return legacy, fmt.Sprintf("using the legacy data directory %s; mount it at %s instead (the old path works until 2.0)", legacy, current), nil
	}
	return current, "", nil
}
