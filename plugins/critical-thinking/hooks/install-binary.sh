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
#   - Otherwise -> download EXPECTED_VERSION's archive, install, record version.
#   - Download fails AND a binary already exists -> keep it, warn, exit 0.
#   - Download fails AND no binary exists -> fail loudly (exit 1).
#
# Force re-download: delete ${CLAUDE_PLUGIN_ROOT}/bin/critical-thinking.
# Windows note: requires Git Bash / WSL / another POSIX shell.

set -euo pipefail

REPO="jacaudi/critical-thinking"
PROJECT="critical-thinking"

# DO NOT EDIT BY HAND. Auto-bumped on every release by semantic-release
# (scripts/bump-plugin-version.mjs via .releaserc.json @semantic-release/exec).
EXPECTED_VERSION="v1.9.1"

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
URL="https://github.com/${REPO}/releases/download/${EXPECTED_VERSION}/${PROJECT}_${VERSION}_${OS}_${ARCH}.${EXT}"

echo "critical-thinking: installing ${EXPECTED_VERSION} from ${URL}" >&2

mkdir -p "${BIN_DIR}"
WORK_DIR="$(mktemp -d)"
trap 'rm -rf "${WORK_DIR}"' EXIT

download_ok=1
if [[ "${EXT}" == "tar.gz" ]]; then
  curl -fsSL "${URL}" | tar -xzC "${WORK_DIR}" || download_ok=0
else
  curl -fsSL -o "${WORK_DIR}/release.zip" "${URL}" || download_ok=0
  if [[ "${download_ok}" == "1" ]]; then
    unzip -q "${WORK_DIR}/release.zip" -d "${WORK_DIR}" || download_ok=0
  fi
fi

# Download failure: tolerate it if we already have *some* binary; otherwise fail.
if [[ "${download_ok}" == "0" ]]; then
  if [[ -x "${BIN_PATH}" ]]; then
    echo "critical-thinking: download failed; keeping existing binary at ${BIN_PATH}" >&2
    exit 0
  fi
  echo "critical-thinking: download failed and no existing binary — check network or install manually" >&2
  exit 1
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
echo "critical-thinking: installed ${BIN_PATH} (${EXPECTED_VERSION})" >&2
