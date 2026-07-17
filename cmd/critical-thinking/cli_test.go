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
	if resp.ThoughtNumber != 1 || resp.ThoughtHistoryLength != 1 {
		t.Errorf("resp = %+v", resp)
	}
}

// TestRunCLIAccumulation pins that one in-memory server lives for the whole
// call: history accumulates across input lines, visible in the second
// response's thoughtHistoryLength.
func TestRunCLIAccumulation(t *testing.T) {
	in := strings.Join([]string{
		`{"thought":"first","thoughtNumber":1,"totalThoughts":2,"nextThoughtNeeded":true,"confidence":0.5,"assumptions":[],"critique":"c","counterArgument":"ca","nextStepRationale":"continue"}`,
		`{"thought":"second","thoughtNumber":2,"totalThoughts":2,"nextThoughtNeeded":false,"confidence":0.7,"assumptions":[],"critique":"c2","counterArgument":"ca2"}`,
	}, "\n") + "\n"

	var out, errb bytes.Buffer
	code := runCLI(strings.NewReader(in), &out, &errb)
	if code != 0 {
		t.Fatalf("exit = %d; stderr = %s", code, errb.String())
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 NDJSON lines, got %d:\n%s", len(lines), out.String())
	}
	var resp thinking.ThoughtResponse
	if err := json.Unmarshal([]byte(lines[1]), &resp); err != nil {
		t.Fatalf("second line is not a ThoughtResponse: %v\n%s", err, lines[1])
	}
	if resp.ThoughtNumber != 2 || resp.ThoughtHistoryLength != 2 {
		t.Errorf("second resp = %+v; want thoughtNumber 2, thoughtHistoryLength 2", resp)
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
	if !strings.Contains(out.String(), `"thoughtHistoryLength"`) {
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
	if !strings.Contains(out.String(), `"thoughtHistoryLength"`) {
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
	if resp.ThoughtNumber != 1 || resp.ThoughtHistoryLength != 1 {
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
	if !strings.Contains(out.String(), `"thoughtHistoryLength":1`) {
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
