#!/usr/bin/env node
// Bumps `ghcr.io/jacaudi/critical-thinking:vX.Y.Z` examples in user-facing docs
// to the next release version. Invoked by semantic-release via @semantic-release/exec
// during the prepare phase: `node scripts/bump-docker-tags.mjs ${nextRelease.version}`.

import { readFileSync, writeFileSync } from "node:fs";

const version = process.argv[2];
if (!version || !/^\d+\.\d+\.\d+/.test(version)) {
  console.error(`bump-docker-tags: expected a semver argument, got: ${version}`);
  process.exit(1);
}

const targets = ["README.md", "docs/clients.md"];
const pattern = /(ghcr\.io\/jacaudi\/critical-thinking:v)\d+\.\d+\.\d+/g;
const replacement = `$1${version}`;

let changed = 0;
for (const path of targets) {
  const before = readFileSync(path, "utf8");
  const after = before.replace(pattern, replacement);
  if (after !== before) {
    writeFileSync(path, after);
    changed++;
    console.log(`bump-docker-tags: updated ${path} -> v${version}`);
  }
}
console.log(`bump-docker-tags: ${changed} file(s) changed`);
