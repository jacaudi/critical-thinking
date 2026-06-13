#!/usr/bin/env node
// Bumps EXPECTED_VERSION in the plugin's install-binary.sh to the next release
// version. Invoked by semantic-release (@semantic-release/exec prepareCmd):
//   node scripts/bump-plugin-version.mjs ${nextRelease.version}
// A second arg overrides the target file (used by tests).

import { readFileSync, writeFileSync } from "node:fs";

const version = process.argv[2];
if (!version || !/^\d+\.\d+\.\d+/.test(version)) {
  console.error(`bump-plugin-version: expected a semver argument, got: ${version}`);
  process.exit(1);
}

const target =
  process.argv[3] || "plugins/critical-thinking/hooks/install-binary.sh";
const tag = `v${version}`;
const pattern = /^EXPECTED_VERSION="v\d+\.\d+\.\d+"/m;

const before = readFileSync(target, "utf8");
if (!pattern.test(before)) {
  console.error(`bump-plugin-version: no EXPECTED_VERSION line in ${target}`);
  process.exit(1);
}
const after = before.replace(pattern, `EXPECTED_VERSION="${tag}"`);
if (after !== before) {
  writeFileSync(target, after);
  console.log(`bump-plugin-version: ${target} -> ${tag}`);
} else {
  console.log(`bump-plugin-version: ${target} already at ${tag}`);
}
