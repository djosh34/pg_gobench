package database

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"errors"
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
	query.Set("sslmode", sslMode(source.TLS))
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

func sslMode(cfg config.TLS) string {
	if hasTLS(cfg) {
		return "verify-full"
	}

	return "disable"
}

func hasTLS(cfg config.TLS) bool {
	return cfg.CACert != "" || cfg.Cert != "" || cfg.Key != ""
}

func buildTLSConfig(cfg config.TLS, base *tls.Config) (*tls.Config, error) {
	if cfg.Cert == "" && cfg.Key != "" {
		return nil, errors.New("source.tls.cert is required when source.tls.key is set")
	}
	if cfg.Key == "" && cfg.Cert != "" {
		return nil, errors.New("source.tls.key is required when source.tls.cert is set")
	}

	tlsConfig := &tls.Config{}
	if base != nil {
		tlsConfig = base.Clone()
	}

	if cfg.CACert != "" {
		rootPEM, err := os.ReadFile(cfg.CACert)
		if err != nil {
			return nil, fmt.Errorf("read source.tls.ca_cert %q: %w", cfg.CACert, err)
		}

		rootCAs := x509.NewCertPool()
		if !rootCAs.AppendCertsFromPEM(rootPEM) {
			return nil, fmt.Errorf("parse source.tls.ca_cert %q: no certificates found", cfg.CACert)
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

type pinger interface {
	PingContext(context.Context) error
}
