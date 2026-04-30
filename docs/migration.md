# Migration

Cumulative breaking-change log for `rubber-ducky-mcp`. Most recent changes first.

## From `0.6.x` (post-rewrite, prior to length-cap and optional-field work)

### Length caps on critical fields

Server-side rune-counted maxLength on the four critical-thinking fields. Over-cap requests return `IsError: true`.

| Field | Cap (runes) | Notes |
|---|---:|---|
| `critique` | 280 | Always enforced |
| `counterArgument` | 280 | Always enforced |
| `assumptions[i]` | 200 | Per-entry |
| `nextStepRationale` | 200 | Only enforced when `nextThoughtNeeded=true` |

The caps are intentionally tight. They force one-tight-sentence-per-field discipline; padded prose returns an error rather than being silently accepted. If a critique genuinely needs more than 280 chars, split the thinking across two `criticalthinking` calls — that's the design intent.

### `thoughtNumber` and `totalThoughts` are now optional after the first thought

- Omit `thoughtNumber` to let the server auto-assign:
  - **Trunk** thoughts: `len(history)+1`.
  - **Branch** thoughts (when `branchFromThought` and `branchId` are set): position within the branch (1, 2, 3, …) — **not** a global ordinal.
  - **Revisions** (when `isRevision` and `revisesThought` are set): next trunk slot.
- Omit `totalThoughts` to inherit the most recent **trunk** thought's value. Branch thoughts are explicitly skipped during inheritance so a branch's auto-bumped `totalThoughts` cannot contaminate the trunk's running estimate.
- The first trunk thought of a session must still send `totalThoughts` explicitly; omitting it returns `IsError: true`.
- Sending values explicitly is still accepted and overrides — useful for unambiguous revisions or when a client treats `thoughtNumber` as a global ordinal.

### Response no longer echoes `thoughtNumber` / `totalThoughts` / `nextThoughtNeeded`

The caller already sent these — echoing them was pure redundancy. The response now contains only:

```json
{
  "branches": [...],
  "thoughtHistoryLength": N,
  "sessionConfidence": 0.X,
  "branchConfidences": { ... }
}
```

Read the full per-thought state from the `thinking://current` resource if needed.

### Binary, server name, and log line renamed to `rubber-ducky-mcp`

For consistency with the Go module path. The Docker image build artifact and the MCP `Implementation.Name` field are now `rubber-ducky-mcp` (was `rubber-ducky-thinking`). Update `mcp.json` `command` fields and any shell-script references. Client-side server *aliases* (the key under `mcpServers`) are user-controlled and unaffected.

## From `http-sequential-thinking` (Node predecessor)

This Go server is the successor to `jacaudi/http-sequential-thinking`. The original differences:

- **Tool renamed:** `sequentialthinking` → `criticalthinking`. Update `mcp.json` references.
- **Required new fields:** every call must send `confidence`, `assumptions`, `critique`, `counterArgument`. Calls missing these fail with `IsError: true`.
- **`nextStepRationale` required when `nextThoughtNeeded: true`.**
- **Web UI removed.** Use [MCP Inspector](https://github.com/modelcontextprotocol/inspector) or `curl` for manual testing.
- **CORS default tightened.** Set `ALLOWED_ORIGINS` explicitly to allow browser clients.

The original `http-sequential-thinking` had no notion of confidence calibration, assumptions, critique, or counter-argument. That's the deliberate philosophical break: the original tool was infrastructure for sequential prompting; this one is infrastructure for critical sequential prompting.
