#!/usr/bin/env bash
# Asserts hooks.json is valid JSON and the SessionStart matcher is narrowed to
# startup|resume (issue #77 finding 2 — clear/compact cannot change the on-disk
# plugin, so the install/version check must not run there).
set -uo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HOOKS="${HERE}/../hooks/hooks.json"
PASS=0; FAIL=0

if jq -e . "${HOOKS}" >/dev/null 2>&1; then echo "ok   - hooks.json is valid JSON"; PASS=$((PASS+1));
else echo "FAIL - hooks.json invalid JSON"; FAIL=$((FAIL+1)); fi

matcher="$(jq -r '.hooks.SessionStart[0].matcher' "${HOOKS}")"
if [[ "${matcher}" == "startup|resume" ]]; then echo "ok   - SessionStart matcher narrowed"; PASS=$((PASS+1));
else echo "FAIL - SessionStart matcher is '${matcher}', expected 'startup|resume'"; FAIL=$((FAIL+1)); fi

echo "---"; echo "PASS=${PASS} FAIL=${FAIL}"
[[ "${FAIL}" -eq 0 ]]
