package imagefetch

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// TestPublicAddress160 covers internal networks, metadata, IPv4 mappings and IPv6 transition bypasses.
func TestPublicAddress160(t *testing.T) {
	for _, raw := range []string{"127.0.0.1", "127.20.1.1", "10.0.0.1", "172.16.0.1", "172.31.255.255", "192.168.1.1", "169.254.169.254", "169.254.1.1", "0.0.0.0", "0.1.2.3", "100.100.100.200", "224.0.0.1", "::", "::1", "::ffff:127.0.0.1", "::ffff:10.0.0.1", "::127.0.0.1", "fc00::1", "fd00:ec2::254", "fe80::1", "fec0::1", "240.0.0.1", "2001:db8::1", "ff02::1", "64:ff9b::a00:1", "2002:7f00:1::1"} {
		t.Run(raw, func(t *testing.T) {
			if publicAddress(netip.MustParseAddr(raw)) {
				t.Fatal("accepted internal address")
			}
		})
	}
	for _, raw := range []string{"8.8.8.8", "1.1.1.1", "2606:4700:4700::1111"} {
		if !publicAddress(netip.MustParseAddr(raw)) {
			t.Fatalf("rejected public address %s", raw)
		}
	}
}

// TestImportedImageLocalBlock160 proves persisted provenance reaches Download while root sources stay compatible.
func TestImportedImageLocalBlock160(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1); _, _ = w.Write([]byte("test image")) }))
	defer server.Close()
	source := PublicSource(server.URL)
	if _, err := Download(context.Background(), source); err == nil {
		t.Fatal("accepted imported loopback source")
	}
	if hits.Load() != 0 {
		t.Fatal("restricted request reached internal server")
	}
	if _, err := Download(context.Background(), server.URL); err != nil {
		t.Fatal(err)
	}
	if hits.Load() != 1 {
		t.Fatal("root local source no longer works")
	}
	for _, raw := range []string{"ftp://example.com/a", "file:///etc/passwd", "http://user:password@example.com/a"} {
		if PublicSource(raw) != "" {
			t.Fatalf("accepted %s", raw)
		}
	}
	if PublicSource(source) != source {
		t.Fatal("provenance is not stable across export/reimport")
	}
}

type peerConn struct {
	net.Conn
	remote net.Addr
}

func (c peerConn) RemoteAddr() net.Addr { return c.remote }

// TestPublicDNSAndConnectedIP160 checks private DNS answers, mixed answers, IP pinning and the actual peer.
func TestPublicDNSAndConnectedIP160(t *testing.T) {
	for _, answers := range [][]netip.Addr{{netip.MustParseAddr("10.1.2.3")}, {netip.MustParseAddr("8.8.8.8"), netip.MustParseAddr("127.0.0.1")}} {
		dialed := false
		tr := publicTransport(func(context.Context, string, string) ([]netip.Addr, error) { return answers, nil }, func(context.Context, string, string) (net.Conn, error) {
			dialed = true
			return nil, errors.New("unexpected dial")
		})
		if _, err := tr.DialContext(context.Background(), "tcp", "images.example:80"); err == nil || dialed {
			t.Fatal("private DNS answer was dialed")
		}
	}
	for _, peer := range []string{"127.0.0.1", "1.1.1.1", "8.8.8.8"} {
		lookups := 0
		var destination string
		a, b := net.Pipe()
		tr := publicTransport(func(context.Context, string, string) ([]netip.Addr, error) {
			lookups++
			return []netip.Addr{netip.MustParseAddr("8.8.8.8")}, nil
		}, func(_ context.Context, _ string, address string) (net.Conn, error) {
			destination = address
			return peerConn{a, &net.TCPAddr{IP: net.ParseIP(peer), Port: 80}}, nil
		})
		conn, err := tr.DialContext(context.Background(), "tcp", "images.example:80")
		if (peer == "8.8.8.8") != (err == nil) {
			t.Fatalf("peer %s: %v", peer, err)
		}
		if lookups != 1 || destination != "8.8.8.8:80" {
			t.Fatal("destination was not pinned to its validated DNS address")
		}
		if conn != nil {
			conn.Close()
		}
		a.Close()
		b.Close()
	}
}

// TestPublicRedirects160 simulates a public first hop without depending on external DNS or the internet.
func TestPublicRedirects160(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/private-ip":
			http.Redirect(w, r, "http://127.0.0.1/secret", http.StatusFound)
		case "/private-dns":
			http.Redirect(w, r, "http://internal.example/secret", http.StatusFound)
		case "/loop":
			http.Redirect(w, r, "http://public.example/loop", http.StatusFound)
		case "/slow":
			<-r.Context().Done()
		default:
			_, _ = io.WriteString(w, "public image")
		}
	}))
	defer server.Close()
	serverURL, _ := url.Parse(server.URL)
	var dials atomic.Int32
	lookup := func(_ context.Context, _ string, host string) ([]netip.Addr, error) {
		if host == "public.example" {
			return []netip.Addr{netip.MustParseAddr("8.8.8.8")}, nil
		}
		return []netip.Addr{netip.MustParseAddr("127.0.0.1")}, nil
	}
	tr := publicTransport(lookup, func(ctx context.Context, network, address string) (net.Conn, error) {
		dials.Add(1)
		conn, err := (&net.Dialer{}).DialContext(ctx, network, serverURL.Host)
		if err != nil {
			return nil, err
		}
		return peerConn{conn, &net.TCPAddr{IP: net.ParseIP("8.8.8.8"), Port: 80}}, nil
	})
	defer tr.CloseIdleConnections()
	client := imageClient(true, tr)
	for _, path := range []string{"/private-ip", "/private-dns"} {
		before := dials.Load()
		if resp, err := client.Get("http://public.example" + path); err == nil {
			resp.Body.Close()
			t.Fatal("accepted private redirect")
		}
		if dials.Load() != before+1 {
			t.Fatal("redirect dialed a private destination")
		}
	}
	if resp, err := client.Get("http://public.example/loop"); err == nil {
		resp.Body.Close()
		t.Fatal("unbounded redirects")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://public.example/slow", nil)
	if resp, err := client.Do(req); err == nil {
		resp.Body.Close()
		t.Fatal("ignored timeout")
	}
	if client.Timeout != 10*time.Second {
		t.Fatal("missing total deadline")
	}
	resp, err := client.Get("http://public.example/image")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil || !strings.Contains(string(data), "public image") {
		t.Fatal("public image failed")
	}
}
