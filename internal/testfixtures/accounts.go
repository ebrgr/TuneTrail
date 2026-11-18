// Package testfixtures builds explicit account/profile pairs in disposable stores.
package testfixtures

import (
	"fmt"
	"testing"
)

// Profile seeds fixed identities for tests without bypassing ownership constraints.
func Profile(t testing.TB, store interface{ Exec(string, ...any) error }, id int, slug, name string, created int64) {
	t.Helper()
	if err := store.Exec(`INSERT INTO users(id,username,password,status) VALUES(?,?,X'','pending')`, id, fmt.Sprintf("fixture-%d", id)); err != nil {
		t.Fatal(err)
	}
	if err := store.Exec(`INSERT INTO listening_profiles(id,user_id,slug,display_name,created_at) VALUES(?,?,?,?,?)`, id, id, slug, name, created); err != nil {
		t.Fatal(err)
	}
}
