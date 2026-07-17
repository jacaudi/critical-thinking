package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/jacaudi/critical-thinking/internal/thinking"
	"github.com/spf13/cobra"
)

// errCLIFailed is the sentinel returned by the cli command's RunE when input
// failed to process (any line in stream mode; the single document in --once
// mode). runCLI/runOnce have already written diagnostics to stderr; this
// sentinel drives main()'s exit code to 1. The root leaves
// SilenceErrors=false, so cobra also prints this error's message to stderr as
// a one-line summary — never to stdout.
var errCLIFailed = errors.New("cli: input failed")

// runCLI runs the thinking engine over a plain stdin→stdout loop (no MCP).
// One in-memory thinking.NewServer() lives for the call, so history,
// confidence, and branches accumulate across input lines — the analog of a
// stdio MCP session. Input is NDJSON: one ThoughtData per non-blank line.
// Output is NDJSON too: one structured ThoughtResponse per processed line.
// Returns 0 if every line succeeded, 1 if any line errored.
func runCLI(stdin io.Reader, stdout, stderr io.Writer) int {
	state := thinking.NewServer()
	sc := bufio.NewScanner(stdin)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024) // tolerate long thoughts
	lineNo := 0
	failed := false
	for sc.Scan() {
		lineNo++
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		if !processOne(state, line, fmt.Sprintf("line %d", lineNo), stdout, stderr) {
			failed = true
		}
	}
	if err := sc.Err(); err != nil {
		_, _ = fmt.Fprintf(stderr, "cli: read: %v\n", err)
		return 1
	}
	if failed {
		return 1
	}
	return 0
}

// processOne unmarshals one ThoughtData JSON document from raw, processes it
// against state, and writes the result — the single source of the per-input
// contract shared by the stream loop and --once. src labels the input in
// diagnostics ("line 3", "argument", "stdin"). Success emits StructuredJSON
// to stdout; an IsError result emits the engine's error JSON to stdout too,
// so the NDJSON stream stays line-aligned. Malformed input is diagnosed on
// stderr only. Returns false if the input failed.
func processOne(state *thinking.SequentialThinkingServer, raw []byte, src string, stdout, stderr io.Writer) bool {
	var td thinking.ThoughtData
	// Write errors on stdout/stderr aren't actionable here; the return value
	// already reflects per-input success.
	if err := json.Unmarshal(raw, &td); err != nil {
		_, _ = fmt.Fprintf(stderr, "cli: %s: %v\n", src, err)
		return false
	}
	res, err := state.ProcessThought(td)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "cli: %s: %v\n", src, err)
		return false
	}
	if res.IsError {
		_, _ = fmt.Fprintln(stdout, res.Text) // error JSON keeps NDJSON aligned
		return false
	}
	_, _ = fmt.Fprintln(stdout, res.StructuredJSON)
	return true
}

// runOnce processes exactly one ThoughtData through a fresh in-memory server
// and exits — the single-shot analog of runCLI for scripting/testing (#74).
// The document comes from arg when non-nil, else from ALL of stdin — one JSON
// document, so pretty-printed multi-line JSON is fine, unlike the NDJSON
// loop. Empty input and trailing data after the document are unmarshal
// failures: with no next line to continue to, --once has nothing to skip.
// Returns 0 on success, 1 on any failure.
func runOnce(arg *string, stdin io.Reader, stdout, stderr io.Writer) int {
	var raw []byte
	src := "stdin"
	if arg != nil {
		raw, src = []byte(*arg), "argument"
	} else {
		// Unbounded by design: the stream loop's 1 MiB buffer is a
		// bufio.Scanner token-size mechanism, not an input-size contract
		// (serve mode reads unbounded JSON-RPC frames too). Design §5.2.
		b, err := io.ReadAll(stdin)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "cli: read: %v\n", err)
			return 1
		}
		raw = b
	}
	if processOne(thinking.NewServer(), raw, src, stdout, stderr) {
		return 0
	}
	return 1
}

// newCliCmd streams NDJSON ThoughtData from stdin through the engine (no MCP),
// emitting one structured ThoughtResponse JSON object per line. Stream mode
// processes EVERY line, then returns errCLIFailed iff any line failed, so the
// exit code is 1 without fail-fast (pin 1). --once instead processes exactly
// one ThoughtData — from the positional argument, or stdin when omitted —
// against a fresh server, and exits 0/1 (#74).
func newCliCmd() *cobra.Command {
	var once bool
	cmd := &cobra.Command{
		Use:   "cli [thought-json]",
		Short: "Process ThoughtData through the engine, streamed via stdin or as a single --once call (no MCP host)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// The positional is only meaningful in --once mode; rejecting it
			// otherwise keeps the stream contract unchanged. Checked here (not
			// a custom Args validator) so the flag value is unambiguously
			// parsed and all mode logic reads top-to-bottom in one place.
			if len(args) == 1 && !once {
				return errors.New("cli: an argument requires --once")
			}
			var code int
			if once {
				var arg *string
				if len(args) == 1 {
					arg = &args[0]
				}
				code = runOnce(arg, cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr())
			} else {
				code = runCLI(cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr())
			}
			if code != 0 {
				return errCLIFailed
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&once, "once", false, "process exactly one ThoughtData (from the argument, or stdin if omitted) and exit")
	return cmd
}
