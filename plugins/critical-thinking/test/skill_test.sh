#!/usr/bin/env bash
# Pins the sentences in SKILL.md that the hook, the tool description, and the
# design rely on; rejects the vocabulary of the stateful era; and checks that
# every fact the hook and the skill both state appears on both surfaces.
set -uo pipefail
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SKILL="${HERE}/../skills/critical-thinking/SKILL.md"
HOOK="${HERE}/../hooks/activate.sh"
[[ -f "${SKILL}" ]] || { echo "FAIL - ${SKILL} not found"; exit 1; }
[[ -f "${HOOK}" ]] || { echo "FAIL - ${HOOK} not found"; exit 1; }

# Skill-only sentences.
for needle in "The tool is stateless" "Your own context is the record" \
  "keeps no history of its own" "the discipline the tool enforces" \
  "waiver holds for the rest of the conversation" "before any edit, state-changing command, or answer"; do
  grep -qF -- "${needle}" "${SKILL}" || { echo "FAIL - SKILL.md missing '${needle}'"; exit 1; }
done
# Stateful-era vocabulary, in any casing. "session"/"episode" have no
# legitimate use in this file; Claude Code's own SessionStart hook is described
# in the README, not here.
for stale in "session" "episode" "before doing anything else" "before work" "the schema enforces"; do
  if grep -qi -- "${stale}" "${SKILL}"; then echo "FAIL - SKILL.md still says '${stale}'"; exit 1; fi
done

# Facts both surfaces state must appear on both. Single source of that list.
ctx="$(bash "${HOOK}" | jq -r '.hookSpecificOutput.additionalContext')"
for shared in "leaf name" "state-changing command" "orient" "go ahead" \
  "restate" "real ask" "assumptions" "ambiguit" "clarifying question" \
  "logic errors" "completeness" "uncertain or unfinished" "caveats" \
  "2–3" "5–7" "10+" "not quotas" "count thoughts, not questions" \
  "schema-validation error is not unavailability" "wait for direction" "proceed without" "says so plainly"; do
  grep -qiF -- "${shared}" "${SKILL}" || { echo "FAIL - SKILL.md lacks shared fact '${shared}'"; exit 1; }
  printf '%s' "${ctx}" | grep -qiF -- "${shared}" || { echo "FAIL - hook lacks shared fact '${shared}'"; exit 1; }
done
echo "ok - SKILL.md carries the stateless contract and agrees with the hook"
