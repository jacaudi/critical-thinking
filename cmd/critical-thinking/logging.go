package main

import (
	"fmt"
	"io"
	"log/slog"
)

// newLogger builds an slog.Logger writing to w at the given level. format selects
// the handler: "text" (human-readable) or "json" (structured). Any other value
// is an error — the {text,json} contract lives here and nowhere else. In
// production w is always os.Stderr (set by the root PersistentPreRunE); stdout is
// reserved for the JSON-RPC protocol and command output.
func newLogger(w io.Writer, level slog.Level, format string) (*slog.Logger, error) {
	opts := &slog.HandlerOptions{Level: level}
	var h slog.Handler
	switch format {
	case "text":
		h = slog.NewTextHandler(w, opts)
	case "json":
		h = slog.NewJSONHandler(w, opts)
	default:
		return nil, fmt.Errorf("invalid --log-format %q (want text|json)", format)
	}
	return slog.New(h), nil
}
