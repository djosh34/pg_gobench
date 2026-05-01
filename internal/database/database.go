package database

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"

	"pg_gobench/internal/config"
)

func Open(source config.Source) (*sql.DB, error) {
	connConfig, err := newConnConfig(source)
	if err != nil {
		return nil, err
	}

	return stdlib.OpenDB(*connConfig), nil
}

func newConnConfig(source config.Source) (*pgx.ConnConfig, error) {
	connURL := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(source.Username, source.Password),
		Host:   net.JoinHostPort(source.Host, strconv.Itoa(source.Port)),
		Path:   source.DBName,
	}

	query := connURL.Query()
	query.Set("sslmode", string(source.SSLMode))
	connURL.RawQuery = query.Encode()

	connConfig, err := pgx.ParseConfig(connURL.String())
	if err != nil {
		return nil, fmt.Errorf("build postgres connection config: %w", err)
	}
	if hasTLS(source.TLS) {
		tlsConfig, err := buildTLSConfig(source.TLS, connConfig.TLSConfig)
		if err != nil {
			return nil, err
		}
		connConfig.TLSConfig = tlsConfig
		connConfig.Fallbacks = nil
	}

	return connConfig, nil
}

func CheckReadiness(ctx context.Context, db pinger) error {
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("readiness ping: %w", err)
	}

	return nil
}

func hasTLS(cfg config.TLS) bool {
	return cfg.CACert != "" || cfg.Cert != "" || cfg.Key != ""
}

func buildTLSConfig(cfg config.TLS, base *tls.Config) (*tls.Config, error) {
	tlsConfig := &tls.Config{}
	if base != nil {
		tlsConfig = base.Clone()
	}

	if cfg.CACert != "" {
		rootCAs, err := loadRootCAs(cfg.CACert)
		if err != nil {
			return nil, err
		}
		tlsConfig.RootCAs = rootCAs
	}

	if cfg.Cert != "" {
		certificate, err := tls.LoadX509KeyPair(cfg.Cert, cfg.Key)
		if err != nil {
			return nil, fmt.Errorf("load source.tls cert/key %q and %q: %w", cfg.Cert, cfg.Key, err)
		}
		tlsConfig.Certificates = []tls.Certificate{certificate}
	}

	return tlsConfig, nil
}

func loadRootCAs(path string) (*x509.CertPool, error) {
	rootPEM, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read source.tls.ca_cert %q: %w", path, err)
	}

	rootCAs := x509.NewCertPool()
	foundCertificateBlock := false
	foundCA := false

	for len(rootPEM) > 0 {
		block, remainder := pem.Decode(rootPEM)
		if block == nil {
			if len(bytes.TrimSpace(rootPEM)) == 0 {
				break
			}

			return nil, fmt.Errorf("parse source.tls.ca_cert %q: malformed PEM data", path)
		}
		rootPEM = remainder

		if block.Type != "CERTIFICATE" {
			return nil, fmt.Errorf("parse source.tls.ca_cert %q: unexpected PEM block type %q", path, block.Type)
		}

		foundCertificateBlock = true

		certificate, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse source.tls.ca_cert %q: %w", path, err)
		}
		if !certificate.IsCA {
			continue
		}

		rootCAs.AddCert(certificate)
		foundCA = true
	}

	if !foundCertificateBlock {
		return nil, fmt.Errorf("parse source.tls.ca_cert %q: no PEM certificate blocks found", path)
	}
	if !foundCA {
		return nil, fmt.Errorf("parse source.tls.ca_cert %q: no usable CA certificates found", path)
	}

	return rootCAs, nil
}

type pinger interface {
	PingContext(context.Context) error
}
