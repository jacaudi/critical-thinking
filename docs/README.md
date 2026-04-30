# Documentation

Reference material for `rubber-ducky-mcp`. The top-level [README](../README.md) is the landing page; everything below is detail.

| File | Covers |
|---|---|
| [configuration.md](configuration.md) | Env vars (`ALLOWED_ORIGINS`, `DOCKER`, `DISABLE_THOUGHT_LOGGING`), HTTP endpoints, session lifecycle, idle timeout |
| [clients.md](clients.md) | `mcp.json` snippets for Claude Desktop, Codex CLI, VS Code, Cursor — both stdio and HTTP transports |
| [development.md](development.md) | Building, running tests with `-race`, debugging with MCP Inspector, release workflow |
| [migration.md](migration.md) | Cumulative breaking-change log since the `http-sequential-thinking` Node predecessor |

For the tool's full input contract (every field, every length cap, every rule), read the tool description itself — clients receive it on `tools/list`. Code lives in [`internal/thinking/description.go`](../internal/thinking/description.go).
