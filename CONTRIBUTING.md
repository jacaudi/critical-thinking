# Contributing

Thanks for your interest in improving critical-thinking! This is a short pointer to the conventions already documented in the repo.

## Getting started

The full developer guide lives in **[docs/development.md](docs/development.md)** — toolchain (Go 1.26+), build, test, project layout, and how to exercise the tool with the MCP Inspector. Start there.

Common tasks are wired into [`taskfile.yml`](taskfile.yml) (install [Task](https://taskfile.dev), then run `task --list`):

```bash
task ci           # everything CI runs: repo checks, Go checks, plugin checks
task test         # the Go suite with the race detector + the plugin shell tests
task lint         # static checks only (actionlint, yamllint, hadolint, golangci-lint, shellcheck)
task build        # build the binary into bin/
```

`task ci` runs exactly what CI runs. The `.devcontainer/` folder gives you a container with every tool the targets need (`devcontainer exec --workspace-folder . task ci`); see [docs/development.md](docs/development.md) for why `-race` is the standard mode.

## Commit messages

Releases are automated with [release-please](https://github.com/googleapis/release-please), so commits must follow [Conventional Commits](https://www.conventionalcommits.org/). release-please keeps a `chore: release X.Y.0` pull request open on `main` carrying the CHANGELOG and the version bumps; merging that PR cuts the tag, the GitHub release (goreleaser attaches the binaries) and the container image. Nobody tags by hand.

| Type | Effect |
|------|--------|
| `fix: ...`, `feat: ...`, `deps: ...` | Queued into the release PR |
| `feat!: ...` / `BREAKING CHANGE:` | Queued into the release PR; the major stays at 1 (project policy) |
| `chore:`, `docs:`, `refactor:`, `test:`, `ci:` | No release on their own; listed in the next release's changelog |

Every release bumps the **minor** version (`versioning: always-bump-minor` in `.github/release-please-config.json`): the project policy is that the major stays at 1, and release-please has no strategy that maps a breaking change to a minor bump while keeping patch bumps for fixes, so a fix-only release is a minor bump too. `chore(deps):` (the Renovate prefix) is not a releasable unit by itself; dependency bumps ship with the next `fix`/`feat`.

## The tool description is a protocol

The string in [`internal/thinking/description.go`](internal/thinking/description.go) is the contract every client agent reads. Treat changes there like wire-format changes — bump the package version and add an entry to [docs/migration.md](docs/migration.md). See the "Treating the description as a protocol" section in [docs/development.md](docs/development.md) for details.

## Pull requests

- Open PRs against `main` from a topic branch.
- Keep changes focused; include tests for behavior changes (`internal/thinking` is fully unit-testable by design).
- Make sure `task ci` passes locally (or in the dev container).
