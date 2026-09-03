# Documentation

Reference material for `critical-thinking`. The top-level [README](../README.md) is the landing page; everything below is detail.

| File | Covers |
|---|---|
| [usage.md](usage.md) | Install, MCP-server & CLI-pipe usage, a worked session |
| [configuration.md](configuration.md) | Env vars (`CTHINK_ALLOWED_ORIGINS`, `CTHINK_HTTP_HOST`, `CTHINK_OIDC_*`, `CTHINK_VERBOSE`, `CTHINK_LOG_FORMAT`, `CTHINK_OTEL_ENABLED`), stateless HTTP, endpoints, CORS, auth, observability |
| [clients.md](clients.md) | `mcp.json` snippets for Claude Desktop, Codex CLI, VS Code, Cursor — both stdio and HTTP transports |
| [development.md](development.md) | Building, running tests with `-race`, debugging with MCP Inspector, release workflow |
| [migration.md](migration.md) | Cumulative breaking-change log since the `http-sequential-thinking` Node predecessor |
| [history/](history/) | The review of the TypeScript predecessor that motivated the Go rewrite |
| [../plugins/critical-thinking/](../plugins/critical-thinking/) | The Claude Code plugin — bundles the server install, the two-gate verification skill, and a hook that injects the protocol into every prompt |

For the tool's full input contract (every field and every rule), read the tool description itself — clients receive it on `tools/list`. Code lives in [`internal/thinking/description.go`](../internal/thinking/description.go).
