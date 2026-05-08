# Changelog

## [1.2.1](https://github.com/jacaudi/critical-thinking-plugin/compare/v1.2.0...v1.2.1) (2026-05-08)

### Bug Fixes

* **docs:** drop plugin scaffolding, add Claude Code recipes, auto-bump docker tags ([7438b38](https://github.com/jacaudi/critical-thinking-plugin/commit/7438b38a9c2b2576628f03aa1ee1ce16b4376c9c))

## [1.2.0](https://github.com/jacaudi/critical-thinking-plugin/compare/v1.1.0...v1.2.0) (2026-05-08)

### Features

* **hook:** pin binary version in script; CI auto-bumps on every release ([d1ff595](https://github.com/jacaudi/critical-thinking-plugin/commit/d1ff595cf4d3856f9439887b4e3a3d8c55c2f10a))

## [1.1.0](https://github.com/jacaudi/critical-thinking-plugin/compare/v1.0.0...v1.1.0) (2026-05-08)

### Features

* **ci:** build versioned container image as part of release ([96bdfff](https://github.com/jacaudi/critical-thinking-plugin/commit/96bdfff9ca8c7f498e47372f9cbeb302ba89dc6f))

## 1.0.0 (2026-05-08)

### Bug Fixes

* address final-review findings (idle cleanup, CSRF, preflight headers, migration docs) ([c6c85ba](https://github.com/jacaudi/critical-thinking-plugin/commit/c6c85baec5deab911c7af71daf9578a1d92bd4fd))
* **deps:** update dependency @modelcontextprotocol/sdk to ^1.22.0 ([#10](https://github.com/jacaudi/critical-thinking-plugin/issues/10)) ([f3c908d](https://github.com/jacaudi/critical-thinking-plugin/commit/f3c908d8db168659692bfd3b4ea875a08a2b4acd))
* **deps:** update dependency chalk to ^5.6.2 ([#7](https://github.com/jacaudi/critical-thinking-plugin/issues/7)) ([ea455b8](https://github.com/jacaudi/critical-thinking-plugin/commit/ea455b8204b6c2f8a68ff34c34d8a91abc5b26f0))


### Features

* /health endpoint with status, transport, activeSessions, version ([0f19f78](https://github.com/jacaudi/critical-thinking-plugin/commit/0f19f781d0df674f6ea956f393645cd5980c3d0d))
* add configurable CORS and Docker support ([a459e2b](https://github.com/jacaudi/critical-thinking-plugin/commit/a459e2ba05ead9a1d56cb53092aa885515d9bded))
* add marketplace manifest for plugin discovery ([63efbc4](https://github.com/jacaudi/critical-thinking-plugin/commit/63efbc4edd9b096a8a674e46c3f26a2a335885db))
* CORS middleware with ALLOWED_ORIGINS env var (default empty) ([ba2f69c](https://github.com/jacaudi/critical-thinking-plugin/commit/ba2f69ccc1b8832fe2a39c5be3ceefa5a546e03a))
* graceful shutdown and 1h idle session cleanup ([6c7e4a3](https://github.com/jacaudi/critical-thinking-plugin/commit/6c7e4a3a80cf3cdd368796a1c823f3df89363c10))
* HTTP transport with per-session factory closure ([fa5daf4](https://github.com/jacaudi/critical-thinking-plugin/commit/fa5daf41f6e59c92f2d236e9f4147777024abb96))
* implement Docker build and push action with configurable inputs and metadata extraction ([55a7392](https://github.com/jacaudi/critical-thinking-plugin/commit/55a7392ab0bb25e5d731ce73f2ca38a767f5e857))
* implemented comprehensive unit testing with Jest/TypeScript configuration, extracted SequentialThinkingServer class for testability, created 9 passing unit tests covering all core functionality, and resolved CI/CD pipeline failures ([#1](https://github.com/jacaudi/critical-thinking-plugin/issues/1)) ([ecad678](https://github.com/jacaudi/critical-thinking-plugin/commit/ecad678ed833ac1a1611353c0184ea3f888fc8a6))
* **plugin:** SessionStart hook downloads MCP server binary on install ([60f4173](https://github.com/jacaudi/critical-thinking-plugin/commit/60f4173753f0f0136c6c6c65ab37a4b8e02d3a2a))
* **plugin:** ship .mcp.json so install auto-configures the MCP server ([f9d3e13](https://github.com/jacaudi/critical-thinking-plugin/commit/f9d3e13fcacac716ef6a408df76ad2f29f81a899))
* stdio transport with criticalthinking tool registered ([e749058](https://github.com/jacaudi/critical-thinking-plugin/commit/e74905873a9babca036505e5fc434146a2461124))
* thinking://current resource scoped to per-session snapshot ([a2f5675](https://github.com/jacaudi/critical-thinking-plugin/commit/a2f56757b04f3dcfbfc01561135a35736885c0f8))
* **thinking:** add ThoughtData and ThoughtResponse types ([6ae8a11](https://github.com/jacaudi/critical-thinking-plugin/commit/6ae8a11e3d1d22b54df1cecce5c58494398aeea1))
* **thinking:** add ToolDescription string ([daa9385](https://github.com/jacaudi/critical-thinking-plugin/commit/daa9385d65d9523aa269c2140c776c00ac32b3e1))
* **thinking:** aggregate confidence per branch, separate from trunk ([4866492](https://github.com/jacaudi/critical-thinking-plugin/commit/48664922cd74a9d21940d8b28226273a20662d46))
* **thinking:** polished header variants and dual-line branch footer ([b07a16a](https://github.com/jacaudi/critical-thinking-plugin/commit/b07a16afd69205eeeb0276e4b1e4836304373e87))
* **thinking:** ProcessThought happy path with history append and auto-bump ([712d7f8](https://github.com/jacaudi/critical-thinking-plugin/commit/712d7f8f8ee70bce0a668cb034bb74ecfe7172cb))
* **thinking:** record branches; range-check revisesThought and branchFromThought ([06aca76](https://github.com/jacaudi/critical-thinking-plugin/commit/06aca76bf153f2d84d062945200c48b566b261ab))
* **thinking:** render rubber-duck transcript (pass-1 form) ([a454102](https://github.com/jacaudi/critical-thinking-plugin/commit/a454102d97219553673659d4817f85e8a1976fa1))
* **thinking:** scaffold SequentialThinkingServer with empty state ([2cc63ad](https://github.com/jacaudi/critical-thinking-plugin/commit/2cc63ad69a6073c58c3dc0c3db222518865ccbfd))
* **thinking:** validate confidence range, conditional nextStepRationale, branch both-or-neither ([e651ffd](https://github.com/jacaudi/critical-thinking-plugin/commit/e651ffd8b48fc53f77ee88b1a85dfc8a403e48ef))
* **thinking:** validate required ThoughtData fields ([80b229e](https://github.com/jacaudi/critical-thinking-plugin/commit/80b229e205d471693d3c4a8194f79e62bb411b1b))
