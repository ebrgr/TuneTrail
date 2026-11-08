// ui-preview runs only fictional, loopback presentation servers. No .env or DB.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/ebrgr/TuneTrail/internal/webui"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func loopback(addr string) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("preview requires a literal loopback IP: %s", addr)
	}
	return nil
}
func main() {
	addr := flag.String("listen", "127.0.0.1:14111", "Go preview loopback address")
	flag.Parse()
	if err := run(*addr); err != nil {
		log.Fatal(err)
	}
}

// run starts a single fictional Go preview on a loopback address.
func run(addr string) error {
	if err := loopback(addr); err != nil {
		return err
	}
	handler, err := webui.NewPreview()
	if err != nil {
		return err
	}
	servers := []*http.Server{{Addr: addr, Handler: handler, ReadHeaderTimeout: 5 * time.Second}}
	listeners := []net.Listener{}
	for _, s := range servers {
		l, err := net.Listen("tcp", s.Addr)
		if err != nil {
			for _, open := range listeners {
				_ = open.Close()
			}
			return err
		}
		listeners = append(listeners, l)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	errs := make(chan error, len(servers))
	for i, s := range servers {
		log.Printf("Fictional UI preview: http://%s (no database)", s.Addr)
		go func(s *http.Server, l net.Listener) { errs <- s.Serve(l) }(s, listeners[i])
	}
	select {
	case <-ctx.Done():
	case err = <-errs:
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	for _, s := range servers {
		_ = s.Shutdown(shutdown)
	}
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
