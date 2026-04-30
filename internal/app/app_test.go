package app_test

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"pg_gobench/internal/app"
	"pg_gobench/internal/config"
)

func TestParseConfig(t *testing.T) {
	t.Run("requires config path", func(t *testing.T) {
		_, err := app.ParseConfig(nil)
		if err == nil {
			t.Fatal("ParseConfig returned nil error without config path")
		}
		if !strings.Contains(err.Error(), "config") {
			t.Fatalf("ParseConfig error = %q, want mention of config", err)
		}
	})

	t.Run("accepts explicit bind address and minimal yaml config", func(t *testing.T) {
		configPath := writeConfigFile(t, `
source:
  host: localhost
  port: 5432
  sslmode: disable
  username:
    value: postgres
  password:
    value: secret
  dbname: postgres
`)

		cfg, err := app.ParseConfig([]string{"-addr", "127.0.0.1:9090", "-config", configPath})
		if err != nil {
			t.Fatalf("ParseConfig returned error: %v", err)
		}
		if cfg.Addr != "127.0.0.1:9090" {
			t.Fatalf("Addr = %q, want %q", cfg.Addr, "127.0.0.1:9090")
		}
	})

	t.Run("rejects empty bind address", func(t *testing.T) {
		configPath := writeConfigFile(t, `
source:
  host: localhost
  port: 5432
  sslmode: disable
  username:
    value: postgres
  password:
    value: secret
  dbname: postgres
`)

		_, err := app.ParseConfig([]string{"-addr", "", "-config", configPath})
		if err == nil {
			t.Fatal("ParseConfig returned nil error for empty bind address")
		}
		if !strings.Contains(err.Error(), "addr") {
			t.Fatalf("ParseConfig error = %q, want mention of addr", err)
		}
	})

	t.Run("rejects unknown flag", func(t *testing.T) {
		_, err := app.ParseConfig([]string{"-bogus"})
		if err == nil {
			t.Fatal("ParseConfig returned nil error for unknown flag")
		}
		if !strings.Contains(err.Error(), "flag provided but not defined") {
			t.Fatalf("ParseConfig error = %q, want unknown flag message", err)
		}
	})
}

func writeConfigFile(t *testing.T, contents string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(strings.TrimSpace(contents)), 0o600); err != nil {
		t.Fatalf("WriteFile config: %v", err)
	}

	return path
}

func TestRunServesHealthzAndShutsDownOnContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stdoutReader, stdoutWriter := io.Pipe()
	defer func() {
		if err := stdoutReader.Close(); err != nil {
			t.Fatalf("Close stdoutReader: %v", err)
		}
	}()

	errCh := make(chan error, 1)
	go func() {
		runErr := app.Run(ctx, app.Config{
			Addr:   "127.0.0.1:0",
			Source: testSourceConfig(),
		}, stdoutWriter, io.Discard)
		closeErr := stdoutWriter.Close()
		if runErr != nil {
			errCh <- runErr
			return
		}
		if closeErr != nil {
			errCh <- fmt.Errorf("close stdoutWriter: %w", closeErr)
			return
		}
		errCh <- nil
	}()

	addr := readListeningAddr(t, stdoutReader)
	response, err := http.Get("http://" + addr + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			t.Fatalf("Close response body: %v", err)
		}
	}()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", response.StatusCode, http.StatusOK)
	}

	var healthz map[string]string
	if err := json.NewDecoder(response.Body).Decode(&healthz); err != nil {
		t.Fatalf("Decode /healthz response body: %v", err)
	}
	if healthz["status"] != "ok" {
		t.Fatalf("healthz status = %q, want %q", healthz["status"], "ok")
	}

	readyzResponse, err := http.Get("http://" + addr + "/readyz")
	if err != nil {
		t.Fatalf("GET /readyz: %v", err)
	}
	defer func() {
		if err := readyzResponse.Body.Close(); err != nil {
			t.Fatalf("Close readyz response body: %v", err)
		}
	}()

	if readyzResponse.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("readyz StatusCode = %d, want %d", readyzResponse.StatusCode, http.StatusServiceUnavailable)
	}

	var readyz map[string]string
	if err := json.NewDecoder(readyzResponse.Body).Decode(&readyz); err != nil {
		t.Fatalf("Decode /readyz response body: %v", err)
	}
	if !strings.Contains(readyz["error"], "readiness ping") {
		t.Fatalf("readyz error = %q, want readiness ping context", readyz["error"])
	}

	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Run returned error after cancellation: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not exit after cancellation")
	}
}

func TestRunReturnsListenerError(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer func() {
		if err := listener.Close(); err != nil {
			t.Fatalf("Close listener: %v", err)
		}
	}()

	err = app.Run(context.Background(), app.Config{
		Addr:   listener.Addr().String(),
		Source: testSourceConfig(),
	}, io.Discard, io.Discard)
	if err == nil {
		t.Fatal("Run returned nil error for occupied address")
	}
	if !strings.Contains(err.Error(), "listen") {
		t.Fatalf("Run error = %q, want mention of listen", err)
	}
}

func readListeningAddr(t *testing.T, r io.Reader) string {
	t.Helper()

	lineCh := make(chan string, 1)
	errCh := make(chan error, 1)

	go func() {
		line, err := bufio.NewReader(r).ReadString('\n')
		if err != nil {
			errCh <- err
			return
		}
		lineCh <- strings.TrimSpace(strings.TrimPrefix(line, "listening on "))
	}()

	select {
	case line := <-lineCh:
		if line == "" {
			t.Fatal("read empty listening address")
		}
		return line
	case err := <-errCh:
		t.Fatalf("ReadString listening line: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for listening address")
	}

	return ""
}

func testSourceConfig() config.Source {
	return config.Source{
		Host:     "127.0.0.1",
		Port:     1,
		SSLMode:  config.SSLModeDisable,
		Username: "postgres",
		Password: "secret",
		DBName:   "postgres",
	}
}
