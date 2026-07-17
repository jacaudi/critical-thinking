# Configuration

## Environment variables

| Env var | Default | Purpose |
|---|---|---|
| `CTHINK_ALLOWED_ORIGINS` | (empty) | Comma-separated list of browser origins permitted to call `/mcp`. Wired into both the outer CORS layer and the SDK's CSRF protection (`http.CrossOriginProtection.AddTrustedOrigin`). Default rejects all browser origins. Non-browser callers (no `Origin` / no `Sec-Fetch-Site` header) are unaffected. |
| `CTHINK_HTTP_HOST` | `127.0.0.1` | Host the HTTP server binds to. Set to `0.0.0.0` to bind all interfaces (the published Docker image sets this). |
| `CTHINK_OIDC_ISSUER` | (empty) | OIDC issuer URL for bearer-token auth on `/mcp`. **Empty disables auth** (default; preserves prior behavior). When set, every `/mcp` request must carry a valid `Authorization: Bearer <jwt>`. The server performs OIDC discovery at startup and **fails to start** if the issuer is unreachable. |
| `CTHINK_OIDC_AUDIENCE` | (empty) | Expected `aud` claim. **Required when `CTHINK_OIDC_ISSUER` is set** — the server refuses to start otherwise (an empty audience would disable audience validation). |
| `CTHINK_VERBOSE` | `false` | Enables debug logging (and the stdio JSON-RPC frame trace). Env equivalent of `--verbose`; the flag overrides it. |
| `CTHINK_LOG_FORMAT` | `text` | Log handler format: `text` or `json`. Env equivalent of `--log-format`; the flag overrides it. |

All config is read through Viper with the `CTHINK_` prefix. For the logging
settings, precedence is **flag > env > default** (e.g. `--log-format` overrides
`CTHINK_LOG_FORMAT` overrides `text`).

## Transports

### Stdio (default)

```bash
critical-thinking serve
```

One process serves one session. There is no cross-stream isolation concern because there is no second stream — the process IS the session. Use this for direct integration with MCP hosts (Claude Desktop, Codex CLI, VS Code).

### Streamable HTTP

```bash
critical-thinking serve --http :3000
```

The HTTP server binds to `127.0.0.1` by default (set `CTHINK_HTTP_HOST=0.0.0.0` to bind all interfaces). Each session gets its own `*mcp.Server` with its own `SequentialThinkingServer`, constructed inside a factory closure — there is no map keyed by session ID anywhere, by design. The closure scope is the cross-session isolation invariant.

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
| `/mcp` | `POST`, `GET`, `DELETE` | Main MCP endpoint (Streamable HTTP) |
| `/health` | `GET` | Returns `{status, transport, sessionsCreated, version}`. `sessionsCreated` is a **lifetime** counter of sessions ever created in this process; it is NOT pruned when the SDK closes idle sessions. Treat it as a creation counter, not an active-session gauge. |

## Session lifecycle

Sessions are in-memory only. Idle sessions expire after **1 hour**, enforced by the SDK via `StreamableHTTPOptions.SessionTimeout`. When the SDK closes a session, the bound `*mcp.Server` (and the `*SequentialThinkingServer` it captures) becomes unreachable and is released for GC.

There is no callback fired when the SDK closes a session, so the in-process registry that powers `/health.sessionsCreated` drifts upward — that's intentional. If you need an accurate active-session count, get it from your reverse proxy or load balancer, not from this server.

## CORS and CSRF

When `CTHINK_ALLOWED_ORIGINS` is empty, browser requests with an `Origin` header are rejected with HTTP 403. Non-browser clients (no `Origin`, no `Sec-Fetch-Site`) bypass the check entirely. When set, matching origins receive `Access-Control-Allow-Origin: <origin>`, `Access-Control-Allow-Credentials: true`, `Access-Control-Expose-Headers: mcp-session-id`, and a `Vary: Origin` header for cache-poisoning mitigation.

The same origin list is registered with the SDK's CSRF protection (`http.CrossOriginProtection.AddTrustedOrigin`) so the SDK's same-origin policy doesn't double-reject permitted browser callers.

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

Authentication is orthogonal to CORS and the SDK's `CrossOriginProtection`: all three apply
independently.
