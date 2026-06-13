#!/usr/bin/env bash
# UserPromptSubmit hook: inject the always-on critical-thinking two-gate
# protocol into the model's context for this turn. A hook cannot invoke a skill;
# it injects the standing instruction and the model runs the gates.
set -euo pipefail

read -r -d '' CONTEXT <<'EOF' || true
CRITICAL-THINKING PROTOCOL (always on). This prompt is subject to two mandatory verification gates using the `criticalthinking` tool. Skip ONLY trivial acknowledgements ("thanks", "got it").

Gate 1 — Intent (BEFORE any work): run a `criticalthinking` session to restate the ask in your own words, separate the real ask from the stated ask, surface the assumptions you are about to make, and flag ambiguities. Only then begin work.

Gate 2 — Result (BEFORE responding): run a `criticalthinking` session to verify the result actually answers what was asked, check for drift or missed requirements, confirm completeness, and decide on caveats.

Scale gate depth to complexity (simple: 2-3 thoughts; medium: 5-7; complex: 10+). If the `criticalthinking` tool is unavailable (absent from tools, or a connection/transport error — not a schema error), HALT and tell the user; do not silently proceed.
EOF

jq -n --arg ctx "${CONTEXT}" \
  '{hookSpecificOutput: {hookEventName: "UserPromptSubmit", additionalContext: $ctx}}'
