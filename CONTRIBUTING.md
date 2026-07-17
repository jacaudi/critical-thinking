# Contributing

Thanks for your interest in improving critical-thinking! This is a short pointer to the conventions already documented in the repo.

## Getting started

The full developer guide lives in **[docs/development.md](docs/development.md)** — toolchain (Go 1.26+), build, test, project layout, and how to exercise the tool with the MCP Inspector. Start there.

Common tasks are wired into [`taskfile.yml`](taskfile.yml) (install [Task](https://taskfile.dev), then run `task --list`):

```bash
task build        # build the binary into bin/
task test         # go test ./...
task test-race    # race detector + coverage (the standard mode for this project)
task vet          # go vet ./...
task lint         # golangci-lint run ./...
```

`-race` is the expected test mode here — the HTTP path has non-trivial concurrency invariants that a plain `go test` will not catch. Run `task vet` and `gofmt -d .` clean before pushing; CI runs `vet`, `gofmt`, `go test -race`, and a Docker build on every push and PR.

## Commit messages

Releases are automated with [semantic-release](https://github.com/semantic-release/semantic-release), so commits must follow [Conventional Commits](https://www.conventionalcommits.org/):

| Type | Effect |
|------|--------|
| `fix: ...` | Patch release |
| `feat: ...` | Minor release |
| `feat!: ...` / `BREAKING CHANGE:` | Major release |
| `chore:`, `docs:`, `test:`, `ci:` | No release |

## The tool description is a protocol

The string in [`internal/thinking/description.go`](internal/thinking/description.go) is the contract every client agent reads. Treat changes there like wire-format changes — bump the package version and add an entry to [docs/migration.md](docs/migration.md). See the "Treating the description as a protocol" section in [docs/development.md](docs/development.md) for details.

## Pull requests

- Open PRs against `main` from a topic branch.
- Keep changes focused; include tests for behavior changes (`internal/thinking` is fully unit-testable by design).
- Make sure `task test-race`, `task vet`, and `task lint` pass locally.
