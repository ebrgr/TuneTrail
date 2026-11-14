package utils

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMoveFileSameFilesystem(t *testing.T) {
	dir := t.TempDir()
	src, dst := filepath.Join(dir, "a.json"), filepath.Join(dir, "b.json")
	require.NoError(t, os.WriteFile(src, []byte("data"), 0600))
	require.NoError(t, MoveFile(src, dst))
	b, err := os.ReadFile(dst)
	require.NoError(t, err)
	require.Equal(t, "data", string(b))
	require.NoFileExists(t, src)
}

// Koito #185: import/ and import_complete/ mounted on different volumes.
func TestMoveFileCrossDeviceCopies(t *testing.T) {
	old := rename
	t.Cleanup(func() { rename = old })
	rename = func(string, string) error { return &os.LinkError{Op: "rename", Err: syscall.EXDEV} }

	src, dst := filepath.Join(t.TempDir(), "a.json"), filepath.Join(t.TempDir(), "b.json")
	require.NoError(t, os.WriteFile(src, []byte("listens"), 0640))
	require.NoError(t, MoveFile(src, dst))
	b, err := os.ReadFile(dst)
	require.NoError(t, err)
	require.Equal(t, "listens", string(b))
	info, err := os.Stat(dst)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0640), info.Mode().Perm())
	require.NoFileExists(t, src)
	leftovers, _ := filepath.Glob(filepath.Join(filepath.Dir(dst), ".move-*"))
	require.Empty(t, leftovers)
}

func TestMoveFileCrossDeviceRejectsSymlink(t *testing.T) {
	old := rename
	t.Cleanup(func() { rename = old })
	rename = func(string, string) error { return &os.LinkError{Op: "rename", Err: syscall.EXDEV} }

	dir := t.TempDir()
	target, src := filepath.Join(dir, "target"), filepath.Join(dir, "link")
	require.NoError(t, os.WriteFile(target, []byte("x"), 0600))
	require.NoError(t, os.Symlink(target, src))
	require.Error(t, MoveFile(src, filepath.Join(t.TempDir(), "b")))
	require.FileExists(t, target)
}

func TestMoveFileOtherErrorsPassThrough(t *testing.T) {
	err := MoveFile(filepath.Join(t.TempDir(), "missing"), filepath.Join(t.TempDir(), "b"))
	require.ErrorIs(t, err, os.ErrNotExist)
}
