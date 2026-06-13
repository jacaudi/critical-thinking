# Critical Thinking

A Model Context Protocol server for **critical, narrated, sequential thinking**. Think one step at a time, out loud — with required confidence calibration and adversarial self-critique on every thought.

It fuses three disciplines:

1. **Sequential thinking** — break problems into ordered, numbered steps; revise; branch.
2. **Thinking out loud** — explain each thought in first-person, exploratory voice. Putting half-formed reasoning into words is itself the double-check on it.
3. **Critical self-examination** — every thought is paired with confidence, assumptions, critique, and a counter-argument.

The single tool is `criticalthinking`. Every call must include the four critical-thinking fields — there is no opt-out, by design.

**Install & usage → [docs/usage.md](docs/usage.md).** One-line install:

```bash
go install github.com/jacaudi/critical-thinking/cmd/critical-thinking@latest
```

Or install the **Claude Code plugin** under [`plugins/critical-thinking/`](plugins/critical-thinking/): it auto-installs the server, adds an always-on two-gate verification skill, and a hook that activates it every turn. See [its README](plugins/critical-thinking/README.md).

## One-call example

Request:

```json
{
  "thought": "I think we should normalize first because reads dominate writes.",
  "thoughtNumber": 1, "totalThoughts": 3, "nextThoughtNeeded": true,
  "confidence": 0.6,
  "assumptions": ["read:write ratio is ~10:1"],
  "critique": "Drifted into solution mode without confirming the ratio.",
  "counterArgument": "If writes dominate, monolith-first is simpler.",
  "nextStepRationale": "Verify the read:write ratio before committing to normalization."
}
```

Response (`structuredContent`):

```json
{
  "thoughtNumber": 1, "totalThoughts": 3, "nextThoughtNeeded": true,
  "branches": [], "thoughtHistoryLength": 1, "sessionConfidence": 0.6
}
```

The `text` content is a rendered transcript in first-person, exploratory voice. Every call must send `thoughtNumber` and `totalThoughts` (both required, ≥ 1); if `thoughtNumber` exceeds `totalThoughts` the server raises `totalThoughts` to match. Keep each field to one tight sentence — the tool description asks for that brevity; the server does not enforce a hard limit. The full contract lives in the tool description itself.

## Resources

The server exposes `thinking://current` — a per-session JSON snapshot of the full thought history (trunk + branches, all critical-thinking fields preserved).

## Documentation

- [docs/usage.md](docs/usage.md) — install, MCP-server & CLI-pipe usage, a worked session
  - [Install](docs/usage.md#install) · [As an MCP server](docs/usage.md#as-an-mcp-server) · [As a CLI pipe](docs/usage.md#as-a-cli-pipe-no-mcp-host) · [Worked session](docs/usage.md#a-worked-session)
- [docs/clients.md](docs/clients.md) — Claude Desktop, Codex CLI, VS Code, Cursor recipes
- [docs/configuration.md](docs/configuration.md) — env vars, HTTP endpoints, [logging](docs/configuration.md#logging), CORS, session lifecycle
- [docs/migration.md](docs/migration.md) — breaking changes since `http-sequential-thinking`
- [docs/development.md](docs/development.md) — building, testing, debugging with MCP Inspector
- [plugins/critical-thinking/](plugins/critical-thinking/) — the Claude Code plugin (bundled server install + always-on critical-thinking skill)

## License

[MIT](LICENSE).
