package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"pg_gobench/internal/app"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	cfg, err := app.ParseConfig(args)
	if err != nil {
		writeFatal(stderr, "parse config: %v\n", err)
		return 2
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.Run(ctx, cfg, stdout, stderr); err != nil {
		writeFatal(stderr, "run service: %v\n", err)
		return 1
	}

	return 0
}

func writeFatal(w io.Writer, format string, args ...any) {
	if _, err := fmt.Fprintf(w, format, args...); err != nil {
		panic(fmt.Errorf("write fatal message: %w", err))
	}
}
