package webassets

import "testing"

// TestAccountTabs126 checks the tab addresses of both modes: the own account never carries
// an ID, the administration context always does, and legacy sections map to a tab.
func TestAccountTabs126(t *testing.T) {
	self := AccountTabs(true, 7, TabKeys)
	want := []string{"/account", "/account/profile", "/account/keys", "/account/history", "/account/relay"}
	if len(self) != len(want) {
		t.Fatalf("got %d tabs, want %d", len(self), len(want))
	}
	for i, tab := range self {
		if tab.URL != want[i] {
			t.Errorf("self tab %d = %q, want %q", i, tab.URL, want[i])
		}
		if tab.Current != (want[i] == "/account/keys") {
			t.Errorf("self tab %q current = %v", tab.URL, tab.Current)
		}
	}
	context := AccountTabs(false, 7, TabProfile)
	wantContext := []string{"/admin?section=account&profile=7", "/admin?profile=7", "/admin?section=keys&profile=7",
		"/admin?section=import&profile=7", "/admin?section=relay&profile=7"}
	for i, tab := range context {
		if tab.URL != wantContext[i] {
			t.Errorf("context tab %d = %q, want %q", i, tab.URL, wantContext[i])
		}
	}
	for section, tab := range map[string]string{"": TabProfile, "profiles": TabProfile, "account": TabAccount, "keys": TabKeys,
		"import": TabHistory, "export": TabHistory, "relay": TabRelay} {
		if got, ok := TabFromSection(section); !ok || got != tab {
			t.Errorf("TabFromSection(%q) = %q, %v", section, got, ok)
		}
	}
	if _, ok := TabFromSection("settings"); ok {
		t.Error("an unknown section must not map to a tab")
	}
}
