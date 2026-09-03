#!/usr/bin/env bash
# Verifies activate.sh emits valid UserPromptSubmit hook JSON carrying the
# two-gate protocol, with the directional facts a model must not lose.
# Facts shared with SKILL.md are checked on both surfaces by skill_test.sh.
set -uo pipefail
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT="${HERE}/../hooks/activate.sh"

out="$(bash "${SCRIPT}")" || { echo "FAIL - script errored"; exit 1; }

# Valid JSON?
echo "${out}" | jq -e . >/dev/null || { echo "FAIL - not valid JSON"; exit 1; }
# Correct event name?
echo "${out}" | jq -e '.hookSpecificOutput.hookEventName == "UserPromptSubmit"' >/dev/null \
  || { echo "FAIL - wrong/absent hookEventName"; exit 1; }
ctx="$(echo "${out}" | jq -r '.hookSpecificOutput.additionalContext')"
# Directional literals: the order of gate vs action, and the halt exit.
for needle in "Gate 1 — Intent (BEFORE any edit, state-changing command, or answer" \
  "Gate 2 — Result (BEFORE responding" "HALT" "offer to proceed without it" "wait for direction" \
  "holds for the rest of the conversation" "(5) decides" "(5) decides on caveats"; do
  case "${ctx}" in *"${needle}"*) ;; *) echo "FAIL - context missing '${needle}'"; exit 1;; esac
done
# No vocabulary from the stateful era.
for stale in "episode" "session" "before any work"; do
  if printf '%s' "${ctx}" | grep -qi -- "${stale}"; then echo "FAIL - context still says '${stale}'"; exit 1; fi
done
echo "ok - activate.sh emits valid protocol JSON"
