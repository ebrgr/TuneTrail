package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestTOMLStartupAndReset verifies the real command uses the same _FILE directory for both operations.
func TestTOMLStartupAndReset(t *testing.T) {
	if action := os.Getenv("TUNETRAIL_TEST_039_ACTION"); action != "" {
		os.Args = []string{"tunetrail"}
		if action == "reset" {
			os.Args = append(os.Args, "reset-password")
		}
		main()
		return
	}
	root := t.TempDir()
	dir := filepath.Join(root, "data")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	conf := `schema_version=1
[musicbrainz]
disabled=true
[images]
disable_deezer=true
disable_cover_art_archive=true
[time]
force_tz="America/Sao_Paulo"
`
	if err := os.WriteFile(filepath.Join(dir, "conf.toml"), []byte(conf), 0600); err != nil {
		t.Fatal(err)
	}
	dirFile := filepath.Join(root, "dir")
	if err := os.WriteFile(dirFile, []byte(dir), 0600); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	env := []string{}
	for _, v := range os.Environ() {
		if !strings.HasPrefix(v, "KOITO_") && !strings.HasPrefix(v, "TUNETRAIL_TEST_039_ACTION=") {
			env = append(env, v)
		}
	}
	env = append(env, "KOITO_CONFIG_DIR_FILE="+dirFile, "KOITO_BIND_ADDR=127.0.0.1", fmt.Sprintf("KOITO_LISTEN_PORT=%d", port), "KOITO_DEFAULT_PASSWORD=fixture-only", "KOITO_SKIP_IMPORT=true")
	command := func(action string) *exec.Cmd {
		cmd := exec.Command(os.Args[0], "-test.run=^TestTOMLStartupAndReset$")
		cmd.Env = append(append([]string{}, env...), "TUNETRAIL_TEST_039_ACTION="+action)
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
		return cmd
	}
	server := command("run")
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	stopped := false
	defer func() {
		if !stopped {
			server.Process.Kill()
			server.Wait()
		}
	}()
	client := &http.Client{Timeout: time.Second}
	healthy := false
	for deadline := time.Now().Add(15 * time.Second); time.Now().Before(deadline); time.Sleep(100 * time.Millisecond) {
		resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/apis/web/v1/health", port))
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				healthy = true
				break
			}
		}
	}
	if !healthy {
		t.Fatal("server did not start with TOML")
	}
	if _, err := os.Stat(filepath.Join(dir, "tunetrail.db")); err != nil {
		t.Fatal("expected database not created in configured directory")
	}
	// Terminate the disposable server before testing a separate reset process.
	if err := server.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = server.Wait()
	stopped = true
	if err := command("reset").Run(); err != nil {
		t.Fatal("reset-password failed on the existing TOML database")
	}
	invalidDir := filepath.Join(root, "invalid")
	if err := os.Mkdir(invalidDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(invalidDir, "conf.toml"), []byte("schema_version=999\n"), 0600); err != nil {
		t.Fatal(err)
	}
	invalid := command("run")
	invalid.Env = append(invalid.Env, "KOITO_CONFIG_DIR="+invalidDir)
	if err := invalid.Run(); err == nil {
		t.Fatal("invalid schema unexpectedly started")
	}
	if _, err := os.Stat(filepath.Join(invalidDir, "tunetrail.db")); !os.IsNotExist(err) {
		t.Fatal("invalid schema must not create a database")
	}
	// Corrupt and recover the same existing installation without touching its database.
	database := filepath.Join(dir, "tunetrail.db")
	beforeDB, err := os.ReadFile(database)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "conf.toml"), []byte("schema_version=999\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := command("run").Run(); err == nil {
		t.Fatal("invalid existing installation unexpectedly started")
	}
	afterDB, err := os.ReadFile(database)
	if err != nil || string(beforeDB) != string(afterDB) {
		t.Fatal("invalid config changed the existing database")
	}
	if err := os.WriteFile(filepath.Join(dir, "conf.toml"), []byte(conf), 0600); err != nil {
		t.Fatal(err)
	}
	server = command("run")
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	stopped = false
	healthy = false
	for deadline := time.Now().Add(15 * time.Second); time.Now().Before(deadline); time.Sleep(100 * time.Millisecond) {
		resp, e := client.Get(fmt.Sprintf("http://127.0.0.1:%d/apis/web/v1/health", port))
		if e == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				healthy = true
				break
			}
		}
	}
	if !healthy {
		t.Fatal("corrected configuration did not recover the existing installation")
	}
	got, err := os.ReadFile(filepath.Join(dir, "conf.toml"))
	if err != nil || string(got) != conf {
		t.Fatal("config was modified")
	}
}
