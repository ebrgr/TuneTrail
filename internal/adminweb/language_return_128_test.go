package adminweb

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSafeLanguageReturn128 keeps the sign-in document as a language-switch return and nothing wider.
func TestSafeLanguageReturn128(t *testing.T) {
	for in, want := range map[string]string{
		"/login":                         "/login",
		"/login?changed=1":               "/login?changed=1",
		"/login?redeemed=1":              "/login?redeemed=1",
		"/login?next=%2Faccount%2Fkeys":  "/login?next=%2Faccount%2Fkeys",
		"/second":                        "/second",
		"/account":                       "/account",
		"/login?next=https%3A%2F%2Fevil": "",
		"/login?next=%2Flogin":           "",
		"/login?next=%2Fa&next=%2Fb":     "",
		"/login?other=1":                 "",
		"/login?changed=2":               "",
		"/login/extra":                   "",
		"//evil.test/login":              "",
		"https://evil.test/login":        "",
		"/login#fragment":                "",
		"":                               "",
	} {
		require.Equal(t, want, SafeLanguageReturn(in), in)
	}
	// The post-login rule is unchanged: /login is never a destination.
	require.Empty(t, SafeReturn("/login"))
}
