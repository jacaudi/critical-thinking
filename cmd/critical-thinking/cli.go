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

// errCLIFailed is the sentinel returned by the cli command's RunE when at least
// one input line failed. runCLI has already written per-line diagnostics to
// stderr; this sentinel drives main()'s exit code to 1. The root leaves
// SilenceErrors=false, so cobra also prints this error's message to stderr as a
// one-line summary — never to stdout.
var errCLIFailed = errors.New("cli: one or more input lines failed")

// runCLI runs the thinking engine over a plain stdin→stdout loop (no MCP).
// One in-memory thinking.NewServer() lives for the call, so history,
// confidence, and branches accumulate across input lines — the analog of a
// stdio MCP session. Input is NDJSON: one ThoughtData per non-blank line.
// Output is NDJSON too: one structured ThoughtResponse per processed line. A
// line the engine rejects emits its error JSON to stdout so the stream stays
// line-aligned; malformed-JSON lines are diagnosed on stderr only. Returns 0
// if every line succeeded, 1 if any line errored.
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
		var td thinking.ThoughtData
		// Write errors on stdout/stderr aren't actionable here; the exit code
		// already reflects per-line success via failed.
		if err := json.Unmarshal(line, &td); err != nil {
			_, _ = fmt.Fprintf(stderr, "cli: line %d: %v\n", lineNo, err)
			failed = true
			continue
		}
		res, err := state.ProcessThought(td)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "cli: line %d: %v\n", lineNo, err)
			failed = true
			continue
		}
		if res.IsError {
			failed = true
			_, _ = fmt.Fprintln(stdout, res.Text) // error JSON keeps NDJSON aligned
			continue
		}
		_, _ = fmt.Fprintln(stdout, res.StructuredJSON)
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

// newCliCmd streams NDJSON ThoughtData from stdin through the engine (no MCP),
// emitting one structured ThoughtResponse JSON object per line. It processes
// EVERY line, then returns errCLIFailed iff any line failed, so the exit code
// is 1 without fail-fast (pin 1).
func newCliCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cli",
		Short: "Stream NDJSON ThoughtData through the engine (no MCP host)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			code := runCLI(cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr())
			if code != 0 {
				return errCLIFailed
			}
			return nil
		},
	}
}
