package cfg

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestResolveDefaultConfigDir covers every row of the table in config_dir.go
// with temporary stand-ins for /etc/tunetrail and /etc/koito (ISSUE-111).
func TestResolveDefaultConfigDir(t *testing.T) {
	mk := func(t *testing.T, dir string, db string) {
		t.Helper()
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		if db != "" {
			if err := os.WriteFile(filepath.Join(dir, db), []byte("x"), 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, tc := range []struct {
		name              string
		current, legacy   string // "" = absent, "dir" = empty directory, otherwise a database file name
		want              string // "current" or "legacy"
		notice, ambiguous bool
	}{
		{"fresh install, nothing mounted", "", "", "current", false, false},
		{"new mount, empty", "dir", "", "current", false, false},
		{"new mount with data", "tunetrail.db", "", "current", false, false},
		{"new mount with a legacy koito.db", "koito.db", "", "current", false, false},
		{"old mount with data", "", "tunetrail.db", "legacy", true, false},
		{"old mount with a legacy koito.db", "", "koito.db", "legacy", true, false},
		{"old mount empty, fresh install", "", "dir", "legacy", true, false},
		{"both mounted, data only in the new one", "tunetrail.db", "dir", "current", false, false},
		{"both mounted, data only in the old one", "dir", "tunetrail.db", "legacy", true, false},
		{"both mounted and empty", "dir", "dir", "current", false, false},
		{"both contain a database", "tunetrail.db", "koito.db", "", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			current, legacy := filepath.Join(root, "tunetrail"), filepath.Join(root, "koito")
			for dir, state := range map[string]string{current: tc.current, legacy: tc.legacy} {
				switch state {
				case "":
				case "dir":
					mk(t, dir, "")
				default:
					mk(t, dir, state)
				}
			}
			dir, notice, err := resolveDefaultConfigDir(current, legacy)
			if tc.ambiguous {
				if !errors.Is(err, errAmbiguousDataDir) || !strings.Contains(err.Error(), "TUNETRAIL_CONFIG_DIR") {
					t.Fatalf("error %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			want := map[string]string{"current": current, "legacy": legacy}[tc.want]
			if dir != want || (notice != "") != tc.notice {
				t.Fatalf("dir %s notice %q", dir, notice)
			}
		})
	}
}
