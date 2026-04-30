# Client setup

All snippets assume the binary `rubber-ducky-mcp` is on your `$PATH`. After `go install github.com/jacaudi/rubber-ducky-mcp@latest`, that's `$GOPATH/bin/rubber-ducky-mcp` — make sure `$GOPATH/bin` is on `$PATH`, or use the absolute path in the `command` field.

## Claude Desktop

`~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) or `%APPDATA%\Claude\claude_desktop_config.json` (Windows):

```json
{
  "mcpServers": {
    "rubber-ducky": {
      "command": "rubber-ducky-mcp"
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
    "rubber-ducky": {
      "command": "rubber-ducky-mcp"
    }
  }
}
```

## VS Code (Continue, Cline, etc.)

Most VS Code MCP-aware extensions use the same `mcp.json` shape:

```json
{
  "mcpServers": {
    "rubber-ducky": {
      "command": "rubber-ducky-mcp"
    }
  }
}
```

## Cursor

`~/.cursor/mcp.json`:

```json
{
  "mcpServers": {
    "rubber-ducky": {
      "url": "http://localhost:3000/mcp"
    }
  }
}
```

(Cursor currently prefers HTTP transport.) Run the server separately with `rubber-ducky-mcp -http :3000`.

## Generic HTTP (any client)

Run the server in HTTP mode and point your client at `/mcp`:

```bash
rubber-ducky-mcp -http :3000
```

```json
{
  "mcpServers": {
    "rubber-ducky": {
      "url": "http://localhost:3000/mcp"
    }
  }
}
```

For browser-based clients, set `ALLOWED_ORIGINS` to permit your origin — see [configuration.md](configuration.md).

## Docker

```bash
docker run -d --name rubber-ducky -p 3000:3000 ghcr.io/jacaudi/rubber-ducky-mcp:latest
```

Then use the HTTP client config above. The image binds to `0.0.0.0` automatically (via `DOCKER=true`); pair it with appropriate firewall rules in production.
