package main

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"
	"testing"
)

func TestServeCmdDefaultsToStdio(t *testing.T) {
	cmd := newServeCmd()

	var stdioCalled bool
	var httpAddr string
	cmd.stdioRun = func() error { stdioCalled = true; return nil }
	cmd.httpRun = func(addr string) error { httpAddr = addr; return nil }

	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !stdioCalled {
		t.Error("bare `serve` should run stdio")
	}
	if httpAddr != "" {
		t.Errorf("bare `serve` should not run HTTP; got addr %q", httpAddr)
	}
}

func TestServeCmdHTTPWhenFlagSet(t *testing.T) {
	cmd := newServeCmd()

	var stdioCalled bool
	var httpAddr string
	cmd.stdioRun = func() error { stdioCalled = true; return nil }
	cmd.httpRun = func(addr string) error { httpAddr = addr; return nil }

	cmd.SetArgs([]string{"--http", ":3000"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if stdioCalled {
		t.Error("`serve --http` should not run stdio")
	}
	if httpAddr != ":3000" {
		t.Errorf("httpRun addr = %q, want :3000", httpAddr)
	}
}

func TestServeRunEPropagatesError(t *testing.T) {
	cmd := newServeCmd()
	wantErr := errors.New("boom")
	cmd.stdioRun = func() error { return wantErr }
	cmd.httpRun = func(string) error { return nil }
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if !errors.Is(err, wantErr) {
		t.Errorf("serve RunE err = %v, want %v", err, wantErr)
	}
}

// TestServeStdoutStaysCleanWhenLogging proves the load-bearing invariant: when
// the serve run path logs via the default slog logger, the record goes to the
// (stderr-bound) handler writer and NEVER to stdout. The stub re-points
// slog.SetDefault at a buffer because running serveCmd standalone does not
// invoke the root PersistentPreRunE.
func TestServeStdoutStaysCleanWhenLogging(t *testing.T) {
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })

	c := newServeCmd()
	var logBuf bytes.Buffer
	c.stdioRun = func() error {
		logger, err := newLogger(&logBuf, slog.LevelInfo, "text")
		if err != nil {
			return err
		}
		slog.SetDefault(logger)
		slog.Info("serve is logging")
		return nil
	}

	var out, errBuf bytes.Buffer
	c.SetOut(&out)
	c.SetErr(&errBuf)
	c.SetArgs([]string{})
	if err := c.Execute(); err != nil {
		t.Fatalf("Execute() err = %v", err)
	}
	if out.Len() != 0 {
		t.Errorf("stdout must stay clean while logging; got %q", out.String())
	}
	if !strings.Contains(logBuf.String(), "serve is logging") {
		t.Errorf("log record should reach the stderr-side buffer; got %q", logBuf.String())
	}
}
