package utils

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

// rename is replaceable in tests to simulate a cross-device move.
var rename = os.Rename

// MoveFile renames src to dst and falls back to copy-then-remove when they are on
// different filesystems (EXDEV), e.g. import/ and import_complete/ on separate volumes.
// Only regular files are copied; the copy is synced and renamed into place atomically.
func MoveFile(src, dst string) error {
	err := rename(src, dst)
	if err == nil || !errors.Is(err, syscall.EXDEV) {
		return err
	}
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("move %s: not a regular file", src)
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp, err := os.CreateTemp(filepath.Dir(dst), ".move-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := func(err error) error {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if _, err := io.Copy(tmp, in); err != nil {
		return cleanup(err)
	}
	if err := tmp.Chmod(info.Mode().Perm()); err != nil {
		return cleanup(err)
	}
	if err := tmp.Sync(); err != nil {
		return cleanup(err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, dst); err != nil {
		os.Remove(tmpName)
		return err
	}
	return os.Remove(src)
}
