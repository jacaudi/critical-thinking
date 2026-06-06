package main

import (
	"errors"
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
