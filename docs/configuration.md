# Configuration

## Environment variables

| Env var | Default | Purpose |
|---|---|---|
| `CTHINK_ALLOWED_ORIGINS` | (empty) | Comma-separated list of browser origins permitted to call `/mcp`. Wired into both the outer CORS allow-list and Go's CSRF layer (`http.CrossOriginProtection.AddTrustedOrigin`) that wraps the MCP handler. Default rejects all browser origins. Non-browser callers (no `Origin` / no `Sec-Fetch-Site` header) are unaffected. |
| `CTHINK_HTTP_HOST` | `127.0.0.1` | Host the HTTP server binds to. Set to `0.0.0.0` to bind all interfaces (the published Docker image sets this). |
| `CTHINK_OIDC_ISSUER` | (empty) | OIDC issuer URL for bearer-token auth on `/mcp`. **Empty disables auth** (default; preserves prior behavior). When set, every `/mcp` request must carry a valid `Authorization: Bearer <jwt>`. The server performs OIDC discovery at startup and **fails to start** if the issuer is unreachable. |
| `CTHINK_OIDC_AUDIENCE` | (empty) | Expected `aud` claim. **Required when `CTHINK_OIDC_ISSUER` is set** — the server refuses to start otherwise (an empty audience would disable audience validation). |
| `CTHINK_VERBOSE` | `false` | Enables debug logging (and the stdio JSON-RPC frame trace). Env equivalent of `--verbose`; the flag overrides it. |
| `CTHINK_LOG_FORMAT` | `text` | Log handler format: `text` or `json`. Env equivalent of `--log-format`; the flag overrides it. |
| `CTHINK_OTEL_ENABLED` | `false` | Install OpenTelemetry tracer/meter providers (OTLP/HTTP exporters) for `serve`. When false (default), all instrumentation is no-op and no telemetry dependency is active at runtime. |

All config is read through Viper with the `CTHINK_` prefix. For the logging
settings, precedence is **flag > env > default** (e.g. `--log-format` overrides
`CTHINK_LOG_FORMAT` overrides `text`).

## Transports

### Stdio (default)

```bash
critical-thinking serve
```

One process, one stdin/stdout pair. The server keeps no state, so nothing accumulates over the life of the process. Use this for direct integration with MCP hosts (Claude Desktop, Codex CLI, VS Code).

### Streamable HTTP

```bash
critical-thinking serve --http :3000
```

The HTTP server binds to `127.0.0.1` by default (set `CTHINK_HTTP_HOST=0.0.0.0` to bind all interfaces). It runs in the MCP SDK's **stateless mode**, which is how the SDK serves MCP protocol `2026-07-28` (the sessionless model): the server neither issues nor reads `Mcp-Session-Id`, each `POST /mcp` is complete in itself, and `GET`/`DELETE /mcp` answer `405 Method Not Allowed`. Older clients that still send `initialize` (protocol `2025-11-25` and earlier) are answered normally by a temporary per-request session. One `*mcp.Server` serves every request, so replicas behind a load balancer need no session affinity. Request bodies are capped at the SDK default of 4 MiB (`413` beyond that).

If a client insists on the pre-sessionless behaviour (session ids echoed back, `DELETE` accepted), the SDK offers a temporary compatibility switch: run the server with `MCPGODEBUG=allowsessionsinstateless=1`. It is slated for removal in go-sdk v1.9.0; treat it as a bridge, not a setting.

## Logging

All logs go to **stderr** via `log/slog`. In stdio mode stdout is the JSON-RPC
channel, and in `cli` / `schema` / `version` it carries command output — so nothing
but protocol/output ever reaches stdout.

| Flag | Default | Effect |
|---|---|---|
| `--verbose` | off | Sets the log level to `Debug`. In stdio mode it also traces every JSON-RPC frame to stderr (off by default). |
| `--log-format` | `text` | Handler format: `text` (human-readable) or `json` (structured). Any other value exits non-zero with an error on stderr. |

Both are persistent root flags, so they work before or after any subcommand
(`critical-thinking --verbose serve`, `critical-thinking serve --log-format=json`).
They also read from `CTHINK_VERBOSE` / `CTHINK_LOG_FORMAT` (flag > env > default).
The library engine (`internal/thinking`) emits no logs — it returns errors and lets
the caller decide.

## HTTP endpoints

| Path | Methods | Purpose |
|---|---|---|
| `/mcp` | `POST` (`OPTIONS` preflight via CORS; `GET`/`DELETE` → `405`) | Main MCP endpoint (Streamable HTTP, stateless) |
| `/health` | `GET` | Returns `{status, transport, version}`. |

A `POST /mcp` must carry `Content-Type: application/json` and an `Accept` header that lists both `application/json` and `text/event-stream`; the SDK answers `415`/`400` otherwise. MCP clients do this automatically — it only matters for hand-written `curl` calls.

## CORS and CSRF

When `CTHINK_ALLOWED_ORIGINS` is empty, browser requests with an `Origin` header are rejected with HTTP 403. Non-browser clients (no `Origin`, no `Sec-Fetch-Site`) bypass the check entirely. When set, matching origins receive `Access-Control-Allow-Origin: <origin>`, `Access-Control-Allow-Credentials: true`, and a `Vary: Origin` header for cache-poisoning mitigation. Every response advertises `Access-Control-Allow-Methods: POST, OPTIONS` and `Access-Control-Allow-Headers: Content-Type, Authorization, Mcp-Protocol-Version, Mcp-Method, Mcp-Name, mcp-session-id` — `Authorization` so browser clients can present OIDC bearer tokens, `Mcp-Method` and `Mcp-Name` because protocol `2026-07-28` sends them on every request, `mcp-session-id` so preflights from older clients that still send it succeed (the server ignores the header). The allow-list wraps the whole mux, so a browser `Origin` outside it is refused on `/health` too; probes without an `Origin` are unaffected.

The same origin list is registered with Go's `http.CrossOriginProtection`, which wraps the MCP handler (the SDK's own cross-origin option has been deprecated since go-sdk v1.6.1 and is not used), so a permitted browser origin is not double-rejected by a same-origin policy.

## Authentication (OIDC bearer tokens)

The HTTP transport optionally authenticates `/mcp` with OIDC bearer tokens. It is **disabled by
default**: with `CTHINK_OIDC_ISSUER` unset, behavior is unchanged and `/mcp` is unauthenticated
(the server logs a startup warning saying so).

When `CTHINK_OIDC_ISSUER` is set (and `CTHINK_OIDC_AUDIENCE` provided):

- Every `/mcp` request must carry `Authorization: Bearer <jwt>`. The token's **signature, issuer,
  audience (`aud`), and expiry** are validated against the issuer's published JWKS. Any failure
  returns `401` with a `WWW-Authenticate: Bearer` challenge; the internal reason is never sent to
  the client.
- `/health` stays **unauthenticated** so liveness/readiness probes keep working.
- `OPTIONS` preflight is unaffected — CORS short-circuits it before auth runs.

> **Token requirements (read before configuring your IdP):**
> - The presented bearer token **must be a signed JWT** whose `aud` claim **equals**
>   `CTHINK_OIDC_AUDIENCE`. Request tokens for this audience from your IdP accordingly (many IdPs set
>   `aud` from a "resource"/"audience"/"scope" parameter on the token request). A token whose `aud`
>   is a different resource URI or client ID will be rejected with `401`.
> - **Opaque (non-JWT) access tokens are not supported.** Some IdPs issue opaque bearer tokens that
>   can only be validated via a token-introspection endpoint (RFC 7662). This design validates JWTs
>   against the issuer's JWKS only — it does not call introspection — so opaque tokens will not
>   verify. Configure your IdP to issue JWT access/ID tokens for this audience.

**Security posture:**

- **Fail fast at startup.** If `CTHINK_OIDC_ISSUER` is set but `CTHINK_OIDC_AUDIENCE` is empty, or the
  issuer's discovery endpoint is unreachable at boot, the server **exits non-zero** rather than
  serving. A server that cannot authenticate should not start. (In Kubernetes, rely on the restart
  policy / readiness ordering if the IdP and this server start together.)
- **Fail closed at runtime.** JWKS keys are cached and rotated automatically, so a brief IdP outage
  does not break verification of already-seen signing keys. A token that cannot be cryptographically
  verified is always rejected — the server never falls back to accepting unverified tokens.

Authentication is orthogonal to CORS and Go's `CrossOriginProtection` layer: all three apply
independently.

## Observability (OpenTelemetry)

Set `CTHINK_OTEL_ENABLED=true` to enable tracing and metrics for `serve`
(both stdio and HTTP transports). Everything else is standard OTel SDK
environment configuration — no `CTHINK_OTEL_*` variants exist:

- `OTEL_EXPORTER_OTLP_ENDPOINT` — collector base URL. Default
  `https://localhost:4318` (OTLP/HTTP). Most local collectors speak
  plaintext: use an `http://` scheme (e.g. `http://localhost:4318`) or set
  `OTEL_EXPORTER_OTLP_INSECURE=true`, otherwise you will see TLS handshake
  warnings on stderr.
- `OTEL_TRACES_SAMPLER` / `OTEL_TRACES_SAMPLER_ARG`, `OTEL_SERVICE_NAME`,
  `OTEL_RESOURCE_ATTRIBUTES`, and the rest of the standard set are honored
  by the OTel SDK directly.

Export failures (e.g. no collector running) are logged as warnings on
stderr and are never fatal; stdout is never touched (in stdio mode it is
the JSON-RPC channel).

Signals emitted:

| Signal | What |
|---|---|
| Span `mcp.<method>` | One server span per JSON-RPC method on both transports; `tools/call` spans carry `mcp.tool.name` and bounded domain attributes (`ct.thought_number`, `ct.total_thoughts`, `ct.confidence`, `ct.is_revision`, `ct.is_branch`). |
| HTTP server spans/metrics | From `otelhttp` around `/mcp` (`/health` is excluded). |
| `ct.mcp.calls` | Counter by `mcp.method` + `ct.outcome` (`ok`/`error`/`tool_error`). |
| `ct.mcp.duration` | Histogram (seconds) by `mcp.method`. |

Privacy: the content of `thought`, `assumptions`, `critique`,
`counterArgument`, and `nextStepRationale` is never attached to any span or
metric (enforced by a test over every recorded span), and the only metric
labels are the fixed `mcp.method` and `ct.outcome` values.
