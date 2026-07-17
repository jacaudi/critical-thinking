#!/usr/bin/env bash
# Download the critical-thinking MCP server binary from a pinned GitHub Release
# for the current platform into ${CLAUDE_PLUGIN_ROOT}/bin/ so the plugin's
# .mcp.json can launch it without a Go toolchain.
#
# Update model: EXPECTED_VERSION is the source of truth for which binary the
# plugin expects. semantic-release auto-bumps it on every release (see
# scripts/bump-plugin-version.mjs + .releaserc.json). On `claude plugin update`
# the new script arrives, the version mismatch fires, and the binary refreshes
# on next session start. No runtime API calls, no TTL guessing.
#
# Behavior:
#   - Binary present AND .installed-version == EXPECTED_VERSION -> no-op.
#   - Otherwise -> download EXPECTED_VERSION's archive + checksums.txt, verify the
#     SHA-256, extract, install, record version.
#   - Archive OR checksums download fails (network) AND a binary already exists ->
#     keep it, warn, exit 0. No existing binary -> fail loudly (exit 1). Never
#     installs an unverified binary.
#   - Checksum MISMATCH (possible tampering/corruption) -> always fail closed
#     (exit 1); the artifact is never made executable.
#
# Force re-download: delete ${CLAUDE_PLUGIN_ROOT}/bin/critical-thinking.
# Windows note: requires Git Bash / WSL / another POSIX shell.

set -euo pipefail

REPO="jacaudi/critical-thinking"
PROJECT="critical-thinking"

# DO NOT EDIT BY HAND. Auto-bumped on every release by semantic-release
# (scripts/bump-plugin-version.mjs via .releaserc.json @semantic-release/exec).
EXPECTED_VERSION="v1.14.0"

PLUGIN_ROOT="${CLAUDE_PLUGIN_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")"/.. && pwd)}"
BIN_DIR="${PLUGIN_ROOT}/bin"
BIN_PATH="${BIN_DIR}/${PROJECT}"
INSTALLED_VERSION_FILE="${BIN_DIR}/.installed-version"

# Fast path: binary present and at the expected version.
if [[ -x "${BIN_PATH}" && -f "${INSTALLED_VERSION_FILE}" ]]; then
  installed_tag="$(cat "${INSTALLED_VERSION_FILE}" 2>/dev/null || echo "")"
  if [[ "${installed_tag}" == "${EXPECTED_VERSION}" ]]; then
    exit 0
  fi
fi

UNAME_S="$(uname -s)"
case "${UNAME_S}" in
  Linux*)               OS=linux ;;
  Darwin*)              OS=darwin ;;
  MINGW*|MSYS*|CYGWIN*) OS=windows ;;
  *) echo "critical-thinking: unsupported OS: ${UNAME_S}" >&2; exit 1 ;;
esac

case "$(uname -m)" in
  x86_64|amd64)  ARCH=amd64 ;;
  arm64|aarch64) ARCH=arm64 ;;
  *) echo "critical-thinking: unsupported arch: $(uname -m)" >&2; exit 1 ;;
esac

VERSION="${EXPECTED_VERSION#v}"
EXT="tar.gz"
[[ "${OS}" == "windows" ]] && EXT="zip"
ARCHIVE_NAME="${PROJECT}_${VERSION}_${OS}_${ARCH}.${EXT}"
URL="https://github.com/${REPO}/releases/download/${EXPECTED_VERSION}/${ARCHIVE_NAME}"
CHECKSUMS_URL="https://github.com/${REPO}/releases/download/${EXPECTED_VERSION}/checksums.txt"

echo "critical-thinking: installing ${EXPECTED_VERSION} from ${URL}" >&2

mkdir -p "${BIN_DIR}"
WORK_DIR="$(mktemp -d)"
trap 'rm -rf "${WORK_DIR}"' EXIT

# Prefer sha256sum (Linux, Git Bash); fall back to shasum -a 256 (macOS default).
# Reads "<hash>  <name>" lines on stdin; cwd must contain <name>.
sha256_check() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum -c -
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 -c -
  else
    echo "critical-thinking: no sha256sum/shasum found; cannot verify integrity" >&2
    return 1
  fi
}

# keep_existing_or_fail <reason>: network-class failure. Preserve a working binary
# if we have one (exit 0); otherwise fail loudly (exit 1). NEVER installs.
keep_existing_or_fail() {
  if [[ -x "${BIN_PATH}" ]]; then
    echo "critical-thinking: $1; keeping existing binary at ${BIN_PATH}" >&2
    exit 0
  fi
  echo "critical-thinking: $1 and no existing binary — check network or install manually" >&2
  exit 1
}

# Download the archive to a file (must be on disk to hash it).
curl -fsSL -o "${WORK_DIR}/${ARCHIVE_NAME}" "${URL}" \
  || keep_existing_or_fail "archive download failed"

# Download the checksums manifest from the same release/tag.
curl -fsSL -o "${WORK_DIR}/checksums.txt" "${CHECKSUMS_URL}" \
  || keep_existing_or_fail "checksums download failed"

# Verify integrity BEFORE extracting or installing. A mismatch is a stronger
# signal than a network failure (possible tampering/corruption of an auto-executed
# binary), so it ALWAYS fails closed — we never install the artifact.
if ! ( cd "${WORK_DIR}" && grep " ${ARCHIVE_NAME}\$" checksums.txt | sha256_check ); then
  echo "critical-thinking: CHECKSUM VERIFICATION FAILED for ${ARCHIVE_NAME} — refusing to install (possible corruption or tampering)" >&2
  exit 1
fi

# Verified: extract, then install.
if [[ "${EXT}" == "tar.gz" ]]; then
  tar -xzf "${WORK_DIR}/${ARCHIVE_NAME}" -C "${WORK_DIR}"
else
  unzip -q "${WORK_DIR}/${ARCHIVE_NAME}" -d "${WORK_DIR}"
fi

SRC="${WORK_DIR}/${PROJECT}"
[[ "${OS}" == "windows" ]] && SRC="${WORK_DIR}/${PROJECT}.exe"

if [[ ! -f "${SRC}" ]]; then
  echo "critical-thinking: archive did not contain ${SRC}" >&2
  exit 1
fi

mv "${SRC}" "${BIN_PATH}"
chmod +x "${BIN_PATH}"
printf '%s\n' "${EXPECTED_VERSION}" > "${INSTALLED_VERSION_FILE}"
echo "critical-thinking: installed ${BIN_PATH} (${EXPECTED_VERSION}, checksum verified)" >&2
