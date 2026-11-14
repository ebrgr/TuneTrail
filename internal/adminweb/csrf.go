package adminweb

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/google/uuid"
)

// preauthCookie carries the login nonce for at most formTTL; renamed directly in
// ISSUE-111 because a restart already invalidates every nonce (process-local key).
const preauthCookie = "tunetrail_preauth"
const formTTL = 30 * time.Minute

// CSRF owns a process-local key: forms expire after a restart in this single-instance app.
type CSRF struct {
	key [32]byte
	now func() time.Time
}

// NewCSRF fails closed if cryptographic randomness is unavailable.
func NewCSRF() (*CSRF, error) {
	c := &CSRF{now: time.Now}
	if _, err := rand.Read(c.key[:]); err != nil {
		return nil, err
	}
	return c, nil
}

// seal uses unambiguous length-prefixed purpose/binding with a versioned expiration.
func (c *CSRF) seal(purpose, binding string, expires int64) string {
	prefix := "v1." + strconv.FormatInt(expires, 10)
	mac := hmac.New(sha256.New, c.key[:])
	fmt.Fprintf(mac, "%d:%s%d:%s%s", len(purpose), purpose, len(binding), binding, prefix)
	return prefix + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// valid accepts only current, bounded tokens with a constant-time MAC comparison.
func (c *CSRF) valid(token, purpose, binding string) bool {
	if binding == "" || len(token) > 512 {
		return false
	}
	p := strings.Split(token, ".")
	if len(p) != 3 || p[0] != "v1" {
		return false
	}
	expires, err := strconv.ParseInt(p[1], 10, 64)
	now := c.now().Unix()
	if err != nil || expires <= now || expires > now+int64(formTTL/time.Second) {
		return false
	}
	actual, err := base64.RawURLEncoding.DecodeString(p[2])
	if err != nil {
		return false
	}
	expected := strings.Split(c.seal(purpose, binding, expires), ".")[2]
	want, _ := base64.RawURLEncoding.DecodeString(expected)
	return hmac.Equal(actual, want)
}

// sessionBinding requires an already authenticated actor supplied by the adapter.
func sessionBinding(identity admin.Identity) (string, error) {
	if err := admin.Authorize(identity.User); err != nil {
		return "", err
	}
	if identity.SessionID == uuid.Nil {
		return "", admin.ErrUnauthorized
	}
	return identity.SessionID.String(), nil
}

// SessionToken is reusable until expiry; it does not make a mutation idempotent.
func (c *CSRF) SessionToken(identity admin.Identity) (string, error) {
	binding, err := sessionBinding(identity)
	if err != nil {
		return "", err
	}
	return c.seal("session-form", binding, c.now().Add(formTTL).Unix()), nil
}

// oneCookie rejects ambiguous duplicate cookies rather than choosing a binding arbitrarily.
func oneCookie(r *http.Request, name string) (string, bool) {
	v := ""
	n := 0
	for _, cookie := range r.Cookies() {
		if cookie.Name == name {
			v = cookie.Value
			n++
		}
	}
	return v, n == 1
}

// loginBinding authenticates the signed nonce cookie independently of any form field.
func (c *CSRF) loginBinding(r *http.Request) (string, int64, bool) {
	value, ok := oneCookie(r, preauthCookie)
	parts := strings.SplitN(value, ".", 2)
	if !ok || len(parts) != 2 {
		return "", 0, false
	}
	nonce, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil || len(nonce) != 32 || !c.valid(parts[1], "preauth-cookie", parts[0]) {
		return "", 0, false
	}
	expiry, _ := strconv.ParseInt(strings.Split(parts[1], ".")[1], 10, 64)
	return value, expiry, true
}

// LoginToken creates a signed preauth cookie only when no live nonce exists; GET never extends it.
func (c *CSRF) LoginToken(w http.ResponseWriter, r *http.Request) (string, error) {
	binding, expiry, ok := c.loginBinding(r)
	if !ok {
		var nonce [32]byte
		if _, err := rand.Read(nonce[:]); err != nil {
			return "", err
		}
		n := base64.RawURLEncoding.EncodeToString(nonce[:])
		expiry = c.now().Add(formTTL).Unix()
		binding = n + "." + c.seal("preauth-cookie", n, expiry)
		http.SetCookie(w, &http.Cookie{Name: preauthCookie, Value: binding, Path: "/", Expires: time.Unix(expiry, 0), MaxAge: int(formTTL / time.Second), HttpOnly: true, Secure: SecureRequest(r), SameSite: http.SameSiteStrictMode})
	}
	w.Header().Set("Cache-Control", "no-store")
	return c.seal("login-form", binding, expiry), nil
}

// ClearPreauth removes a login nonce only when one was actually present.
func ClearPreauth(w http.ResponseWriter, r *http.Request) {
	for _, cookie := range r.Cookies() {
		if cookie.Name == preauthCookie {
			http.SetCookie(w, &http.Cookie{Name: preauthCookie, Path: "/", MaxAge: -1, HttpOnly: true, Secure: SecureRequest(r), SameSite: http.SameSiteStrictMode})
			return
		}
	}
}
