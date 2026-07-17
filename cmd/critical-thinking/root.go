package main

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

// newRootCmd assembles the command tree. SilenceUsage keeps cobra from writing
// usage text on error — critical because `serve` (stdio) uses stdout as the
// JSON-RPC channel and must never be polluted. SilenceErrors is intentionally
// left FALSE so cobra writes a returned error (e.g. an unknown subcommand or bad
// flag) to stderr, giving the user a helpful diagnostic without ever touching
// stdout. Each subcommand's RunE owns its own stderr diagnostics; main() maps a
// non-nil error to exit 1. Bare invocation prints help and exits 0 — no
// auto-stdio (D5).
func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "critical-thinking",
		Short: "Critical, narrated, sequential thinking — an MCP server and CLI",
		Long: "critical-thinking is a Model Context Protocol server for critical,\n" +
			"narrated, sequential problem-solving.\n\n" +
			"Run `critical-thinking serve` to start the MCP server (stdio by\n" +
			"default, or --http for Streamable HTTP). See the subcommands below.",
		Example: "  critical-thinking serve\n" +
			"  critical-thinking serve --http :3000\n" +
			"  critical-thinking cli\n" +
			"  critical-thinking cli --once '<thought-json>'\n" +
			"  critical-thinking schema\n" +
			"  critical-thinking version",
		SilenceUsage:  true,
		SilenceErrors: false, // M1: report unknown-command/bad-flag errors to stderr (never stdout); SilenceUsage already prevents usage spam.
		// No Run/RunE: bare invocation prints help and exits 0 (D5).
		Version: versionText(),
	}
	// Print --version output verbatim (no name prefix), matching the version
	// subcommand's text form (D7).
	root.SetVersionTemplate("{{.Version}}\n")

	// Default I/O to the process streams; tests override via SetOut/SetErr.
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)

	// Phase 2/3: root persistent flags backed by Viper (CTHINK_ env prefix).
	// --verbose / CTHINK_VERBOSE lowers the level to Debug (and enables the stdio
	// JSON-RPC frame trace, see runStdio); --log-format / CTHINK_LOG_FORMAT selects
	// the handler. Precedence flag > env > default. The {text,json} contract lives
	// in newLogger, which returns an error on any other value.
	root.PersistentFlags().Bool("verbose", false, "enable debug logging (and stdio JSON-RPC frame tracing)")
	root.PersistentFlags().String("log-format", "text", "log output format: text|json")
	v := newConfigViper()

	root.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		if err := bindFlags(v, cmd.Flags()); err != nil {
			return err
		}
		level := slog.LevelInfo
		if v.GetBool("verbose") {
			level = slog.LevelDebug
		}
		logger, err := newLogger(os.Stderr, level, v.GetString("log_format"))
		if err != nil {
			return err // SilenceErrors=false → cobra prints to stderr; never stdout
		}
		slog.SetDefault(logger)
		return nil
	}

	root.AddCommand(
		newServeCmd(v).Command,
		newCliCmd(),
		newSchemaCmd(),
		newVersionCmd(),
	)
	return root
}
