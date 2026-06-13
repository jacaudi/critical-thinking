# critical-thinking (Claude Code plugin)

Installs the [`critical-thinking`](https://github.com/jacaudi/critical-thinking) MCP server, ships the **two-gate critical-thinking verification** skill, and force-activates that skill on every prompt.

## What it does

1. **Installs the MCP server.** A `SessionStart` hook downloads the pinned server binary for your OS/arch into the plugin's `bin/` directory (no Go toolchain needed). The bundled `.mcp.json` launches it over stdio (`critical-thinking serve`).
2. **Adds a skill.** `critical-thinking` runs two verification gates per substantive prompt: **Gate 1** (intent) before work, **Gate 2** (result) before responding — both via the `criticalthinking` tool.
3. **Always on.** A `UserPromptSubmit` hook injects the two-gate protocol into every turn (trivial acknowledgements exempt).

## Install

Install the plugin through your Claude Code plugin marketplace / path. On the next session start the install hook fetches the binary; no manual step.

The server binary version is pinned in `hooks/install-binary.sh` (`EXPECTED_VERSION`) and tracks server releases automatically.

## HTTP transport (optional)

The default is stdio. To use Streamable HTTP instead:

### Local

Run the server yourself and point the MCP entry at the URL (replace the bundled stdio `.mcp.json` `mcpServers` entry):

```bash
critical-thinking serve --http :3000
```

```json
{ "mcpServers": { "critical-thinking": { "url": "http://localhost:3000/mcp" } } }
```

### Remote

Point at a remote server URL:

```json
{ "mcpServers": { "critical-thinking": { "url": "https://your-host.example/mcp" } } }
```

For browser-origin or bind-host configuration, see the server's
[`docs/configuration.md`](https://github.com/jacaudi/critical-thinking/blob/main/docs/configuration.md)
(`CTHINK_ALLOWED_ORIGINS`, `CTHINK_HTTP_HOST`).

## Disabling the always-on hook

The activation hook is always-on by default. To fall back to description-driven
auto-invocation (the model decides when to run the skill) without the per-turn
forcing, remove the `UserPromptSubmit` block from `hooks/hooks.json` in your
installed copy, or disable the plugin's hooks via your Claude Code settings. The
skill itself still loads and auto-triggers from its description.

## License

[MIT](https://github.com/jacaudi/critical-thinking/blob/main/LICENSE).
