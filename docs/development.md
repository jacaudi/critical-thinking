# Development

## Toolchain

Go 1.26+ builds the binary on its own (the MCP SDK is a Go module). The verification targets also need [Task](https://taskfile.dev) and the lint tools listed under the dev container below, which has all of them.

## Build

```bash
task build   # bin/critical-thinking, stamped with the git version
```

`task build` passes `-X main.version=...` (plus commit and date) so the build's version is surfaced via `/health` and the MCP `Implementation.Version`.

## Test

All verification targets live in [`taskfile.yml`](../taskfile.yml); CI runs the same targets, so `task ci` locally is CI.

```bash
task ci          # repo checks (actionlint, yamllint, hadolint), Go checks (fmt, tidy, golangci-lint, govulncheck, go test -race), plugin lint+tests
task test        # just the tests (Go suite with the race detector, plugin shell tests)
task --list      # every target, including the per-module ones (go:lint, plugin:test, ...)
```

`-race` is the standard mode for this project: one `*mcp.Server` serves every HTTP request concurrently, and the OpenTelemetry middleware binds instruments once per process, so plain `go test` would miss data races on those paths.

### Dev container

`.devcontainer/` defines a container with the Go toolchain, Task, and every lint tool the targets need (golangci-lint, govulncheck, actionlint, hadolint, yamllint, shellcheck, jq) at the same pinned versions CI installs. With the [Dev Containers CLI](https://github.com/devcontainers/cli):

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
├── .github/                      # ci.yaml + local reusable stages (ci-*.yml), release-please config, renovate
├── .devcontainer/                # Go + Task + the lint tools; `task ci` runs here
├── taskfile.yml                  # THE CI CONTRACT: `ci` = what CI runs (modules in .taskfiles/)
└── Dockerfile                    # multi-stage, distroless final
```

The `internal/thinking` package has zero dependency on the MCP SDK and holds no
state — `Process` is a pure function — so it is fully unit-testable and safe
under any concurrency. All OpenTelemetry instrumentation lives in
`cmd/critical-thinking` (`otel.go`, `otelmiddleware.go`).

## Release workflow

[`.github/workflows/ci.yaml`](../.github/workflows/ci.yaml) runs on every push. Its `test` stage is exactly `task ci`; when a PR is open for the branch it also builds the multi-arch image (`build`) and boots it (`smoke`, which runs `task smoke`), and the single required status check is the `ci` job. Each stage is a local reusable workflow (`ci-*.yml`) sharing the `.github/actions/setup` composite for toolchains.

Releases are cut by [release-please](https://github.com/googleapis/release-please) (`release` stage, `main` only). It keeps a `chore: release X.Y.0` PR open with the CHANGELOG, the image tags in `docs/usage.md` and `docs/clients.md`, and the plugin hook's `EXPECTED_VERSION` (all via `x-release-please-version` annotations, configured in `.github/release-please-config.json`). Merging that PR creates the `vX.Y.Z` tag and GitHub release; the same run then attaches the goreleaser binaries and checksums (`release-binaries`), builds the release image from the tag (`release-image`, pushing `ghcr.io/jacaudi/critical-thinking:vX.Y.Z`, `:vX.Y`, `:vX` and `:latest`) and smoke-tests it (`release-smoke`). Nobody tags by hand; see [CONTRIBUTING.md](../CONTRIBUTING.md) for the commit-type → release mapping. `release-republish.yml` (manual) repairs a release image that shipped wrong.

## Treating the description as a protocol

The string in [`internal/thinking/description.go`](../internal/thinking/description.go) is the contract every client agent reads. Treat changes there like wire-format changes: bump the package version and add an entry to [migration.md](migration.md). Field renames, semantic changes, or removed guidance can all silently break agent behavior.
