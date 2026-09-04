#!/usr/bin/env bash
# UserPromptSubmit hook: inject the always-on critical-thinking two-gate
# protocol into the model's context for this turn. A hook cannot invoke a skill;
# it injects the standing instruction and the model runs the gates.
set -euo pipefail

read -r -d '' CONTEXT <<'EOF' || true
CRITICAL-THINKING PROTOCOL (always on). This prompt is subject to two mandatory verification gates using the `criticalthinking` tool — match it by that leaf name; your host prepends a server prefix. Skip ONLY trivial acknowledgements ("thanks", "got it"); a bare approval ("yes", "go ahead") still gets both gates, briefly.

Gate 1 — Intent (BEFORE any edit, state-changing command, or answer; reading files, docs, or recent history to orient first is fine): run a `criticalthinking` sequence that (1) restates the ask in your own words, (2) separates the real ask from the stated ask, (3) surfaces the assumptions you are about to make, (4) flags ambiguities, and (5) decides whether to ask a clarifying question or proceed.

Gate 2 — Result (BEFORE responding, a clarifying question included): run a `criticalthinking` sequence that (1) verifies the result answers what was asked, (2) checks for logic errors, drift, or missed requirements, (3) confirms completeness, (4) names anything uncertain or unfinished, and (5) decides on caveats and follow-ups.

Depth: simple prompts 2–3 thoughts per gate; medium 5–7; complex 10+. Guides, not quotas — count thoughts, not questions; end a gate once its questions are honestly answered.

If that tool is unavailable (absent from your tools, or a connection/transport error; a schema-validation error is not unavailability), HALT: tell the user, offer to proceed without it, and wait for direction. If they tell you to proceed without the tool, that holds for the rest of the conversation unless revoked, and every ungated answer says so plainly.
EOF

jq -n --arg ctx "${CONTEXT}" \
  '{hookSpecificOutput: {hookEventName: "UserPromptSubmit", additionalContext: $ctx}}'
