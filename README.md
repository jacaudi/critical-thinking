# Rubber Ducky MCP

A Model Context Protocol server for **critical, narrated, sequential thinking**. A rubber duck you talk to while you think — one step at a time — with required confidence calibration and adversarial self-critique on every thought.

It fuses three disciplines:

1. **Sequential thinking** — break problems into ordered, numbered steps; revise; branch.
2. **Rubber-duck narration** — explain each thought out loud, in first-person, to an imagined listener.
3. **Critical self-examination** — every thought is paired with confidence, assumptions, critique, and a counter-argument.

The single tool is `criticalthinking`. Every call must include the four critical-thinking fields — there is no opt-out, by design.

## Install

```bash
go install github.com/jacaudi/rubber-ducky-mcp@latest
# or
docker pull ghcr.io/jacaudi/rubber-ducky-mcp:latest
```

The Go install lands the binary at `$GOPATH/bin/rubber-ducky-mcp`.

## Run

```bash
# stdio (default; for Claude Desktop, Codex CLI, VS Code, etc.)
rubber-ducky-mcp

# Streamable HTTP
rubber-ducky-mcp -http :3000

# Docker (HTTP on :3000)
docker run --rm -p 3000:3000 ghcr.io/jacaudi/rubber-ducky-mcp:latest
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

The `text` content is a rendered transcript in rubber-duck voice. Subsequent calls can omit `thoughtNumber` (auto-assigned) and `totalThoughts` (inherited). Every critical-thinking field has a server-side length cap to enforce one-tight-sentence discipline. The full contract lives in the tool description itself.

## Client setup

`mcp.json` (Claude Desktop / Codex CLI / VS Code):

```json
{
  "mcpServers": {
    "rubber-ducky": { "command": "rubber-ducky-mcp" }
  }
}
```

Or HTTP:

```json
{
  "mcpServers": {
    "rubber-ducky": { "url": "http://localhost:3000/mcp" }
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

## License

[MIT](LICENSE).
