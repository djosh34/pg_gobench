package config_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pg_gobench/internal/config"
)

func TestLoadResolvesEnvRefCredentials(t *testing.T) {
	t.Setenv("POSTGRES_USERNAME", "postgres")
	t.Setenv("POSTGRES_PASSWORD", "secret")

	path := writeConfigFile(t, `
source:
  host: localhost
  port: 5432
  username:
    env-ref: POSTGRES_USERNAME
  password:
    env-ref: POSTGRES_PASSWORD
  dbname: postgres
`)

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Source.Username != "postgres" {
		t.Fatalf("Username = %q, want %q", cfg.Source.Username, "postgres")
	}
	if cfg.Source.Password != "secret" {
		t.Fatalf("Password = %q, want %q", cfg.Source.Password, "secret")
	}
}

func TestLoadRejectsMissingOrEmptyEnvRefCredentials(t *testing.T) {
	t.Run("missing username env var", func(t *testing.T) {
		t.Setenv("POSTGRES_PASSWORD", "secret")

		path := writeConfigFile(t, `
source:
  host: localhost
  port: 5432
  username:
    env-ref: POSTGRES_USERNAME
  password:
    env-ref: POSTGRES_PASSWORD
  dbname: postgres
`)

		_, err := config.Load(path)
		if err == nil {
			t.Fatal("Load returned nil error for missing username env var")
		}
		if !strings.Contains(err.Error(), "source.username") {
			t.Fatalf("Load error = %q, want mention of source.username", err)
		}
	})

	t.Run("empty password env var", func(t *testing.T) {
		t.Setenv("POSTGRES_USERNAME", "postgres")
		t.Setenv("POSTGRES_PASSWORD", "")

		path := writeConfigFile(t, `
source:
  host: localhost
  port: 5432
  username:
    env-ref: POSTGRES_USERNAME
  password:
    env-ref: POSTGRES_PASSWORD
  dbname: postgres
`)

		_, err := config.Load(path)
		if err == nil {
			t.Fatal("Load returned nil error for empty password env var")
		}
		if !strings.Contains(err.Error(), "source.password") {
			t.Fatalf("Load error = %q, want mention of source.password", err)
		}
	})
}

func TestLoadResolvesSecretFileCredentials(t *testing.T) {
	usernamePath := writeSecretFile(t, "postgres\n")
	passwordPath := writeSecretFile(t, "secret\r\n")

	path := writeConfigFile(t, `
source:
  host: localhost
  port: 5432
  username:
    secret-file: `+usernamePath+`
  password:
    secret-file: `+passwordPath+`
  dbname: postgres
`)

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Source.Username != "postgres" {
		t.Fatalf("Username = %q, want %q", cfg.Source.Username, "postgres")
	}
	if cfg.Source.Password != "secret" {
		t.Fatalf("Password = %q, want %q", cfg.Source.Password, "secret")
	}
}

func TestLoadRejectsUnreadableOrEmptySecretFileCredentials(t *testing.T) {
	t.Run("missing username secret file", func(t *testing.T) {
		passwordPath := writeSecretFile(t, "secret\n")

		path := writeConfigFile(t, `
source:
  host: localhost
  port: 5432
  username:
    secret-file: `+filepath.Join(t.TempDir(), "missing-user")+`
  password:
    secret-file: `+passwordPath+`
  dbname: postgres
`)

		_, err := config.Load(path)
		if err == nil {
			t.Fatal("Load returned nil error for missing username secret file")
		}
		if !strings.Contains(err.Error(), "source.username") {
			t.Fatalf("Load error = %q, want mention of source.username", err)
		}
	})

	t.Run("empty password secret file after trimming line endings", func(t *testing.T) {
		usernamePath := writeSecretFile(t, "postgres")
		passwordPath := writeSecretFile(t, "\r\n")

		path := writeConfigFile(t, `
source:
  host: localhost
  port: 5432
  username:
    secret-file: `+usernamePath+`
  password:
    secret-file: `+passwordPath+`
  dbname: postgres
`)

		_, err := config.Load(path)
		if err == nil {
			t.Fatal("Load returned nil error for empty password secret file")
		}
		if !strings.Contains(err.Error(), "source.password") {
			t.Fatalf("Load error = %q, want mention of source.password", err)
		}
	})
}

func TestLoadRejectsInvalidConfigShape(t *testing.T) {
	testCases := []struct {
		name        string
		contents    string
		wantMessage string
	}{
		{
			name: "unknown source field",
			contents: `
source:
  host: localhost
  port: 5432
  username:
    value: postgres
  password:
    value: secret
  dbname: postgres
  unexpected: nope
`,
			wantMessage: "unexpected",
		},
		{
			name: "missing source object",
			contents: `
other:
  host: localhost
`,
			wantMessage: "source",
		},
		{
			name: "missing required host",
			contents: `
source:
  port: 5432
  username:
    value: postgres
  password:
    value: secret
  dbname: postgres
`,
			wantMessage: "source.host",
		},
		{
			name: "invalid port above range",
			contents: `
source:
  host: localhost
  port: 65536
  username:
    value: postgres
  password:
    value: secret
  dbname: postgres
`,
			wantMessage: "source.port",
		},
		{
			name: "multiple username credential modes",
			contents: `
source:
  host: localhost
  port: 5432
  username:
    value: postgres
    env-ref: POSTGRES_USERNAME
  password:
    value: secret
  dbname: postgres
`,
			wantMessage: "source.username",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			path := writeConfigFile(t, tc.contents)

			_, err := config.Load(path)
			if err == nil {
				t.Fatal("Load returned nil error for invalid config")
			}
			if !strings.Contains(err.Error(), tc.wantMessage) {
				t.Fatalf("Load error = %q, want mention of %q", err, tc.wantMessage)
			}
		})
	}
}

func TestLoadDoesNotExpandEnvOutsideExplicitCredentialRefs(t *testing.T) {
	t.Setenv("DB_HOST", "db.internal")
	t.Setenv("TLS_CERT_PATH", "/tmp/client.crt")

	path := writeConfigFile(t, `
source:
  host: ${DB_HOST}
  port: 5432
  username:
    value: ${POSTGRES_USERNAME}
  password:
    value: ${POSTGRES_PASSWORD}
  dbname: postgres
  tls:
    cert: ${TLS_CERT_PATH}
`)

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Source.Host != "${DB_HOST}" {
		t.Fatalf("Host = %q, want literal env placeholder", cfg.Source.Host)
	}
	if cfg.Source.Username != "${POSTGRES_USERNAME}" {
		t.Fatalf("Username = %q, want literal env placeholder", cfg.Source.Username)
	}
	if cfg.Source.Password != "${POSTGRES_PASSWORD}" {
		t.Fatalf("Password = %q, want literal env placeholder", cfg.Source.Password)
	}
	if cfg.Source.TLS.Cert != "${TLS_CERT_PATH}" {
		t.Fatalf("TLS.Cert = %q, want literal env placeholder", cfg.Source.TLS.Cert)
	}
}

func TestLoadTreatsTLSValuesAsPathsOnly(t *testing.T) {
	path := writeConfigFile(t, fmt.Sprintf(`
source:
  host: localhost
  port: 5432
  username:
    value: postgres
  password:
    value: secret
  dbname: postgres
  tls:
    ca_cert:
      env-ref: TLS_CA_CERT
    cert:
      secret-file: %s
    key: |
      -----BEGIN PRIVATE KEY-----
      abc
      -----END PRIVATE KEY-----
`, writeSecretFile(t, "ignored")))

	_, err := config.Load(path)
	if err == nil {
		t.Fatal("Load returned nil error for non-path TLS values")
	}
	if !strings.Contains(err.Error(), "tls") {
		t.Fatalf("Load error = %q, want mention of tls", err)
	}
}

func writeConfigFile(t *testing.T, contents string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(strings.TrimSpace(contents)), 0o600); err != nil {
		t.Fatalf("WriteFile config: %v", err)
	}

	return path
}

func writeSecretFile(t *testing.T, contents string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("WriteFile secret: %v", err)
	}

	return path
}
