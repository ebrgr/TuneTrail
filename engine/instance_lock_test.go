package engine

import (
	"errors"
	"testing"
)

// TestInstanceLockRefusesSecondServer: a second server on the same directory
// must fail fast; a different directory and a released lock are fine.
func TestInstanceLockRefusesSecondServer(t *testing.T) {
	dir := t.TempDir()
	first, err := acquireInstanceLock(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := acquireInstanceLock(dir); !errors.Is(err, errInstanceRunning) {
		t.Fatalf("second lock: %v", err)
	}
	other, err := acquireInstanceLock(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	other.Close()
	first.Close()
	again, err := acquireInstanceLock(dir)
	if err != nil {
		t.Fatalf("lock after release: %v", err)
	}
	again.Close()
}
