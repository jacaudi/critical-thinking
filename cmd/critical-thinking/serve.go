package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// serveCmd wraps the cobra command with overridable run hooks so path
// selection (stdio vs HTTP) is testable without binding a port. The hooks
// default to the real runStdio/runHTTP.
type serveCmd struct {
	*cobra.Command
	stdioRun func() error
	httpRun  func(cfg httpConfig, addr string) error
}

// newServeCmd builds the `serve` command. With no --http flag it runs the
// stdio MCP transport; with --http <addr> it runs the Streamable HTTP server,
// resolving its config (origins, host) from the shared Viper instance v.
func newServeCmd(v *viper.Viper) *serveCmd {
	c := &serveCmd{
		stdioRun: runStdio,
		httpRun:  runHTTP,
	}
	var httpAddr string
	c.Command = &cobra.Command{
		Use:   "serve",
		Short: "Run the MCP server (stdio by default; --http for Streamable HTTP)",
		Long: "Run the critical-thinking MCP server.\n\n" +
			"With no flags it serves over stdio (the default transport for\n" +
			"Claude Desktop, Codex CLI, VS Code, etc.). With --http <addr> it\n" +
			"serves Streamable HTTP at that address (e.g. --http :3000).",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if v.GetBool("otel_enabled") {
				shutdown, err := setupOTel(cmd.Context())
				if err != nil {
					return fmt.Errorf("otel setup: %w", err)
				}
				// Flush after the transport exits (for HTTP that is after the
				// graceful drain in runHTTP). A flush failure is logged, never
				// returned: telemetry must not fail a clean shutdown.
				defer func() {
					sctx, cancel := context.WithTimeout(context.Background(), shutdownGrace)
					defer cancel()
					if err := shutdown(sctx); err != nil {
						slog.Warn("otel shutdown", "error", err)
					}
				}()
			}
			if httpAddr != "" {
				return c.httpRun(httpConfigFromViper(v), httpAddr)
			}
			return c.stdioRun()
		},
	}
	c.Command.Flags().StringVar(&httpAddr, "http", "", `serve Streamable HTTP at this address (e.g. ":3000"); empty = stdio`)
	return c
}
