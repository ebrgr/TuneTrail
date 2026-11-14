package cfg

import (
	"net"
	"testing"
)

// TestListenAddrBracketsIPv6 guards Koito #74: an IPv6 bind address must be bracketed.
func TestListenAddrBracketsIPv6(t *testing.T) {
	lock.Lock()
	saved := globalConfig
	lock.Unlock()
	t.Cleanup(func() { lock.Lock(); globalConfig = saved; lock.Unlock() })

	for bind, want := range map[string]string{
		"":          ":4110",
		"0.0.0.0":   "0.0.0.0:4110",
		"127.0.0.1": "127.0.0.1:4110",
		"::":        "[::]:4110",
		"[::]":      "[::]:4110",
		"fd00::1":   "[fd00::1]:4110",
	} {
		lock.Lock()
		globalConfig = &config{bindAddr: bind, listenPort: 4110}
		lock.Unlock()
		got := ListenAddr()
		if got != want {
			t.Errorf("bind %q: ListenAddr %q, want %q", bind, got, want)
		}
		if _, _, err := net.SplitHostPort(got); err != nil {
			t.Errorf("bind %q: %v", bind, err)
		}
	}
}
