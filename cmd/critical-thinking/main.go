package main

import (
	"flag"
	"fmt"
	"os"
)

// Injected at build time via -ldflags (see taskfile.yml / .goreleaser.yaml / Dockerfile).
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

var httpAddr = flag.String("http", "", "if set (e.g., \":3000\"), serve Streamable HTTP at this address; otherwise use stdio")

func main() {
	// `schema` subcommand: print the tool contract and exit. Checked before
	// flag.Parse so the bare word isn't treated as a flag-parse error.
	if len(os.Args) > 1 && os.Args[1] == "schema" {
		if err := printSchema(os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, "schema:", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	cliMode := flag.Bool("cli", false, "read NDJSON ThoughtData on stdin and stream transcripts (no MCP)")
	jsonOut := flag.Bool("json", false, "with -cli, emit structured ThoughtResponse as NDJSON instead of the transcript")
	flag.Parse()

	if *jsonOut && !*cliMode {
		fmt.Fprintln(os.Stderr, "cli: -json requires -cli")
		os.Exit(1)
	}

	switch {
	case *cliMode:
		os.Exit(runCLI(os.Stdin, os.Stdout, os.Stderr, *jsonOut))
	case *httpAddr != "":
		runHTTP(*httpAddr)
	default:
		runStdio()
	}
}
