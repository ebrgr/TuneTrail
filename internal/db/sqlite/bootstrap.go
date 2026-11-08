package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	migrations "github.com/ebrgr/TuneTrail/db/migrations_sqlite"
	"github.com/pressly/goose/v3"
)

// regularFile rejects links and special files before opening database components.
func regularFile(path string) (bool, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !info.Mode().IsRegular() {
		return false, fmt.Errorf("unsafe database path %s", path)
	}
	return true, nil
}

// legacyDigest detects changes to the original database and its committed WAL.
func legacyDigest(path string) (string, error) {
	h := sha256.New()
	for _, name := range []string{path, path + "-wal"} {
		present, err := regularFile(name)
		if err != nil {
			return "", err
		}
		// SQLite may create an empty WAL while opening a clean database.
		if present && name == path+"-wal" {
			info, err := os.Stat(name)
			if err != nil {
				return "", err
			}
			if info.Size() == 0 {
				present = false
			}
		}
		fmt.Fprintf(h, "%s:%t:", filepath.Base(name), present)
		if present {
			f, err := os.Open(name)
			if err != nil {
				return "", err
			}
			_, err = io.Copy(h, f)
			closeErr := f.Close()
			if err != nil {
				return "", err
			}
			if closeErr != nil {
				return "", closeErr
			}
		}
	}
	for _, name := range []string{path + "-shm", path + "-journal"} {
		if _, err := regularFile(name); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// tempDatabase reserves a private SQLite snapshot path in the data directory.
func tempDatabase(dir, pattern string) (string, error) {
	f, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return "", err
	}
	name := f.Name()
	if err := f.Close(); err != nil {
		os.Remove(name)
		return "", err
	}
	return name, nil
}

// sqliteURI escapes special characters in an absolute filesystem path.
func sqliteURI(path string) string { return (&url.URL{Scheme: "file", Path: path}).String() }

// openExisting opens an existing regular database without implicit creation.
func openExisting(path string) (*sql.DB, error) {
	present, err := regularFile(path)
	if err != nil {
		return nil, err
	}
	if !present {
		return nil, fmt.Errorf("database missing: %s", path)
	}
	// SQLite URI path escaping is needed for directories containing '?' or '#'.
	uri := sqliteURI(path) + "?mode=rw&_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", uri)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// schemaVersion checks the migration history without changing an existing database.
func schemaVersion(db *sql.DB) (int, error) {
	var version int
	if err := db.QueryRow(`SELECT COALESCE(MAX(version_id),0) FROM goose_db_version WHERE is_applied=1`).Scan(&version); err != nil {
		return 0, fmt.Errorf("invalid migration history: %w", err)
	}
	if version < 1 || version > 3 {
		return 0, fmt.Errorf("unsupported schema version %d", version)
	}
	return version, nil
}

// checkIntegrity verifies SQLite pages and referential integrity.
func checkIntegrity(db *sql.DB) error {
	var result string
	if err := db.QueryRow(`PRAGMA integrity_check`).Scan(&result); err != nil || result != "ok" {
		return fmt.Errorf("SQLite integrity check failed: %s %v", result, err)
	}
	rows, err := db.Query(`PRAGMA foreign_key_check`)
	if err != nil {
		return err
	}
	defer rows.Close()
	if rows.Next() {
		return errors.New("SQLite foreign key check failed")
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return nil
}

// checkDatabase refuses corruption, broken references and incompatible v3 layouts.
func checkDatabase(db *sql.DB) error {
	if err := checkIntegrity(db); err != nil {
		return err
	}
	version, err := schemaVersion(db)
	if err != nil {
		return err
	}
	if version != 3 {
		return fmt.Errorf("schema version %d is not consolidated v3", version)
	}
	for _, q := range []string{`SELECT id,slug,display_name,avatar_image_id,avatar_icon,created_at FROM listening_profiles LIMIT 0`, `SELECT user_id,profile_id FROM legacy_user_profiles LIMIT 0`, `SELECT profile_id FROM listens LIMIT 0`, `SELECT profile_id FROM api_keys LIMIT 0`, `SELECT profile_id,url,token FROM profile_relays LIMIT 0`, `SELECT id,profile_id FROM legacy_relay_binding LIMIT 0`} {
		rows, err := db.Query(q)
		if err != nil {
			return fmt.Errorf("incompatible v3 schema: %w", err)
		}
		rows.Close()
	}
	// The consolidated profile schema depends on these safety triggers.
	for _, name := range []string{"trg_api_key_profile_immutable", "trg_legacy_user_profile"} {
		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='trigger' AND name=?`, name).Scan(&count); err != nil || count != 1 {
			return fmt.Errorf("incompatible v3 schema: required trigger %s missing", name)
		}
	}
	return nil
}

// bootstrap serializes selection and publishes only fully prepared databases.
func bootstrap(dir string, allowCreate bool) (string, error) {
	// Normalize relative configuration paths before constructing SQLite URIs.
	absolute, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	dir = absolute
	lockPath := filepath.Join(dir, ".tunetrail-db.lock")
	fd, err := syscall.Open(lockPath, syscall.O_CREAT|syscall.O_RDWR|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return "", err
	}
	lock := os.NewFile(uintptr(fd), lockPath)
	defer lock.Close()
	info, err := lock.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return "", errors.New("unsafe database lock")
	}
	if err := syscall.Flock(fd, syscall.LOCK_EX); err != nil {
		return "", err
	}
	defer syscall.Flock(fd, syscall.LOCK_UN)
	source := filepath.Join(dir, "koito.db")
	target := filepath.Join(dir, "tunetrail.db")
	old, err := regularFile(source)
	if err != nil {
		return "", err
	}
	current, err := regularFile(target)
	if err != nil {
		return "", err
	}
	for _, p := range []string{target + "-wal", target + "-shm", target + "-journal"} {
		exists, err := regularFile(p)
		if err != nil {
			return "", err
		}
		// Orphaned sidecars could be replayed onto a newly published database.
		if exists && !current {
			return "", fmt.Errorf("orphaned database sidecar %s; stop and inspect", p)
		}
	}
	if current {
		db, err := openExisting(target)
		if err != nil {
			return "", err
		}
		defer db.Close()
		version, err := schemaVersion(db)
		if err != nil {
			return "", err
		}
		if old {
			var stored, backup string
			err := db.QueryRow(`SELECT source_digest, backup_name FROM tunetrail_migration WHERE id=1`).Scan(&stored, &backup)
			if err != nil || filepath.Base(backup) != backup || !strings.HasPrefix(backup, "koito.db.pre-tunetrail-") {
				return "", errors.New("conflicting databases: cannot prove legacy migration; stop and inspect both databases")
			}
			if ok, err := regularFile(filepath.Join(dir, backup)); err != nil || !ok {
				return "", errors.New("legacy migration backup missing or unsafe")
			}
			digest, err := legacyDigest(source)
			if err != nil {
				return "", err
			}
			if digest != stored {
				return "", errors.New("legacy database changed after migration; stop and reconcile both databases")
			}
		}
		if version < 3 {
			backup, err := backupBeforeProfiles(context.Background(), db, target)
			if err != nil {
				return "", err
			}
			if backup == "" {
				return "", errors.New("existing v2 database did not produce a backup")
			}
			// Validate and synchronize the backup before mutating the existing database.
			saved, err := openExisting(backup)
			if err != nil {
				return "", err
			}
			err = checkIntegrity(saved)
			if err == nil {
				_, err = schemaVersion(saved)
			}
			closeErr := saved.Close()
			if err != nil {
				return "", fmt.Errorf("backup validation: %w", err)
			}
			if closeErr != nil {
				return "", closeErr
			}
			f, err := os.OpenFile(backup, os.O_RDWR, 0)
			if err != nil {
				return "", err
			}
			err = f.Sync()
			f.Close()
			if err != nil {
				return "", err
			}
			d, err := os.Open(dir)
			if err != nil {
				return "", err
			}
			err = d.Sync()
			d.Close()
			if err != nil {
				return "", err
			}
			goose.SetBaseFS(migrations.Files)
			if err := goose.SetDialect("sqlite3"); err != nil {
				return "", err
			}
			if err := goose.Up(db, "."); err != nil {
				return "", err
			}
		}
		if err := checkDatabase(db); err != nil {
			return "", err
		}
		return target, nil
	}
	if !old && !allowCreate {
		return "", errors.New("no database exists; refusing to create one for password reset")
	}
	var backup, digest string
	if old {
		before, err := legacyDigest(source)
		if err != nil {
			return "", err
		}
		// A read-only SQLite connection sees committed WAL without checkpointing it.
		uri := sqliteURI(source) + "?mode=ro&_pragma=busy_timeout(5000)"
		src, err := sql.Open("sqlite", uri)
		if err != nil {
			return "", err
		}
		src.SetMaxOpenConns(1)
		version, err := schemaVersion(src)
		if err != nil {
			src.Close()
			return "", err
		}
		if version != 2 && version != 3 {
			src.Close()
			return "", fmt.Errorf("unsupported legacy schema version %d", version)
		}
		backup, err = tempDatabase(dir, "koito.db.pre-tunetrail-*.bak")
		if err != nil {
			src.Close()
			return "", err
		}
		if _, err = src.Exec(`VACUUM INTO ?`, backup); err != nil {
			src.Close()
			os.Remove(backup)
			return "", err
		}
		if err = src.Close(); err != nil {
			return "", err
		}
		digest, err = legacyDigest(source)
		if err != nil {
			return "", err
		}
		if before != digest {
			return "", errors.New("legacy database changed during backup; stop the old service and retry")
		}
		f, err := os.OpenFile(backup, os.O_RDWR, 0)
		if err != nil {
			return "", err
		}
		err = f.Sync()
		f.Close()
		if err != nil {
			return "", err
		}
	}
	tmp, err := tempDatabase(dir, ".tunetrail-migrate-*.db")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp)
	if old {
		b, err := openExisting(backup)
		if err != nil {
			return "", err
		}
		// Verify the retained snapshot before using it as a restoration source.
		if err = checkIntegrity(b); err != nil {
			b.Close()
			return "", fmt.Errorf("backup validation: %w", err)
		}
		_, err = b.Exec(`VACUUM INTO ?`, tmp)
		b.Close()
		if err != nil {
			return "", err
		}
	}
	db, err := openExisting(tmp)
	if err != nil {
		return "", err
	}
	if old {
		v, e := schemaVersion(db)
		if e != nil {
			db.Close()
			return "", e
		}
		if v == 3 {
			if e = checkDatabase(db); e != nil {
				db.Close()
				return "", e
			}
		}
	}
	goose.SetBaseFS(migrations.Files)
	if err = goose.SetDialect("sqlite3"); err == nil {
		err = goose.Up(db, ".")
	}
	if err != nil {
		db.Close()
		return "", err
	}
	if old {
		_, err = db.Exec(`CREATE TABLE tunetrail_migration (id INTEGER PRIMARY KEY CHECK (id=1), source_digest TEXT NOT NULL, backup_name TEXT NOT NULL); INSERT INTO tunetrail_migration VALUES (1, ?, ?)`, digest, filepath.Base(backup))
		if err != nil {
			db.Close()
			return "", err
		}
	}
	if err = checkDatabase(db); err != nil {
		db.Close()
		return "", err
	}
	if err = db.Close(); err != nil {
		return "", err
	}
	f, err := os.OpenFile(tmp, os.O_RDWR, 0)
	if err != nil {
		return "", err
	}
	err = f.Sync()
	f.Close()
	if err != nil {
		return "", err
	}
	if err = os.Link(tmp, target); err != nil {
		return "", fmt.Errorf("publish database without overwrite: %w", err)
	}
	d, err := os.Open(dir)
	if err != nil {
		return "", err
	}
	err = d.Sync()
	d.Close()
	if err != nil {
		return "", err
	}
	return target, nil
}
