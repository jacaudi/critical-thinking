#!/usr/bin/env bash
# Verifies activate.sh emits valid UserPromptSubmit hook JSON carrying the
# two-gate protocol text.
set -uo pipefail
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT="${HERE}/../hooks/activate.sh"

out="$(bash "${SCRIPT}")" || { echo "FAIL - script errored"; exit 1; }

# Valid JSON?
echo "${out}" | jq -e . >/dev/null || { echo "FAIL - not valid JSON"; exit 1; }
# Correct event name?
echo "${out}" | jq -e '.hookSpecificOutput.hookEventName == "UserPromptSubmit"' >/dev/null \
  || { echo "FAIL - wrong/absent hookEventName"; exit 1; }
# additionalContext mentions both gates and the tool?
ctx="$(echo "${out}" | jq -r '.hookSpecificOutput.additionalContext')"
for needle in "Gate 1" "Gate 2" "criticalthinking" "HALT"; do
  case "${ctx}" in *"${needle}"*) ;; *) echo "FAIL - context missing '${needle}'"; exit 1;; esac
done
echo "ok - activate.sh emits valid protocol JSON"
