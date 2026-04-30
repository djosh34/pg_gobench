package database_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"pg_gobench/internal/config"
	"pg_gobench/internal/database"
)

func TestOpenReturnsDatabaseHandleForValidatedSource(t *testing.T) {
	t.Parallel()

	db, err := database.Open(config.Source{
		Host:     "127.0.0.1",
		Port:     5432,
		SSLMode:  config.SSLModeDisable,
		Username: "postgres",
		Password: "secret",
		DBName:   "postgres",
	})
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("Close returned error: %v", err)
		}
	})
}

func TestOpenRejectsUnreadableTLSRootCertificate(t *testing.T) {
	t.Parallel()

	_, err := database.Open(config.Source{
		Host:     "127.0.0.1",
		Port:     5432,
		SSLMode:  config.SSLModeVerifyFull,
		Username: "postgres",
		Password: "secret",
		DBName:   "postgres",
		TLS: config.TLS{
			CACert: filepath.Join(t.TempDir(), "missing-ca.pem"),
		},
	})
	if err == nil {
		t.Fatal("Open returned nil error for unreadable TLS CA certificate")
	}
	if !strings.Contains(err.Error(), "ca_cert") {
		t.Fatalf("Open error = %q, want mention of ca_cert", err)
	}
}

func TestOpenLoadsTLSRootCertificatePath(t *testing.T) {
	t.Parallel()

	db, err := database.Open(config.Source{
		Host:     "localhost",
		Port:     5432,
		SSLMode:  config.SSLModeVerifyFull,
		Username: "postgres",
		Password: "secret",
		DBName:   "postgres",
		TLS: config.TLS{
			CACert: writeCertificatePEM(t),
		},
	})
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("Close returned error: %v", err)
		}
	})
}

func writeCertificatePEM(t *testing.T) string {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey returned error: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "pg-gobench-test-ca",
		},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("CreateCertificate returned error: %v", err)
	}

	path := filepath.Join(t.TempDir(), "ca.pem")
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			t.Fatalf("Close returned error: %v", err)
		}
	}()

	if err := pem.Encode(file, &pem.Block{Type: "CERTIFICATE", Bytes: der}); err != nil {
		t.Fatalf("pem.Encode returned error: %v", err)
	}

	return path
}

func TestCheckReadinessPingsDatabase(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := &stubPinger{}

	if err := database.CheckReadiness(ctx, db); err != nil {
		t.Fatalf("CheckReadiness returned error: %v", err)
	}
	if !db.called {
		t.Fatal("CheckReadiness did not call PingContext")
	}
	if db.ctx != ctx {
		t.Fatal("CheckReadiness did not pass the provided context to PingContext")
	}
}

func TestCheckReadinessReturnsPingErrorWithContext(t *testing.T) {
	t.Parallel()

	db := &stubPinger{err: errors.New("dial tcp 127.0.0.1:5432: connection refused")}

	err := database.CheckReadiness(context.Background(), db)
	if err == nil {
		t.Fatal("CheckReadiness returned nil error for ping failure")
	}
	if !strings.Contains(err.Error(), "readiness ping") {
		t.Fatalf("CheckReadiness error = %q, want mention of readiness ping", err)
	}
	if !strings.Contains(err.Error(), "connection refused") {
		t.Fatalf("CheckReadiness error = %q, want original ping error text", err)
	}
}

type stubPinger struct {
	ctx    context.Context
	called bool
	err    error
}

func (p *stubPinger) PingContext(ctx context.Context) error {
	p.ctx = ctx
	p.called = true
	return p.err
}
