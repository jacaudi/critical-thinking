# Contributing

Thanks for your interest in improving critical-thinking! This is a short pointer to the conventions already documented in the repo.

## Getting started

The full developer guide lives in **[docs/development.md](docs/development.md)** — toolchain (Go 1.26+), build, test, project layout, and how to exercise the tool with the MCP Inspector. Start there.

Common tasks are wired into [`taskfile.yml`](taskfile.yml) (install [Task](https://taskfile.dev), then run `task --list`):

```bash
task ci           # everything CI runs, in CI's order
task test-race    # race detector + coverage (the standard mode for this project)
task lint         # golangci-lint run ./...
task build        # build the binary into bin/
```

`task ci` runs exactly what CI runs. The `.devcontainer/` folder gives you a container with every tool the targets need (`devcontainer exec --workspace-folder . task ci`); see [docs/development.md](docs/development.md) for why `-race` is the standard mode.

## Commit messages

Releases are automated with [semantic-release](https://github.com/semantic-release/semantic-release), so commits must follow [Conventional Commits](https://www.conventionalcommits.org/):

| Type | Effect |
|------|--------|
| `fix: ...` | Patch release |
| `feat: ...` | Minor release |
| `feat!: ...` / `BREAKING CHANGE:` | Minor release (project policy in `.releaserc.json`; the major stays at 1) |
| `refactor: ...` | Minor release (project policy) |
| `chore(deps): ...` | Patch release |
| `chore:`, `docs:`, `test:`, `ci:` | No release |

## The tool description is a protocol

The string in [`internal/thinking/description.go`](internal/thinking/description.go) is the contract every client agent reads. Treat changes there like wire-format changes — bump the package version and add an entry to [docs/migration.md](docs/migration.md). See the "Treating the description as a protocol" section in [docs/development.md](docs/development.md) for details.

## Pull requests

- Open PRs against `main` from a topic branch.
- Keep changes focused; include tests for behavior changes (`internal/thinking` is fully unit-testable by design).
- Make sure `task ci` passes locally (or in the dev container).
