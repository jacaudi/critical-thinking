package main

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

// Tests in this package install a process-global logger via slog.SetDefault
// (through the root PersistentPreRunE and the serve logging stub). They must
// NOT call t.Parallel: parallel tests would race on the global default logger.
// Keep every test in this package serial.
func TestRootBareShowsHelpAndExitsZero(t *testing.T) {
	cmd := newRootCmd()
	var out, errBuf bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("bare root Execute() err = %v, want nil (exit 0)", err)
	}
	if !strings.Contains(out.String(), "Available Commands") {
		t.Errorf("bare root should print help; got: %s", out.String())
	}
	// D5: no auto-stdio — help text mentions the serve subcommand instead.
	if !strings.Contains(out.String(), "serve") {
		t.Errorf("help should list the serve subcommand; got: %s", out.String())
	}
}

func TestRootVersionFlagMatchesVersionText(t *testing.T) {
	cmd := newRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--version"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("--version Execute() err = %v", err)
	}
	got := strings.TrimRight(out.String(), "\n")
	if got != versionText() {
		t.Errorf("--version output = %q, want %q", got, versionText())
	}
}

func TestRootSubcommandsRegistered(t *testing.T) {
	cmd := newRootCmd()
	want := map[string]bool{"serve": false, "cli": false, "schema": false, "version": false}
	for _, sub := range cmd.Commands() {
		if _, ok := want[sub.Name()]; ok {
			want[sub.Name()] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("subcommand %q not registered", name)
		}
	}
}

// TestRootErrorPathKeepsStdoutClean proves pin 2: when a subcommand RunE
// errors, nothing reaches stdout (errors/usage go to stderr only).
func TestRootErrorPathKeepsStdoutClean(t *testing.T) {
	cmd := newRootCmd()
	var out, errBuf bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)
	// `cli` with a malformed line returns errCLIFailed.
	cmd.SetIn(strings.NewReader("garbage\n"))
	cmd.SetArgs([]string{"cli"})

	_ = cmd.Execute() // returns errCLIFailed; main would exit 1.
	if out.Len() != 0 {
		t.Errorf("stdout must stay clean on error path; got: %q", out.String())
	}
}

// TestRootUnknownCommandWritesStderrNotStdout proves the M1 refinement:
// SilenceErrors=false routes an unknown-command error to stderr (a helpful
// message) while SilenceUsage=true + stderr routing keep stdout clean (pin 2).
func TestRootUnknownCommandWritesStderrNotStdout(t *testing.T) {
	cmd := newRootCmd()
	var out, errBuf bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)
	cmd.SetArgs([]string{"bogus"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("unknown command should return an error")
	}
	if out.Len() != 0 {
		t.Errorf("stdout must stay clean on unknown-command; got: %q", out.String())
	}
	if !strings.Contains(errBuf.String(), "unknown command") {
		t.Errorf("stderr should describe the unknown command; got: %q", errBuf.String())
	}
}

// TestRootInvalidLogFormatKeepsStdoutClean: a bad --log-format fails closed via
// newLogger before any subcommand runs, on stderr only (stdout stays clean).
func TestRootInvalidLogFormatKeepsStdoutClean(t *testing.T) {
	cmd := newRootCmd()
	var out, errBuf bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)
	cmd.SetArgs([]string{"--log-format", "yaml", "version"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("invalid --log-format should return an error")
	}
	if out.Len() != 0 {
		t.Errorf("stdout must stay clean on invalid --log-format; got %q", out.String())
	}
	if !strings.Contains(errBuf.String(), "want text|json") {
		t.Errorf("stderr should carry newLogger's validation message; got %q", errBuf.String())
	}
}

// TestRootVerboseEnablesDebug: --verbose makes PersistentPreRunE install a
// Debug-level default logger before the (runnable) subcommand executes.
func TestRootVerboseEnablesDebug(t *testing.T) {
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })

	cmd := newRootCmd()
	var out, errBuf bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)
	cmd.SetArgs([]string{"--verbose", "version"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() err = %v", err)
	}
	if !slog.Default().Enabled(context.Background(), slog.LevelDebug) {
		t.Error("--verbose should set the default logger to Debug level")
	}
}

// TestRootVerboseEnvEnablesDebug: CTHINK_VERBOSE=true (env, no flag) enables Debug.
func TestRootVerboseEnvEnablesDebug(t *testing.T) {
	t.Setenv("CTHINK_VERBOSE", "true")
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })

	cmd := newRootCmd()
	var out, errBuf bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)
	cmd.SetArgs([]string{"version"}) // runnable subcommand → PersistentPreRunE runs
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() err = %v", err)
	}
	if !slog.Default().Enabled(context.Background(), slog.LevelDebug) {
		t.Error("CTHINK_VERBOSE=true should set the default logger to Debug")
	}
}

// TestRootInvalidLogFormatEnvKeepsStdoutClean: a bad CTHINK_LOG_FORMAT (env)
// fails closed via newLogger, on stderr only.
func TestRootInvalidLogFormatEnvKeepsStdoutClean(t *testing.T) {
	t.Setenv("CTHINK_LOG_FORMAT", "yaml")
	cmd := newRootCmd()
	var out, errBuf bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)
	cmd.SetArgs([]string{"version"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("invalid CTHINK_LOG_FORMAT should return an error")
	}
	if out.Len() != 0 {
		t.Errorf("stdout must stay clean; got %q", out.String())
	}
	if !strings.Contains(errBuf.String(), "want text|json") {
		t.Errorf("stderr should carry newLogger's validation message; got %q", errBuf.String())
	}
}

// TestRootVerboseFlagOverridesEnv exercises the full cobra path (flags parsed
// before PersistentPreRunE) to prove the precedence promise end-to-end: a
// passed --verbose wins over a conflicting CTHINK_VERBOSE=false. The bindFlags
// unit tests assert this at the boundary; this locks in the integration.
func TestRootVerboseFlagOverridesEnv(t *testing.T) {
	t.Setenv("CTHINK_VERBOSE", "false")
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })

	cmd := newRootCmd()
	var out, errBuf bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)
	cmd.SetArgs([]string{"--verbose", "version"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() err = %v", err)
	}
	if !slog.Default().Enabled(context.Background(), slog.LevelDebug) {
		t.Error("--verbose must override CTHINK_VERBOSE=false (flag > env)")
	}
}
