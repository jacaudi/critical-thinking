# Development

## Toolchain

Go 1.26+ builds the binary on its own (the MCP SDK is a Go module). The verification targets also need [Task](https://taskfile.dev), `jq`, and `shellcheck`; the dev container below has all of them.

## Build

```bash
task build   # bin/critical-thinking, stamped with the git version
```

`task build` passes `-X main.version=...` (plus commit and date) so the build's version is surfaced via `/health` and the MCP `Implementation.Version`.

## Test

All verification targets live in [`taskfile.yml`](../taskfile.yml); CI runs the same targets, so `task ci` locally is CI.

```bash
task ci          # gofmt, vet, golangci-lint, govulncheck, plugin lint+tests, go test -race, build
task test-race   # just the Go suite, race detector on, no test cache
```

`-race` is the standard mode for this project: one `*mcp.Server` serves every HTTP request concurrently, and the OpenTelemetry middleware binds instruments once per process, so plain `go test` would miss data races on those paths.

### Dev container

`.devcontainer/` defines a container with the Go toolchain, Task, `jq`, and `shellcheck` — everything the targets need. With the [Dev Containers CLI](https://github.com/devcontainers/cli):

```bash
devcontainer up --workspace-folder .
devcontainer exec --workspace-folder . task ci
```

VS Code users can open the folder in the container directly.

## Debugging with MCP Inspector

The fastest way to manually exercise the tool is the official [MCP Inspector](https://github.com/modelcontextprotocol/inspector):

```bash
# stdio
npx @modelcontextprotocol/inspector critical-thinking serve

# HTTP
critical-thinking serve --http :3000 &
npx @modelcontextprotocol/inspector --uri http://localhost:3000/mcp
```

The inspector lets you call `criticalthinking` interactively and watch the rendered transcript without writing client code.

## Project layout

```
.
├── cmd/critical-thinking/        # Cobra command tree + MCP/CLI adapters
│   ├── main.go                   # entry point: builds the root command, maps errors to exit codes
│   ├── root.go                   # root command, persistent --verbose/--log-format flags
│   ├── serve.go                  # `serve` command: stdio vs --http path selection
│   ├── mcpserver.go              # MCP server wiring (stdio + stateless Streamable HTTP), CORS, /health
│   ├── auth.go                   # OIDC bearer-token verification for /mcp
│   ├── otel.go                   # OpenTelemetry providers (CTHINK_OTEL_ENABLED)
│   ├── otelmiddleware.go         # per-method spans and ct.mcp.* metrics
│   ├── cli.go                    # `cli` command: NDJSON stdin→stdout, no MCP host
│   ├── config.go                 # Viper config (CTHINK_ prefix), httpConfig
│   ├── logging.go                # slog logger construction (text|json → stderr)
│   ├── schema.go                 # `schema` command: prints the tool contract
│   ├── version.go                # `version` command + --version text
│   └── *_test.go                 # command + integration tests (SDK-client stateless tests, auth, OTel, no-process-state guard)
├── internal/thinking/
│   ├── description.go            # tool description (the prompt-engineering contract)
│   ├── schema.go                 # ThoughtData / ThoughtResponse + Validate()
│   ├── process.go                # Process(): the pure per-call engine
│   ├── transcript.go             # the narrated transcript renderer
│   └── *_test.go                 # unit tests, incl. the no-state/no-SDK boundary guard
├── plugins/critical-thinking/    # Claude Code plugin: skill, hooks, install script, shell tests
├── docs/                         # this documentation; docs/plans/ is local-only
├── scripts/                      # semantic-release helpers (version bumps)
├── .github/                      # CI (calls the taskfile targets) and release workflows
├── .devcontainer/                # Go + Task + jq + shellcheck; `task ci` runs here
├── taskfile.yml                  # every build/lint/test target; `ci` = what CI runs
└── Dockerfile                    # multi-stage, distroless final
```

The `internal/thinking` package has zero dependency on the MCP SDK and holds no
state — `Process` is a pure function — so it is fully unit-testable and safe
under any concurrency. All OpenTelemetry instrumentation lives in
`cmd/critical-thinking` (`otel.go`, `otelmiddleware.go`).

## Release workflow

CI runs on push and PR via GitHub Actions — the same `task ci` targets listed above (gofmt, vet, golangci-lint, govulncheck, plugin lint+tests, `go test -race`, build). Releases are cut by semantic-release on every push to `main`: it reads the conventional commits, creates the `vX.Y.Z` tag and GitHub release (goreleaser attaches the binaries), and the release workflow then builds and pushes the multi-arch Docker image to `ghcr.io/jacaudi/critical-thinking:vX.Y.Z` and updates `:latest`. Nobody tags by hand; see [CONTRIBUTING.md](../CONTRIBUTING.md) for the commit-type → release mapping.

## Treating the description as a protocol

The string in [`internal/thinking/description.go`](../internal/thinking/description.go) is the contract every client agent reads. Treat changes there like wire-format changes: bump the package version and add an entry to [migration.md](migration.md). Field renames, semantic changes, or removed guidance can all silently break agent behavior.
