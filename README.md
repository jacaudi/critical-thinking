# Rubber Ducky MCP

A Model Context Protocol server for **critical, narrated, sequential thinking**. A rubber duck you talk to while you think, one step at a time, with required confidence calibration and adversarial self-critique on every thought.

This server fuses three disciplines:

1. **Sequential thinking** — break problems into ordered, numbered steps; revise; branch.
2. **Rubber-duck narration** — explain each thought out loud, in first-person, to an imagined listener.
3. **Critical self-examination** — every thought is paired with confidence, assumptions, critique, and a counter-argument.

The single tool is `criticalthinking`. Every call must include the critical-thinking fields — there is no opt-out, by design. See the tool description for the full contract.

## Install

### Go

```bash
go install github.com/jacaudi/rubber-ducky-mcp@latest
# binary lands at $GOPATH/bin/rubber-ducky-mcp
```

### Docker

```bash
docker run --rm -p 3000:3000 ghcr.io/jacaudi/rubber-ducky-mcp:latest
```

## Run

### Stdio (default)

```bash
rubber-ducky-thinking
```

Use this for direct integration with MCP hosts (Claude Desktop, Codex CLI, VS Code).

### Streamable HTTP

```bash
rubber-ducky-thinking -http :3000
```

Endpoints:

- `POST/GET/DELETE /mcp` — main MCP endpoint.
- `GET /health` — `{status, transport, sessionsCreated, version}`. `sessionsCreated` is a lifetime counter of sessions ever created in this process; it is NOT pruned when the SDK closes idle sessions.

## Configuration

| Env var | Default | Purpose |
|---|---|---|
| `ALLOWED_ORIGINS` | (empty) | Comma-separated list of browser origins permitted to call `/mcp`. Wired into both the outer CORS layer and the SDK's CSRF protection (`http.CrossOriginProtection.AddTrustedOrigin`). Default rejects all browser origins. Non-browser callers (no `Origin` / no `Sec-Fetch-Site` header) are unaffected. |
| `DOCKER` | unset | When `true`, HTTP server binds to `0.0.0.0` instead of `127.0.0.1`. Set automatically in the published Docker image. |
| `DISABLE_THOUGHT_LOGGING` | unset | Reserved for the future structured-log gate. The current server emits no per-thought logs by default. |

Sessions are in-memory only; idle sessions expire after 1 hour (enforced by the SDK via `StreamableHTTPOptions.SessionTimeout`).

## Migrating from `http-sequential-thinking`

This is the Go successor to `jacaudi/http-sequential-thinking`. Breaking changes:

- **Tool renamed:** `sequentialthinking` → `criticalthinking`. Update `mcp.json` references.
- **Required new fields:** every call must send `confidence`, `assumptions`, `critique`, `counterArgument`. Calls missing these fail with `IsError: true`.
- **`nextStepRationale` required when `nextThoughtNeeded: true`.**
- **Binary renamed:** `http-sequential-thinking` → `rubber-ducky-thinking`. Update `mcp.json` `command` fields and any shell-script references.
- **Server name renamed:** `sequential-thinking-server` → `rubber-ducky-thinking`.
- **Web UI removed.** Use MCP Inspector or `curl` for manual testing.
- **CORS default tightened.** Set `ALLOWED_ORIGINS` explicitly to allow browser clients.
- **Length caps on critical fields (server-side).** `critique` ≤ 280 chars, `counterArgument` ≤ 280, each `assumptions[i]` ≤ 200, `nextStepRationale` ≤ 200 (only enforced when `nextThoughtNeeded=true`). Caps are rune-counted and force one-sentence-per-field discipline; padded prose returns `IsError: true`.
- **`thoughtNumber` and `totalThoughts` are now optional after the first thought.** Omit `thoughtNumber` to let the server auto-assign the next sequential position (trunk: history+1; branch: branch-depth, i.e. position within the branch — *not* a global ordinal). Omit `totalThoughts` to inherit the most recent *trunk* thought's value (branch thoughts don't contaminate the inheritance). The first trunk thought of a session must still include `totalThoughts`. Sending them explicitly is still accepted and overrides. If your client treats stored `thoughtNumber` on branch thoughts as a global ordinal, keep sending it explicitly.
- **Response no longer echoes `thoughtNumber`, `totalThoughts`, or `nextThoughtNeeded`.** Callers already have these — the response now contains only `branches`, `thoughtHistoryLength`, `sessionConfidence`, and `branchConfidences` (when present). Read the full per-thought state from the `thinking://current` resource if you need it.

## Development

```bash
go test -race ./...
go vet ./...
gofmt -d .
go build -ldflags "-X main.version=$(git describe --tags --always)" -o rubber-ducky-thinking .
```

## License

MIT.
