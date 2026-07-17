#!/usr/bin/env bash
# Tests for install-binary.sh. Each case runs the script in an isolated fake
# CLAUDE_PLUGIN_ROOT with curl stubbed on PATH so no network is touched. The
# script's own tar/unzip calls run for real against archives the curl stubs
# build on disk.
set -uo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT="${HERE}/../hooks/install-binary.sh"
BASE_PATH="${PATH}"
PASS=0
FAIL=0

check() { # desc, expected_rc, actual_rc
  if [[ "$2" == "$3" ]]; then echo "ok   - $1"; PASS=$((PASS+1));
  else echo "FAIL - $1 (expected rc=$2 got rc=$3)"; FAIL=$((FAIL+1)); fi
}

# Build a throwaway plugin root with a stub bin/ dir on PATH. Resets PATH from
# BASE_PATH each time (rather than prepending onto the already-modified PATH)
# so a prior case's stub binaries (e.g. stub_fail's no-op tar) never leak into
# a later case that needs the real tar/unzip on the system PATH.
new_root() {
  ROOT="$(mktemp -d)"
  STUB="$(mktemp -d)"
  mkdir -p "${ROOT}/bin"
  export CLAUDE_PLUGIN_ROOT="${ROOT}"
  export PATH="${STUB}:${BASE_PATH}"
  STUBDIR="${STUB}"
}

# Stub curl that emulates: curl -fsSL -o <out> <url>. Serves a fixed archive for
# the release-archive URL and publishes its real SHA-256 for checksums.txt
# (MODE=bad substitutes a wrong hash to simulate tampering/corruption). Used for
# both the plain "successful download" cases and the dedicated checksum cases,
# since both need a real archive on disk whose published hash matches the bytes
# curl actually served.
stub_checksum() {  # $1 = ok|bad
  MODE="$1"
  cat >"${STUBDIR}/curl" <<EOS
#!/usr/bin/env bash
# Emulate: curl -fsSL -o <out> <url>. Serve a fixed archive; publish its real hash.
out=""; prev=""; url=""
for a in "\$@"; do [[ "\$prev" == "-o" ]] && out="\$a"; prev="\$a"; url="\$a"; done
[[ -z "\$out" ]] && exit 22
shared="${STUBDIR}/served-archive.tar.gz"
if [[ "\$url" == *checksums.txt ]]; then
  [[ -f "\$shared" ]] || exit 22
  if command -v sha256sum >/dev/null 2>&1; then h="\$(sha256sum "\$shared" | awk '{print \$1}')";
  else h="\$(shasum -a 256 "\$shared" | awk '{print \$1}')"; fi
  [[ "${MODE}" == "bad" ]] && h="0000000000000000000000000000000000000000000000000000000000000000"
  printf '%s  %s\n' "\$h" "\$(cat "${STUBDIR}/served-name")" > "\$out"
  exit 0
fi
# archive request: build one fixed tar.gz, serve it, and record its basename.
t="\$(mktemp -d)"; printf '#!/bin/sh\necho cthink\n' > "\$t/critical-thinking"
tar -czf "\$shared" -C "\$t" critical-thinking 2>/dev/null
cp "\$shared" "\$out"
basename "\$url" > "${STUBDIR}/served-name"
exit 0
EOS
  chmod +x "${STUBDIR}/curl"
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

# Case 2: version mismatch triggers a (stubbed) successful, checksum-verified download.
new_root; stub_checksum ok
printf 'old\n' > "${ROOT}/bin/.installed-version"
bash "${SCRIPT}"; rc=$?
check "mismatch downloads new binary" 0 "${rc}"
if [[ -x "${ROOT}/bin/critical-thinking" ]]; then echo "ok   - binary installed"; PASS=$((PASS+1));
else echo "FAIL - binary missing"; FAIL=$((FAIL+1)); fi
if [[ "$(cat "${ROOT}/bin/.installed-version" 2>/dev/null)" == "${EXPECTED}" ]]; then echo "ok   - installed-version recorded"; PASS=$((PASS+1));
else echo "FAIL - installed-version not recorded"; FAIL=$((FAIL+1)); fi

# Case 3: download fails but an existing binary is kept (exit 0).
new_root; stub_fail
printf 'stub\n' > "${ROOT}/bin/critical-thinking"; chmod +x "${ROOT}/bin/critical-thinking"
printf 'old\n' > "${ROOT}/bin/.installed-version"
bash "${SCRIPT}"; check "download fail + existing binary tolerated" 0 $?

# Case 4: download fails and no binary exists (exit 1).
new_root; stub_fail
bash "${SCRIPT}"; check "download fail + no binary fails loudly" 1 $?

# --- Checksum verification cases (issue #77) ---

# Case 5: checksum match -> installs and records version.
new_root; stub_checksum ok
printf 'old\n' > "${ROOT}/bin/.installed-version"
bash "${SCRIPT}"; rc=$?
check "checksum match installs" 0 "${rc}"
if [[ -x "${ROOT}/bin/critical-thinking" && "$(cat "${ROOT}/bin/.installed-version" 2>/dev/null)" == "${EXPECTED}" ]]; then
  echo "ok   - verified binary installed"; PASS=$((PASS+1));
else echo "FAIL - verified binary not installed"; FAIL=$((FAIL+1)); fi

# Case 6: checksum MISMATCH -> exit 1 and no install.
new_root; stub_checksum bad
bash "${SCRIPT}"; check "checksum mismatch fails closed" 1 $?
if [[ ! -e "${ROOT}/bin/critical-thinking" ]]; then echo "ok   - no binary on mismatch"; PASS=$((PASS+1));
else echo "FAIL - binary installed despite mismatch"; FAIL=$((FAIL+1)); fi

# Case 7: checksums.txt unreachable but existing binary -> keep, exit 0.
new_root
cat >"${STUBDIR}/curl" <<'EOS'
#!/usr/bin/env bash
out=""; prev=""; url=""
for a in "$@"; do [[ "$prev" == "-o" ]] && out="$a"; prev="$a"; url="$a"; done
if [[ "$url" == *checksums.txt ]]; then exit 22; fi   # checksums fail
t="$(mktemp -d)"; printf 'x\n' > "$t/critical-thinking"; tar -czf "$out" -C "$t" critical-thinking 2>/dev/null; exit 0
EOS
chmod +x "${STUBDIR}/curl"
printf 'stub\n' > "${ROOT}/bin/critical-thinking"; chmod +x "${ROOT}/bin/critical-thinking"
printf 'old\n' > "${ROOT}/bin/.installed-version"
bash "${SCRIPT}"; check "checksums unreachable + existing binary tolerated" 0 $?

# Case 8: checksums.txt unreachable and no binary -> exit 1.
new_root
cat >"${STUBDIR}/curl" <<'EOS'
#!/usr/bin/env bash
out=""; prev=""; url=""
for a in "$@"; do [[ "$prev" == "-o" ]] && out="$a"; prev="$a"; url="$a"; done
if [[ "$url" == *checksums.txt ]]; then exit 22; fi
t="$(mktemp -d)"; printf 'x\n' > "$t/critical-thinking"; tar -czf "$out" -C "$t" critical-thinking 2>/dev/null; exit 0
EOS
chmod +x "${STUBDIR}/curl"
bash "${SCRIPT}"; check "checksums unreachable + no binary fails" 1 $?

echo "---"; echo "PASS=${PASS} FAIL=${FAIL}"
[[ "${FAIL}" -eq 0 ]]
