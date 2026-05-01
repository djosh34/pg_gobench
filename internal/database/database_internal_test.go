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

func TestNewConnConfigBuildsRootPoolFromCAAnchorsInFullchainFile(t *testing.T) {
	t.Parallel()

	root := newTestCertificateAuthority(t, "pg-gobench-root", testCertificate{})
	intermediate := newTestCertificateAuthority(t, "pg-gobench-intermediate", root)
	leaf := newTestLeafCertificate(t, "db.internal", intermediate)

	connConfig, err := newConnConfig(config.Source{
		Host:     "localhost",
		Port:     5432,
		SSLMode:  config.SSLModeVerifyFull,
		Username: "postgres",
		Password: "secret",
		DBName:   "postgres",
		TLS: config.TLS{
			CACert: writeCertificatesPEM(t, "fullchain.pem", leaf, intermediate, root),
		},
	})
	if err != nil {
		t.Fatalf("newConnConfig returned error: %v", err)
	}

	subjects := connConfig.TLSConfig.RootCAs.Subjects()
	if len(subjects) != 2 {
		t.Fatalf("RootCAs.Subjects() returned %d subjects, want 2 CA subjects", len(subjects))
	}
	if containsRawSubject(subjects, leaf.cert.RawSubject) {
		t.Fatal("RootCAs.Subjects() unexpectedly contains the leaf certificate subject")
	}
	if !containsRawSubject(subjects, intermediate.cert.RawSubject) {
		t.Fatal("RootCAs.Subjects() does not contain the intermediate CA subject")
	}
	if !containsRawSubject(subjects, root.cert.RawSubject) {
		t.Fatal("RootCAs.Subjects() does not contain the root CA subject")
	}
	if connConfig.Fallbacks != nil {
		t.Fatal("newConnConfig unexpectedly left TLS fallbacks enabled")
	}
}

func TestNewConnConfigLoadsTraditionalCABundle(t *testing.T) {
	t.Parallel()

	root := newTestCertificateAuthority(t, "pg-gobench-root", testCertificate{})
	backupRoot := newTestCertificateAuthority(t, "pg-gobench-backup-root", testCertificate{})

	connConfig, err := newConnConfig(config.Source{
		Host:     "localhost",
		Port:     5432,
		SSLMode:  config.SSLModeVerifyFull,
		Username: "postgres",
		Password: "secret",
		DBName:   "postgres",
		TLS: config.TLS{
			CACert: writeCertificatesPEM(t, "ca-bundle.pem", root, backupRoot),
		},
	})
	if err != nil {
		t.Fatalf("newConnConfig returned error: %v", err)
	}

	subjects := connConfig.TLSConfig.RootCAs.Subjects()
	if len(subjects) != 2 {
		t.Fatalf("RootCAs.Subjects() returned %d subjects, want 2 CA subjects", len(subjects))
	}
	if !containsRawSubject(subjects, root.cert.RawSubject) {
		t.Fatal("RootCAs.Subjects() does not contain the primary CA subject")
	}
	if !containsRawSubject(subjects, backupRoot.cert.RawSubject) {
		t.Fatal("RootCAs.Subjects() does not contain the backup CA subject")
	}
}

func TestNewConnConfigRejectsLeafOnlyCACertificateFile(t *testing.T) {
	t.Parallel()

	root := newTestCertificateAuthority(t, "pg-gobench-root", testCertificate{})
	leaf := newTestLeafCertificate(t, "db.internal", root)

	_, err := newConnConfig(config.Source{
		Host:     "localhost",
		Port:     5432,
		SSLMode:  config.SSLModeVerifyFull,
		Username: "postgres",
		Password: "secret",
		DBName:   "postgres",
		TLS: config.TLS{
			CACert: writeCertificatesPEM(t, "leaf-only.pem", leaf),
		},
	})
	if err == nil {
		t.Fatal("newConnConfig returned nil error for a leaf-only CA file")
	}
	if !strings.Contains(err.Error(), "source.tls.ca_cert") {
		t.Fatalf("newConnConfig error = %q, want mention of source.tls.ca_cert", err)
	}
	if !strings.Contains(err.Error(), "no usable CA certificates found") {
		t.Fatalf("newConnConfig error = %q, want no usable CA certificates detail", err)
	}
}

func TestNewConnConfigRejectsMalformedCACertificateFile(t *testing.T) {
	t.Parallel()

	_, err := newConnConfig(config.Source{
		Host:     "localhost",
		Port:     5432,
		SSLMode:  config.SSLModeVerifyFull,
		Username: "postgres",
		Password: "secret",
		DBName:   "postgres",
		TLS: config.TLS{
			CACert: writeTextFile(t, "broken-ca.pem", "definitely not PEM"),
		},
	})
	if err == nil {
		t.Fatal("newConnConfig returned nil error for malformed CA PEM")
	}
	if !strings.Contains(err.Error(), "source.tls.ca_cert") {
		t.Fatalf("newConnConfig error = %q, want mention of source.tls.ca_cert", err)
	}
	if !strings.Contains(err.Error(), "malformed PEM data") {
		t.Fatalf("newConnConfig error = %q, want malformed PEM detail", err)
	}
}

func TestNewConnConfigRejectsInvalidCACertificateBytesEvenWhenBundleStartsValid(t *testing.T) {
	t.Parallel()

	root := newTestCertificateAuthority(t, "pg-gobench-root", testCertificate{})

	_, err := newConnConfig(config.Source{
		Host:     "localhost",
		Port:     5432,
		SSLMode:  config.SSLModeVerifyFull,
		Username: "postgres",
		Password: "secret",
		DBName:   "postgres",
		TLS: config.TLS{
			CACert: writeInvalidCertificatePEM(t, "invalid-ca.pem", root.cert.Raw, []byte("not a DER certificate")),
		},
	})
	if err == nil {
		t.Fatal("newConnConfig returned nil error for invalid CA certificate bytes")
	}
	if !strings.Contains(err.Error(), "source.tls.ca_cert") {
		t.Fatalf("newConnConfig error = %q, want mention of source.tls.ca_cert", err)
	}
}

func TestNewConnConfigLoadsClientCertificateMaterial(t *testing.T) {
	t.Parallel()

	root := newTestCertificateAuthority(t, "pg-gobench-root", testCertificate{})
	client := newTestLeafCertificate(t, "pg-gobench-client", root)
	clientCertPath, clientKeyPath := writeCertificateKeyPairPEM(t, "client", client)

	connConfig, err := newConnConfig(config.Source{
		Host:     "localhost",
		Port:     5432,
		SSLMode:  config.SSLModeVerifyFull,
		Username: "postgres",
		Password: "secret",
		DBName:   "postgres",
		TLS: config.TLS{
			CACert: writeCertificatePEM(t),
			Cert:   clientCertPath,
			Key:    clientKeyPath,
		},
	})
	if err != nil {
		t.Fatalf("newConnConfig returned error: %v", err)
	}
	if len(connConfig.TLSConfig.Certificates) != 1 {
		t.Fatalf("TLSConfig.Certificates returned %d certificates, want 1", len(connConfig.TLSConfig.Certificates))
	}
}

func writeCertificatePEM(t *testing.T) string {
	t.Helper()

	ca := newTestCertificateAuthority(t, "pg-gobench-test-ca", testCertificate{})

	return writeCertificatesPEM(t, "ca.pem", ca)
}

type testCertificate struct {
	cert       *x509.Certificate
	privateKey *rsa.PrivateKey
}

func newTestCertificateAuthority(t *testing.T, commonName string, issuer testCertificate) testCertificate {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey returned error: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber:          newSerialNumber(t),
		Subject:               pkix.Name{CommonName: commonName},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	parent := template
	signer := privateKey
	if issuer.cert != nil {
		parent = issuer.cert
		signer = issuer.privateKey
	}

	der, err := x509.CreateCertificate(rand.Reader, template, parent, &privateKey.PublicKey, signer)
	if err != nil {
		t.Fatalf("CreateCertificate returned error: %v", err)
	}

	certificate, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("ParseCertificate returned error: %v", err)
	}

	return testCertificate{
		cert:       certificate,
		privateKey: privateKey,
	}
}

func newTestLeafCertificate(t *testing.T, commonName string, issuer testCertificate) testCertificate {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey returned error: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber:          newSerialNumber(t),
		Subject:               pkix.Name{CommonName: commonName},
		DNSNames:              []string{commonName},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, issuer.cert, &privateKey.PublicKey, issuer.privateKey)
	if err != nil {
		t.Fatalf("CreateCertificate returned error: %v", err)
	}

	certificate, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("ParseCertificate returned error: %v", err)
	}

	return testCertificate{
		cert:       certificate,
		privateKey: privateKey,
	}
}

func writeCertificatesPEM(t *testing.T, filename string, certificates ...testCertificate) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), filename)
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			t.Fatalf("Close returned error: %v", err)
		}
	}()

	for _, certificate := range certificates {
		if err := pem.Encode(file, &pem.Block{Type: "CERTIFICATE", Bytes: certificate.cert.Raw}); err != nil {
			t.Fatalf("pem.Encode returned error: %v", err)
		}
	}

	return path
}

func containsRawSubject(subjects [][]byte, want []byte) bool {
	for _, subject := range subjects {
		if string(subject) == string(want) {
			return true
		}
	}

	return false
}

func newSerialNumber(t *testing.T) *big.Int {
	t.Helper()

	serialNumber, err := rand.Int(rand.Reader, big.NewInt(1<<62))
	if err != nil {
		t.Fatalf("rand.Int returned error: %v", err)
	}
	if serialNumber.Sign() == 0 {
		return big.NewInt(1)
	}

	return serialNumber
}

func writeTextFile(t *testing.T, filename, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), filename)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile(%q) returned error: %v", filename, err)
	}

	return path
}

func writeInvalidCertificatePEM(t *testing.T, filename string, blocks ...[]byte) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), filename)
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			t.Fatalf("Close returned error: %v", err)
		}
	}()

	for index, block := range blocks {
		if err := pem.Encode(file, &pem.Block{Type: "CERTIFICATE", Bytes: block}); err != nil {
			t.Fatalf("pem.Encode block %d returned error: %v", index, err)
		}
	}

	return path
}

func writeCertificateKeyPairPEM(t *testing.T, prefix string, certificate testCertificate) (string, string) {
	t.Helper()

	certPath := filepath.Join(t.TempDir(), prefix+".crt")
	certFile, err := os.Create(certPath)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	defer func() {
		if err := certFile.Close(); err != nil {
			t.Fatalf("Close returned error: %v", err)
		}
	}()

	if err := pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certificate.cert.Raw}); err != nil {
		t.Fatalf("pem.Encode returned error: %v", err)
	}

	keyPath := filepath.Join(t.TempDir(), prefix+".key")
	keyFile, err := os.Create(keyPath)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	defer func() {
		if err := keyFile.Close(); err != nil {
			t.Fatalf("Close returned error: %v", err)
		}
	}()

	privateKeyDER := x509.MarshalPKCS1PrivateKey(certificate.privateKey)
	if err := pem.Encode(keyFile, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: privateKeyDER}); err != nil {
		t.Fatalf("pem.Encode returned error: %v", err)
	}

	return certPath, keyPath
}
