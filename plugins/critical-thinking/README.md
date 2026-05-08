# critical-thinking

A Claude Code plugin that teaches Claude *when* to invoke the `criticalthinking` tool exposed by the [critical-thinking](https://github.com/jacaudi/critical-thinking-plugin) MCP server.

The MCP server documents *how* and *why*. This plugin owns *when*.

## What it does

Ships one skill, `critical-thinking`, that fires as the default discipline for any non-trivial reasoning — in two modes:

1. **Mode 1 — primary thinking.** When you would otherwise reach for silent extended thinking or `mcp__sequential-thinking__sequentialthinking`, route the work through `criticalthinking` instead.
2. **Mode 2 — post-hoc pressure-test.** When the thinking already happened in another channel (silent extended thinking, native chain-of-reasoning, a v1 sequential-thinking session), externalize the review through `criticalthinking` before acting.

Mode 1 is preferred when you control the entry point; Mode 2 is the fallback when the thinking has already happened.

## v2 of sequential-thinking

`criticalthinking` is a strict superset of `mcp__sequential-thinking__sequentialthinking`'s schema. Every v1 field is present with the same semantics. Added on top: required `confidence`, `assumptions`, `critique`, `counterArgument`, and `nextStepRationale` fields, plus per-call and per-branch confidence tracking. The discipline isn't optional — it's enforced by the schema.

When this plugin is installed, prefer `criticalthinking` over the v1 tool for every use case the v1 tool covers.

## Dependency

This plugin requires the `critical-thinking` MCP server to be installed and configured separately. The skill calls the server's `criticalthinking` tool — without the server, the skill has nothing to invoke.

- Server source: https://github.com/jacaudi/critical-thinking-plugin
- Server configuration: see `docs/configuration.md` in the server repo for stdio and HTTP transport setup.

## Install

This plugin is distributed via a Claude Code plugin marketplace. The marketplace that lists it is hosted separately; consult that marketplace's documentation for the exact `/plugin marketplace add` and `/plugin install` commands.

Once installed, the skill is available to Claude immediately — no further configuration.

## Usage

Two invocation modes, both supported out of the box:

**Auto-invocation.** Claude triggers the skill on its own whenever non-trivial reasoning is involved — driven by the skill's description. No user action required.

**Manual invocation.** Type:

```
/critical-thinking:critical-thinking
```

…or use natural language: "think this through," "critique this," "pressure-test that conclusion," "branch on this," "revise that earlier thought." Claude invokes the skill on whatever subject you named.

## Scope

This plugin contains exactly one skill. No hooks, no slash commands beyond the auto-generated plugin-namespaced one, no agents.

## See also

- Top-level project: [Critical Thinking Plugin](../../README.md)
- MCP server source: https://github.com/jacaudi/critical-thinking-plugin

## License

See the parent repository for license terms.
