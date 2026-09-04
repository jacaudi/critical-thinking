# Client setup

All snippets assume the binary `critical-thinking` is on your `$PATH`. After `go install github.com/jacaudi/critical-thinking/cmd/critical-thinking@latest`, that's `$GOPATH/bin/critical-thinking` — make sure `$GOPATH/bin` is on `$PATH`, or use the absolute path in the `command` field.

## Claude Code

### stdio

```bash
claude mcp add critical-thinking -- critical-thinking serve
```

Add `--scope user` to make it available in every project, or `--scope project` to commit it to a `.mcp.json` file in the repo. The default `local` scope keeps it private to the current project on this machine.

### Streamable HTTP

Run the server in one terminal:

```bash
critical-thinking serve --http :3000
```

Register it with Claude Code:

```bash
claude mcp add --transport http critical-thinking http://localhost:3000/mcp
```

### Verify

```bash
claude mcp list
```

Inside a Claude Code session, `/mcp` shows live status, and the `criticalthinking` tool will appear in tool listings.

### Plugin (bundled install)

Instead of the manual `claude mcp add` above, install the in-repo **Claude Code plugin** ([`plugins/critical-thinking/`](../plugins/critical-thinking/)). It registers the MCP server over stdio (downloading the binary automatically), ships the two-gate critical-thinking verification skill, and adds a hook that injects the two-gate protocol into every prompt. See the [plugin README](../plugins/critical-thinking/README.md) for install, the HTTP-transport variants, and how to disable the always-on hook.

## Claude Desktop

`~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) or `%APPDATA%\Claude\claude_desktop_config.json` (Windows):

```json
{
  "mcpServers": {
    "critical-thinking": {
      "command": "critical-thinking",
      "args": ["serve"]
    }
  }
}
```

Restart Claude Desktop after editing.

## Codex CLI

`~/.codex/mcp.json`:

```json
{
  "mcpServers": {
    "critical-thinking": {
      "command": "critical-thinking",
      "args": ["serve"]
    }
  }
}
```

## VS Code (Continue, Cline, etc.)

Most VS Code MCP-aware extensions use the same `mcp.json` shape:

```json
{
  "mcpServers": {
    "critical-thinking": {
      "command": "critical-thinking",
      "args": ["serve"]
    }
  }
}
```

## Cursor

`~/.cursor/mcp.json`:

```json
{
  "mcpServers": {
    "critical-thinking": {
      "url": "http://localhost:3000/mcp"
    }
  }
}
```

(Cursor currently prefers HTTP transport.) Run the server separately with `critical-thinking serve --http :3000`.

## Generic HTTP (any client)

Run the server in HTTP mode and point your client at `/mcp`:

```bash
critical-thinking serve --http :3000
```

```json
{
  "mcpServers": {
    "critical-thinking": {
      "url": "http://localhost:3000/mcp"
    }
  }
}
```

For browser-based clients, set `CTHINK_ALLOWED_ORIGINS` to permit your origin — see [configuration.md](configuration.md).

## Docker

<!-- x-release-please-start-version -->
```bash
docker run -d --name critical-thinking -p 3000:3000 ghcr.io/jacaudi/critical-thinking:v1.16.0
```
<!-- x-release-please-end -->

Then use the HTTP client config above. The image binds to `0.0.0.0` automatically (via `CTHINK_HTTP_HOST=0.0.0.0`); pair it with appropriate firewall rules in production.

## CLI (no MCP host)

Run the engine directly, without an MCP client:

```bash
# Stream thoughts: one ThoughtData JSON object per line on stdin.
# Output is structured ThoughtResponse NDJSON: one JSON object per line.
printf '%s\n' '{"thought":"...","thoughtNumber":1,"totalThoughts":3,"nextThoughtNeeded":true,"confidence":0.6,"assumptions":[],"critique":"...","counterArgument":"...","nextStepRationale":"..."}' \
  | critical-thinking cli

# Single-shot: one thought in, one result out, exit 0/1.
critical-thinking cli --once '{"thought":"...","thoughtNumber":1,"totalThoughts":1,"nextThoughtNeeded":false,"confidence":0.6,"assumptions":[],"critique":"...","counterArgument":"..."}'

# Single-shot from stdin (pretty-printed JSON is fine here):
critical-thinking cli --once < thought.json

# Print the tool contract (description + JSON Schemas) for a model to read:
critical-thinking schema
```

Each input is processed independently; the full stream and `--once` contract
(error routing, exit codes, the required fields) lives in
[usage.md](usage.md#as-a-cli-pipe-no-mcp-host).
