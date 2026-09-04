package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/jacaudi/critical-thinking/internal/thinking"
)

func TestRunCLIJSONOutput(t *testing.T) {
	in := `{"thought":"x","thoughtNumber":1,"totalThoughts":1,"nextThoughtNeeded":false,"confidence":0.5,"assumptions":[],"critique":"c","counterArgument":"ca"}` + "\n"
	var out, errb bytes.Buffer
	code := runCLI(strings.NewReader(in), &out, &errb)
	if code != 0 {
		t.Fatalf("exit = %d; stderr = %s", code, errb.String())
	}
	var resp thinking.ThoughtResponse
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &resp); err != nil {
		t.Fatalf("stdout is not NDJSON ThoughtResponse: %v\n%s", err, out.String())
	}
	if resp.ThoughtNumber != 1 || resp.Confidence != 0.5 {
		t.Errorf("resp = %+v", resp)
	}
}

// TestRunCLILinesAreIndependent: the engine keeps nothing between lines, so
// two lines with the same thoughtNumber both succeed and echo their own data.
func TestRunCLILinesAreIndependent(t *testing.T) {
	in := strings.Join([]string{
		`{"thought":"a","thoughtNumber":1,"totalThoughts":1,"nextThoughtNeeded":false,"confidence":0.2,"assumptions":[],"critique":"c","counterArgument":"ca"}`,
		`{"thought":"b","thoughtNumber":1,"totalThoughts":1,"nextThoughtNeeded":false,"confidence":0.9,"assumptions":[],"critique":"c","counterArgument":"ca"}`,
	}, "\n") + "\n"
	var stdout, stderr bytes.Buffer
	if code := runCLI(strings.NewReader(in), &stdout, &stderr); code != 0 {
		t.Fatalf("exit %d, stderr=%s", code, stderr.String())
	}
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 output lines, got %d: %q", len(lines), stdout.String())
	}
	var first, second thinking.ThoughtResponse
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(lines[1]), &second); err != nil {
		t.Fatal(err)
	}
	if first.Confidence != 0.2 || second.Confidence != 0.9 || first.ThoughtNumber != 1 || second.ThoughtNumber != 1 {
		t.Errorf("lines not independent: %+v %+v", first, second)
	}
}

func TestRunCLIMalformedLineContinues(t *testing.T) {
	in := "{not json\n" +
		`{"thought":"ok","thoughtNumber":1,"totalThoughts":1,"nextThoughtNeeded":false,"confidence":0.5,"assumptions":[],"critique":"c","counterArgument":"ca"}` + "\n"
	var out, errb bytes.Buffer
	code := runCLI(strings.NewReader(in), &out, &errb)
	if code != 1 {
		t.Errorf("exit = %d; want 1", code)
	}
	if !strings.Contains(errb.String(), "line 1") {
		t.Errorf("stderr should reference line 1: %q", errb.String())
	}
	if !strings.Contains(out.String(), `"thoughtNumber":1`) {
		t.Errorf("a valid line after a bad one must still render:\n%s", out.String())
	}
}

// TestRunCLIValidationErrorRouting pins that a line the engine rejects
// (IsError) emits its error JSON to stdout — never stderr — so the NDJSON
// stream stays complete and parseable line-for-line.
func TestRunCLIValidationErrorRouting(t *testing.T) {
	// Missing required "critique" → validation error result (IsError).
	bad := `{"thought":"x","thoughtNumber":1,"totalThoughts":1,"nextThoughtNeeded":false,"confidence":0.5,"assumptions":[],"counterArgument":"ca"}` + "\n"

	var out, errb bytes.Buffer
	if code := runCLI(strings.NewReader(bad), &out, &errb); code != 1 {
		t.Errorf("exit = %d; want 1", code)
	}
	if !strings.Contains(out.String(), `"status":"failed"`) || errb.Len() != 0 {
		t.Errorf("error JSON should go to stdout only; out=%q err=%q", out.String(), errb.String())
	}
}

func TestRunCLIBlankAndEmpty(t *testing.T) {
	var out, errb bytes.Buffer
	if code := runCLI(strings.NewReader("\n   \n"), &out, &errb); code != 0 || out.Len() != 0 {
		t.Errorf("blank/empty: code=%d out=%q", code, out.String())
	}
}

// TestCliCmdExitsNonZeroOnAnyFailureAfterProcessingAll proves pin 1 at the
// subcommand layer: a bad line followed by a good line returns errCLIFailed
// (→ exit 1 in main) AND still emits the good line's output.
func TestCliCmdExitsNonZeroOnAnyFailureAfterProcessingAll(t *testing.T) {
	cmd := newCliCmd()
	cmd.SetIn(strings.NewReader("garbage\n" + `{"thought":"t","thoughtNumber":1,"totalThoughts":3,"nextThoughtNeeded":true,"confidence":0.5,"assumptions":[],"critique":"c","counterArgument":"x","nextStepRationale":"n"}` + "\n"))
	var out, errBuf bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if !errors.Is(err, errCLIFailed) {
		t.Fatalf("Execute() err = %v, want errCLIFailed", err)
	}
	if !strings.Contains(out.String(), `"thoughtNumber":1`) {
		t.Errorf("good line not processed (fail-fast?): %s", out.String())
	}
}

func TestCliCmdSuccessReturnsNil(t *testing.T) {
	cmd := newCliCmd()
	cmd.SetIn(strings.NewReader(`{"thought":"t","thoughtNumber":1,"totalThoughts":3,"nextThoughtNeeded":true,"confidence":0.5,"assumptions":[],"critique":"c","counterArgument":"x","nextStepRationale":"n"}` + "\n"))
	var out, errBuf bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() err = %v, want nil", err)
	}
}

// validOnceInput is one minimal valid ThoughtData document, shared by the
// --once tests.
const validOnceInput = `{"thought":"x","thoughtNumber":1,"totalThoughts":1,"nextThoughtNeeded":false,"confidence":0.5,"assumptions":[],"critique":"c","counterArgument":"ca"}`

func TestRunOnceArg(t *testing.T) {
	arg := validOnceInput
	var out, errb bytes.Buffer
	if code := runOnce(&arg, strings.NewReader(""), &out, &errb); code != 0 {
		t.Fatalf("exit = %d; stderr = %s", code, errb.String())
	}
	var resp thinking.ThoughtResponse
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &resp); err != nil {
		t.Fatalf("stdout is not a ThoughtResponse: %v\n%s", err, out.String())
	}
	if resp.ThoughtNumber != 1 || resp.Confidence != 0.5 {
		t.Errorf("resp = %+v", resp)
	}
	if errb.Len() != 0 {
		t.Errorf("stderr should be empty: %q", errb.String())
	}
}

// Pretty-printed multi-line JSON on stdin must work in --once mode — the one
// input shape the NDJSON stream loop cannot accept.
func TestRunOnceStdinFallbackPrettyJSON(t *testing.T) {
	pretty := "{\n  \"thought\": \"x\",\n  \"thoughtNumber\": 1,\n  \"totalThoughts\": 1,\n  \"nextThoughtNeeded\": false,\n  \"confidence\": 0.5,\n  \"assumptions\": [],\n  \"critique\": \"c\",\n  \"counterArgument\": \"ca\"\n}\n"
	var out, errb bytes.Buffer
	if code := runOnce(nil, strings.NewReader(pretty), &out, &errb); code != 0 {
		t.Fatalf("exit = %d; stderr = %s", code, errb.String())
	}
	if !strings.Contains(out.String(), `"thoughtNumber":1,"totalThoughts":1,"nextThoughtNeeded":false,"confidence":0.5`) {
		t.Errorf("expected one ThoughtResponse on stdout:\n%s", out.String())
	}
}

func TestRunOnceMalformedArg(t *testing.T) {
	arg := "{not json"
	var out, errb bytes.Buffer
	if code := runOnce(&arg, strings.NewReader(""), &out, &errb); code != 1 {
		t.Errorf("exit = %d; want 1", code)
	}
	if !strings.Contains(errb.String(), "argument") {
		t.Errorf("stderr should name the source 'argument': %q", errb.String())
	}
	if out.Len() != 0 {
		t.Errorf("stdout must stay clean: %q", out.String())
	}
}

// Mirrors TestRunCLIValidationErrorRouting for the single-shot path: an
// IsError result emits its error JSON to stdout, never stderr.
func TestRunOnceValidationErrorRouting(t *testing.T) {
	bad := `{"thought":"x","thoughtNumber":1,"totalThoughts":1,"nextThoughtNeeded":false,"confidence":0.5,"assumptions":[],"counterArgument":"ca"}`

	var out, errb bytes.Buffer
	if code := runOnce(&bad, strings.NewReader(""), &out, &errb); code != 1 {
		t.Errorf("exit = %d; want 1", code)
	}
	if !strings.Contains(out.String(), `"status":"failed"`) || errb.Len() != 0 {
		t.Errorf("error JSON should go to stdout only; out=%q err=%q", out.String(), errb.String())
	}
}

// Empty stdin is a FAILURE in --once mode (there is no next line to continue
// to) — deliberately unlike the stream loop's blank-line skip.
func TestRunOnceEmptyStdin(t *testing.T) {
	var out, errb bytes.Buffer
	if code := runOnce(nil, strings.NewReader("\n  \n"), &out, &errb); code != 1 {
		t.Errorf("exit = %d; want 1", code)
	}
	if !strings.Contains(errb.String(), "stdin") {
		t.Errorf("stderr should name the source 'stdin': %q", errb.String())
	}
}

// Trailing data after the document (e.g. two NDJSON lines piped into --once)
// is an error: --once means exactly one thought.
func TestRunOnceTrailingData(t *testing.T) {
	two := validOnceInput + "\n" + validOnceInput + "\n"
	var out, errb bytes.Buffer
	if code := runOnce(nil, strings.NewReader(two), &out, &errb); code != 1 {
		t.Errorf("exit = %d; want 1", code)
	}
}

func TestCliCmdOnceArgSuccess(t *testing.T) {
	cmd := newCliCmd()
	cmd.SetIn(strings.NewReader("")) // must NOT be read when the arg is given
	var out, errBuf bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)
	cmd.SetArgs([]string{"--once", validOnceInput})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() err = %v, want nil", err)
	}
	if !strings.Contains(out.String(), `"thoughtNumber":1,"totalThoughts":1,"nextThoughtNeeded":false,"confidence":0.5`) {
		t.Errorf("expected one ThoughtResponse on stdout: %s", out.String())
	}
}

func TestCliCmdOnceStdinFallback(t *testing.T) {
	cmd := newCliCmd()
	cmd.SetIn(strings.NewReader(validOnceInput))
	var out, errBuf bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)
	cmd.SetArgs([]string{"--once"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() err = %v, want nil", err)
	}
	if !strings.Contains(out.String(), `"thoughtNumber":1,"totalThoughts":1,"nextThoughtNeeded":false,"confidence":0.5`) {
		t.Errorf("expected one ThoughtResponse on stdout: %s", out.String())
	}
}

func TestCliCmdOnceFailureReturnsSentinel(t *testing.T) {
	cmd := newCliCmd()
	cmd.SetIn(strings.NewReader(""))
	var out, errBuf bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)
	cmd.SetArgs([]string{"--once", "{not json"})
	if err := cmd.Execute(); !errors.Is(err, errCLIFailed) {
		t.Fatalf("Execute() err = %v, want errCLIFailed", err)
	}
}

// A positional argument without --once must be rejected, so the stream
// contract is not silently changed.
func TestCliCmdArgWithoutOnceRejected(t *testing.T) {
	cmd := newCliCmd()
	cmd.SetIn(strings.NewReader(""))
	var out, errBuf bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)
	cmd.SetArgs([]string{validOnceInput})
	err := cmd.Execute()
	if err == nil || errors.Is(err, errCLIFailed) {
		t.Fatalf("Execute() err = %v, want a usage error distinct from errCLIFailed", err)
	}
	if !strings.Contains(err.Error(), "--once") {
		t.Errorf("error should point at --once: %v", err)
	}
}
