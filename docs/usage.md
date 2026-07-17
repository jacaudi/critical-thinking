# Usage

`critical-thinking` runs two ways: as an **MCP server** (the usual path — your AI
host calls the `criticalthinking` tool) or as a **CLI pipe** (no MCP host — your
own script feeds it NDJSON). Install once, then pick the path that fits.

## Install

```bash
# Go toolchain (lands at $GOPATH/bin/critical-thinking)
go install github.com/jacaudi/critical-thinking/cmd/critical-thinking@latest

# Container image (pin to a release tag)
docker pull ghcr.io/jacaudi/critical-thinking:v1.12.0
```

Prebuilt binaries for each release are attached to the
[GitHub releases](https://github.com/jacaudi/critical-thinking/releases).

## As an MCP server

The default transport is stdio (what Claude Desktop, Codex CLI, VS Code, etc.
expect); `--http` switches to Streamable HTTP.

```bash
critical-thinking serve                 # stdio (default)
critical-thinking serve --http :3000    # Streamable HTTP on :3000
docker run --rm -p 3000:3000 ghcr.io/jacaudi/critical-thinking:v1.12.0   # HTTP in a container
```

Register it with Claude Code using the `claude` CLI:

```bash
# stdio (Claude Code runs the binary as a subprocess)
claude mcp add critical-thinking -- critical-thinking serve

# Streamable HTTP (run the server separately, point Claude Code at the URL)
critical-thinking serve --http :3000 &
claude mcp add --transport http critical-thinking http://localhost:3000/mcp
```

Scope with `--scope user` (every project) or `--scope project` (committed to
`.mcp.json`); default is `local`. Verify with `claude mcp list`; inside a session
`/mcp` shows server status and tools.

Or hand-write `mcp.json` (Claude Desktop / Codex CLI / VS Code):

```json
{
  "mcpServers": {
    "critical-thinking": { "command": "critical-thinking", "args": ["serve"] }
  }
}
```

```json
{
  "mcpServers": {
    "critical-thinking": { "url": "http://localhost:3000/mcp" }
  }
}
```

- Full per-host recipes (Claude Desktop, Codex CLI, VS Code, Cursor): [clients.md](clients.md)
- Env vars, HTTP host/port, CORS, logging: [configuration.md](configuration.md)

## As a CLI pipe (no MCP host)

`critical-thinking cli` runs the same thinking engine over stdin→stdout — no MCP
host required. This is **not** an MCP integration: no host is involved and no
`criticalthinking` tool is exposed — you drive the binary yourself. Have your own
agent, script, or orchestrator emit NDJSON `ThoughtData` (one JSON object per
line) and read the result back.

- `critical-thinking cli` prints structured `ThoughtResponse` as NDJSON — one
  JSON object per processed line, the machine-readable surface for programmatic
  callers.

History, confidence, and branches accumulate across input lines that share an
`episodeId` (absent → the `"default"` episode) within one run
(the analog of a single stdio MCP session). Every line is processed; the command
exits non-zero if any line fails. A malformed-JSON line is reported on stderr. A
line the engine rejects (for example a validation error) is emitted to stdout
as a JSON error object (`{"error":…,"status":"failed"}`) so the output stream
stays complete and parseable line-for-line.

Each `ThoughtData` line must carry the required fields — `thought`,
`thoughtNumber`, `totalThoughts`, `nextThoughtNeeded`, `confidence` (0.0–1.0),
`assumptions` (use `[]` if none), `critique`, `counterArgument`, and
`nextStepRationale` when `nextThoughtNeeded` is `true`. See
[clients.md#cli-no-mcp-host](clients.md#cli-no-mcp-host) for the full
field-by-field contract.

- `episodeId` (string, optional): partitions state into independent reasoning
  episodes. Absent → the shared `"default"` episode. Reuse one value per problem;
  switch for a new problem. Echoed back in the response.

## A worked session

Three thoughts — an initial thought, a revision of it, then a branch — piped in
as NDJSON:

```bash
critical-thinking cli <<'EOF'
{"thought":"Reads dominate writes here, so I'll normalize the schema first.","thoughtNumber":1,"totalThoughts":3,"nextThoughtNeeded":true,"confidence":0.6,"assumptions":["read:write ratio is ~10:1"],"critique":"I jumped to a solution before confirming the ratio.","counterArgument":"If writes dominate, a denormalized store is simpler.","nextStepRationale":"Verify the read:write ratio before committing to normalization."}
{"thought":"Correction: the measured ratio is ~2:1, so normalization is far less decisive.","thoughtNumber":2,"totalThoughts":3,"nextThoughtNeeded":true,"isRevision":true,"revisesThought":1,"confidence":0.7,"assumptions":["measured read:write ratio is 2:1"],"critique":"My first thought over-weighted reads on an unverified 10:1 guess.","counterArgument":"Even at 2:1 reads still lead, so normalizing isn't wrong, just weaker.","nextStepRationale":"Weigh write-amplification against the modest read advantage."}
{"thought":"Branch: keep one denormalized table and accept write fan-out instead.","thoughtNumber":1,"totalThoughts":2,"nextThoughtNeeded":false,"branchFromThought":1,"branchId":"denormalized","confidence":0.5,"assumptions":["write fan-out stays under 3x"],"critique":"This trades read simplicity for a write cost I have not measured.","counterArgument":"If fan-out exceeds 3x, this branch is worse than normalizing."}
EOF
```

Output — one `ThoughtResponse` per line:

```
{"thoughtNumber":1,"totalThoughts":3,"nextThoughtNeeded":true,"branches":[],"thoughtHistoryLength":1,"sessionConfidence":0.6}
{"thoughtNumber":2,"totalThoughts":3,"nextThoughtNeeded":true,"branches":[],"thoughtHistoryLength":2,"sessionConfidence":0.6499999999999999}
{"thoughtNumber":1,"totalThoughts":2,"nextThoughtNeeded":false,"branches":["denormalized"],"thoughtHistoryLength":3,"sessionConfidence":0.6499999999999999,"branchConfidences":{"denormalized":0.5}}
```

## schema and version

```bash
critical-thinking schema    # prints the full tool contract (description + JSON Schemas) and exits
critical-thinking version   # prints version/commit/date; --json for structured output
```

## See also

- [configuration.md](configuration.md) — env vars, HTTP, logging, CORS
- [clients.md](clients.md) — per-host MCP recipes
- [migration.md](migration.md) — breaking changes across versions
- [development.md](development.md) — building, testing, MCP Inspector
