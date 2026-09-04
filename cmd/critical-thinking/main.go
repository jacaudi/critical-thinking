// Package main is the critical-thinking binary: a Model Context Protocol
// server and CLI for critical, narrated, sequential problem-solving. See
// newRootCmd in root.go for the subcommand tree (serve, cli, schema, version).
package main

import "os"

// Injected at build time via -ldflags (see taskfile.yml / .goreleaser.yaml / Dockerfile).
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
