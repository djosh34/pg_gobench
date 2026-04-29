package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"pg_gobench/internal/httpserver"
)

const defaultAddr = "127.0.0.1:8080"

type Config struct {
	Addr string
}

func ParseConfig(args []string) (Config, error) {
	fs := flag.NewFlagSet("pg_gobench", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	cfg := Config{}
	fs.StringVar(&cfg.Addr, "addr", defaultAddr, "HTTP listen address")

	if err := fs.Parse(args); err != nil {
		return Config{}, fmt.Errorf("parse flags: %w", err)
	}
	if len(fs.Args()) > 0 {
		return Config{}, fmt.Errorf("unexpected positional arguments: %v", fs.Args())
	}
	if cfg.Addr == "" {
		return Config{}, errors.New("addr must not be empty")
	}

	return cfg, nil
}

func Run(ctx context.Context, cfg Config, stdout, stderr io.Writer) error {
	listener, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		return fmt.Errorf("listen on %q: %w", cfg.Addr, err)
	}

	server := httpserver.New(listener.Addr().String())
	server.ErrorLog = log.New(stderr, "httpserver: ", 0)

	if _, err := fmt.Fprintf(stdout, "listening on %s\n", listener.Addr().String()); err != nil {
		if closeErr := listener.Close(); closeErr != nil {
			return fmt.Errorf("write startup message: %v; close listener: %w", err, closeErr)
		}
		return fmt.Errorf("write startup message: %w", err)
	}

	serveErrCh := make(chan error, 1)
	go func() {
		err := server.Serve(listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErrCh <- fmt.Errorf("serve HTTP server: %w", err)
			return
		}
		serveErrCh <- nil
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown HTTP server: %w", err)
		}

		if err := <-serveErrCh; err != nil {
			return err
		}

		return nil
	case err := <-serveErrCh:
		return err
	}
}
