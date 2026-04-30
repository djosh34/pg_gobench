package database

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"pg_gobench/internal/config"
)

func TestNewConnConfigUsesConfiguredSSLMode(t *testing.T) {
	t.Parallel()

	connConfig, err := newConnConfig(config.Source{
		Host:     "localhost",
		Port:     5432,
		SSLMode:  config.SSLModeAllow,
		Username: "postgres",
		Password: "secret",
		DBName:   "postgres",
		TLS: config.TLS{
			CACert: writeCertificatePEM(t),
		},
	})
	if err != nil {
		t.Fatalf("newConnConfig returned error: %v", err)
	}
	if !strings.Contains(connConfig.ConnString(), "sslmode=allow") {
		t.Fatalf("ConnString() = %q, want sslmode=allow", connConfig.ConnString())
	}
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
