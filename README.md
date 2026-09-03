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

Or install the **Claude Code plugin** under [`plugins/critical-thinking/`](plugins/critical-thinking/): it auto-installs the server, adds a two-gate verification skill, and a hook that injects the two-gate protocol into every prompt. See [its README](plugins/critical-thinking/README.md).

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
{ "thoughtNumber": 1, "totalThoughts": 3, "nextThoughtNeeded": true, "confidence": 0.6 }
```

The `text` content is a rendered transcript in first-person, exploratory voice. Every call must send `thoughtNumber` and `totalThoughts` (both required, ≥ 1); if `thoughtNumber` exceeds `totalThoughts` the server raises `totalThoughts` to match. Keep each field to one tight sentence; the server does not enforce a length limit. The full contract lives in the tool description itself.

## Stateless by design

The server keeps nothing between calls: no history, no sessions, no running averages. Each call validates and narrates the one thought it receives. Your own context is the record — the model that wrote thought 3 already has thoughts 1 and 2 in front of it, and it judges its calibration across them itself. Over HTTP the server speaks MCP's sessionless model (protocol `2026-07-28`): no `Mcp-Session-Id`, every `POST` complete in itself, so any number of clients, gateways, or replicas can share one deployment without contaminating each other.

## Documentation

- [docs/usage.md](docs/usage.md) — install, MCP-server & CLI-pipe usage, a worked session
  - [Install](docs/usage.md#install) · [As an MCP server](docs/usage.md#as-an-mcp-server) · [As a CLI pipe](docs/usage.md#as-a-cli-pipe-no-mcp-host) · [Worked session](docs/usage.md#a-worked-session)
- [docs/clients.md](docs/clients.md) — Claude Desktop, Codex CLI, VS Code, Cursor recipes
- [docs/configuration.md](docs/configuration.md) — env vars, HTTP endpoints, [logging](docs/configuration.md#logging), CORS, stateless HTTP
- [docs/migration.md](docs/migration.md) — breaking changes since `http-sequential-thinking`
- [docs/development.md](docs/development.md) — building, testing, debugging with MCP Inspector
- [plugins/critical-thinking/](plugins/critical-thinking/) — the Claude Code plugin (bundled server install + two-gate skill + always-on protocol injection)

## License

[MIT](LICENSE).
