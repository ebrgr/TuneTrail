// Command issue133_fault injects a late Goose failure into a synthetic rehearsal copy.
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing/fstest"

	migrations "github.com/ebrgr/TuneTrail/db/migrations_sqlite"
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

const injection = "late_synthetic_failure"

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: issue133_fault /tmp/tt133-rehearsal-*/failure/tunetrail.db")
		os.Exit(2)
	}
	if err := run(context.Background(), os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, path string) error {
	// Never open a caller-supplied database outside the synthetic fixture tree.
	path, err := filepath.EvalSymlinks(path)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	root := filepath.Dir(dir)
	physicalTemp, err := filepath.EvalSymlinks("/tmp")
	if err != nil {
		return err
	}
	if filepath.Dir(root) != physicalTemp || !strings.HasPrefix(filepath.Base(root), "tt133-rehearsal-") ||
		filepath.Base(dir) != "failure" || filepath.Base(path) != "tunetrail.db" {
		return errors.New("expected a temporary synthetic rehearsal copy")
	}

	uri := (&url.URL{Scheme: "file", Path: path}).String() + "?mode=rw&_pragma=foreign_keys(ON)"
	conn, err := sql.Open("sqlite", uri)
	if err != nil {
		return err
	}
	defer conn.Close()
	conn.SetMaxOpenConns(1)
	names, err := fs.Glob(migrations.Files, "*.sql")
	if err != nil {
		return err
	}
	files := fstest.MapFS{}
	for _, name := range names {
		content, err := migrations.Files.ReadFile(name)
		if err != nil {
			return err
		}
		if strings.HasPrefix(name, "008_") {
			const original = "DROP TABLE identity_assertion;"
			if !strings.Contains(string(content), original) {
				return errors.New("could not find the end of migration 008")
			}
			content = []byte(strings.Replace(string(content), original,
				"CREATE TEMP TABLE fault_probe (ok INTEGER CONSTRAINT late_synthetic_failure CHECK(ok=1)); INSERT INTO fault_probe VALUES (0);", 1))
		}
		files[name] = &fstest.MapFile{Data: content}
	}

	provider, err := goose.NewProvider(goose.DialectSQLite3, conn, files)
	if err != nil {
		return err
	}
	_, err = provider.Up(ctx)
	if err == nil || !strings.Contains(err.Error(), injection) || !strings.Contains(err.Error(), "CHECK") {
		return fmt.Errorf("expected the late injected CHECK failure, got %v", err)
	}
	var version int
	if err := conn.QueryRowContext(ctx, "SELECT max(version_id) FROM goose_db_version WHERE is_applied=1").Scan(&version); err != nil {
		return err
	}
	if version != 7 {
		return fmt.Errorf("failed migration recorded schema version %d", version)
	}
	return nil
}
