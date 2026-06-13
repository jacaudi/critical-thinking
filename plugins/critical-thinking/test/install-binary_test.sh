#!/usr/bin/env bash
# Tests for install-binary.sh. Each case runs the script in an isolated fake
# CLAUDE_PLUGIN_ROOT with curl/tar stubbed on PATH so no network is touched.
set -uo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT="${HERE}/../hooks/install-binary.sh"
PASS=0
FAIL=0

check() { # desc, expected_rc, actual_rc
  if [[ "$2" == "$3" ]]; then echo "ok   - $1"; PASS=$((PASS+1));
  else echo "FAIL - $1 (expected rc=$2 got rc=$3)"; FAIL=$((FAIL+1)); fi
}

# Build a throwaway plugin root with a stub bin/ dir on PATH.
new_root() {
  ROOT="$(mktemp -d)"
  STUB="$(mktemp -d)"
  mkdir -p "${ROOT}/bin"
  export CLAUDE_PLUGIN_ROOT="${ROOT}"
  export PATH="${STUB}:${PATH}"
  STUBDIR="${STUB}"
}

# Stub curl/tar that "downloads" a fake binary into the extraction dir.
stub_success() {
  cat >"${STUBDIR}/curl" <<'EOS'
#!/usr/bin/env bash
# locate -C target dir from a piped tar, or -o file; emulate by writing marker
# We cooperate with the tar stub below via a shared temp marker file.
exit 0
EOS
  cat >"${STUBDIR}/tar" <<'EOS'
#!/usr/bin/env bash
# args: -xzC <dir> ; create the expected binary there.
dir=""; prev=""
for a in "$@"; do [[ "$prev" == "-C" || "$prev" == "-xzC" ]] && dir="$a"; prev="$a"; done
[[ -z "$dir" ]] && for a in "$@"; do [[ -d "$a" ]] && dir="$a"; done
printf '#!/bin/sh\necho cthink\n' > "${dir}/critical-thinking"
exit 0
EOS
  chmod +x "${STUBDIR}/curl" "${STUBDIR}/tar"
}

# Stub curl that fails (network error).
stub_fail() {
  printf '#!/usr/bin/env bash\nexit 22\n' > "${STUBDIR}/curl"; chmod +x "${STUBDIR}/curl"
  printf '#!/usr/bin/env bash\nexit 0\n'  > "${STUBDIR}/tar";  chmod +x "${STUBDIR}/tar"
}

# Case 1: fast-path no-op when binary present at expected version.
new_root
EXPECTED="$(grep -E '^EXPECTED_VERSION=' "${SCRIPT}" | head -1 | cut -d'"' -f2)"
printf 'stub\n' > "${ROOT}/bin/critical-thinking"; chmod +x "${ROOT}/bin/critical-thinking"
printf '%s\n' "${EXPECTED}" > "${ROOT}/bin/.installed-version"
# No curl/tar on PATH would break it ONLY if it tried to download; ensure it doesn't.
printf '#!/usr/bin/env bash\necho "curl must not run" >&2; exit 99\n' > "${STUBDIR}/curl"; chmod +x "${STUBDIR}/curl"
bash "${SCRIPT}"; check "fast-path no-op when version matches" 0 $?

# Case 2: version mismatch triggers a (stubbed) successful download.
new_root; stub_success
printf 'old\n' > "${ROOT}/bin/.installed-version"
bash "${SCRIPT}"; rc=$?
check "mismatch downloads new binary" 0 "${rc}"
[[ -x "${ROOT}/bin/critical-thinking" ]] && echo "ok   - binary installed" && PASS=$((PASS+1)) || { echo "FAIL - binary missing"; FAIL=$((FAIL+1)); }
[[ "$(cat "${ROOT}/bin/.installed-version" 2>/dev/null)" == "${EXPECTED}" ]] && echo "ok   - installed-version recorded" && PASS=$((PASS+1)) || { echo "FAIL - installed-version not recorded"; FAIL=$((FAIL+1)); }

# Case 3: download fails but an existing binary is kept (exit 0).
new_root; stub_fail
printf 'stub\n' > "${ROOT}/bin/critical-thinking"; chmod +x "${ROOT}/bin/critical-thinking"
printf 'old\n' > "${ROOT}/bin/.installed-version"
bash "${SCRIPT}"; check "download fail + existing binary tolerated" 0 $?

# Case 4: download fails and no binary exists (exit 1).
new_root; stub_fail
bash "${SCRIPT}"; check "download fail + no binary fails loudly" 1 $?

echo "---"; echo "PASS=${PASS} FAIL=${FAIL}"
[[ "${FAIL}" -eq 0 ]]
