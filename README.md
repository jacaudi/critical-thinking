# Critical Thinking Plugin

A Model Context Protocol server for **critical, narrated, sequential thinking**. Think one step at a time, out loud — with required confidence calibration and adversarial self-critique on every thought.

It fuses three disciplines:

1. **Sequential thinking** — break problems into ordered, numbered steps; revise; branch.
2. **Thinking out loud** — explain each thought in first-person, exploratory voice. Putting half-formed reasoning into words is itself the double-check on it.
3. **Critical self-examination** — every thought is paired with confidence, assumptions, critique, and a counter-argument.

The single tool is `criticalthinking`. Every call must include the four critical-thinking fields — there is no opt-out, by design.

## Install

```bash
go install github.com/jacaudi/critical-thinking-plugin/cmd/critical-thinking@latest
# or
docker pull ghcr.io/jacaudi/critical-thinking:latest
```

The Go install lands the binary at `$GOPATH/bin/critical-thinking`.

## Run

```bash
# stdio (default; for Claude Desktop, Codex CLI, VS Code, etc.)
critical-thinking

# Streamable HTTP
critical-thinking -http :3000

# Docker (HTTP on :3000)
docker run --rm -p 3000:3000 ghcr.io/jacaudi/critical-thinking:latest
```

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
{ "branches": [], "thoughtHistoryLength": 1, "sessionConfidence": 0.6 }
```

The `text` content is a rendered transcript in first-person, exploratory voice. Subsequent calls can omit `thoughtNumber` (auto-assigned) and `totalThoughts` (inherited). Every critical-thinking field has a server-side length cap to enforce one-tight-sentence discipline. The full contract lives in the tool description itself.

## Client setup

`mcp.json` (Claude Desktop / Codex CLI / VS Code):

```json
{
  "mcpServers": {
    "critical-thinking": { "command": "critical-thinking" }
  }
}
```

Or HTTP:

```json
{
  "mcpServers": {
    "critical-thinking": { "url": "http://localhost:3000/mcp" }
  }
}
```

More client recipes in [docs/clients.md](docs/clients.md).

## Resources

The server exposes `thinking://current` — a per-session JSON snapshot of the full thought history (trunk + branches, all critical-thinking fields preserved).

## Documentation

- [docs/configuration.md](docs/configuration.md) — env vars, HTTP endpoints, session lifecycle
- [docs/clients.md](docs/clients.md) — Claude Desktop, Codex CLI, VS Code, Cursor recipes
- [docs/development.md](docs/development.md) — building, testing, debugging with MCP Inspector
- [docs/migration.md](docs/migration.md) — breaking changes since `http-sequential-thinking`

## Claude Code plugin

The [`critical-thinking`](plugins/critical-thinking/) plugin teaches Claude *when* to reach for the `criticalthinking` tool — both as the primary thinking process (in place of silent extended thinking or v1 sequential-thinking) and as the post-hoc pressure-test after another thinking session. The MCP server is the *how* and *why*; the plugin is the *when*.

## License

[MIT](LICENSE).
