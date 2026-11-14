package engine

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// instanceLock is held for the whole server lifetime; keeping the *os.File
// referenced prevents the garbage collector from closing it (ISSUE-111).
var instanceLock *os.File

// errInstanceRunning means another server already uses the data directory.
var errInstanceRunning = errors.New("another TuneTrail server is already using this data directory")

// acquireInstanceLock takes an exclusive, non-blocking lock on the data
// directory so two servers (for example the old koito-sum compose project and
// the new tunetrail one) never write to the same SQLite database. The
// reset-password command does not take it and keeps working beside a server.
func acquireInstanceLock(dir string) (*os.File, error) {
	path := filepath.Join(dir, ".tunetrail-instance.lock")
	fd, err := syscall.Open(path, syscall.O_CREAT|syscall.O_RDWR|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return nil, fmt.Errorf("open instance lock: %w", err)
	}
	file := os.NewFile(uintptr(fd), path)
	if err := syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		file.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, fmt.Errorf("%w (%s); stop the other container first", errInstanceRunning, dir)
		}
		return nil, fmt.Errorf("lock data directory: %w", err)
	}
	return file, nil
}
