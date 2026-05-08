#!/usr/bin/env bash
# Download the critical-thinking MCP server binary from a pinned GitHub
# Release for the current platform and stash it under ${CLAUDE_PLUGIN_ROOT}/bin/
# so the plugin's .mcp.json can launch it without a Go toolchain on the host.
#
# Update model: this script is the source of truth for which binary version
# the plugin expects. semantic-release bumps EXPECTED_VERSION on every
# release (see .releaserc.json prepareCmd), and the marketplace ships the
# updated script. When the user runs `claude plugin update`, the new script
# arrives, the EXPECTED_VERSION mismatch fires, and the binary gets refreshed
# on next session start. No runtime API calls, no TTL guessing.
#
# Behavior:
#   - Binary present AND .installed-version matches EXPECTED_VERSION → no-op.
#   - Otherwise → download EXPECTED_VERSION's archive, install, record version.
#   - Download fails AND a binary already exists → keep existing, warn, exit 0.
#   - Download fails AND no binary exists → fail loudly.
#
# Force re-download: delete ${CLAUDE_PLUGIN_ROOT}/bin/critical-thinking.
#
# Windows note: requires Git Bash, WSL, or another POSIX shell environment.

set -euo pipefail

REPO="jacaudi/critical-thinking-plugin"
PROJECT="critical-thinking"

# DO NOT EDIT BY HAND. Auto-bumped on every release by semantic-release
# (see .releaserc.json `@semantic-release/exec` prepareCmd).
EXPECTED_VERSION="v1.2.0"

PLUGIN_ROOT="${CLAUDE_PLUGIN_ROOT:-$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]:-$0}" 2>/dev/null || echo "${BASH_SOURCE[0]:-$0}")")"/.. && pwd)}"
BIN_DIR="${PLUGIN_ROOT}/bin"
BIN_PATH="${BIN_DIR}/${PROJECT}"
INSTALLED_VERSION_FILE="${BIN_DIR}/.installed-version"

# Fast path: binary present and at the expected version.
if [[ -x "${BIN_PATH}" && -f "${INSTALLED_VERSION_FILE}" ]]; then
  installed_tag=$(cat "${INSTALLED_VERSION_FILE}" 2>/dev/null || echo "")
  if [[ "${installed_tag}" == "${EXPECTED_VERSION}" ]]; then
    exit 0
  fi
fi

case "$(uname -s)" in
  Linux*)               OS=linux ;;
  Darwin*)              OS=darwin ;;
  MINGW*|MSYS*|CYGWIN*) OS=windows ;;
  *)
    echo "critical-thinking: unsupported OS: $(uname -s)" >&2
    exit 1
    ;;
esac

case "$(uname -m)" in
  x86_64|amd64)  ARCH=amd64 ;;
  arm64|aarch64) ARCH=arm64 ;;
  *)
    echo "critical-thinking: unsupported arch: $(uname -m)" >&2
    exit 1
    ;;
esac

VERSION="${EXPECTED_VERSION#v}"
EXT="tar.gz"
[[ "${OS}" == "windows" ]] && EXT="zip"
URL="https://github.com/${REPO}/releases/download/${EXPECTED_VERSION}/${PROJECT}_${VERSION}_${OS}_${ARCH}.${EXT}"

echo "critical-thinking: installing ${EXPECTED_VERSION} from ${URL}" >&2

mkdir -p "${BIN_DIR}"
TMPDIR="$(mktemp -d)"
trap 'rm -rf "${TMPDIR}"' EXIT

download_ok=1
if [[ "${EXT}" == "tar.gz" ]]; then
  curl -fsSL "${URL}" | tar -xzC "${TMPDIR}" || download_ok=0
else
  curl -fsSL -o "${TMPDIR}/release.zip" "${URL}" || download_ok=0
  if [[ "${download_ok}" == "1" ]]; then
    unzip -q "${TMPDIR}/release.zip" -d "${TMPDIR}" || download_ok=0
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

SRC="${TMPDIR}/${PROJECT}"
[[ "${OS}" == "windows" ]] && SRC="${TMPDIR}/${PROJECT}.exe"

if [[ ! -f "${SRC}" ]]; then
  echo "critical-thinking: archive did not contain ${SRC}" >&2
  exit 1
fi

mv "${SRC}" "${BIN_PATH}"
chmod +x "${BIN_PATH}"
printf '%s\n' "${EXPECTED_VERSION}" > "${INSTALLED_VERSION_FILE}"

echo "critical-thinking: installed ${BIN_PATH} (${EXPECTED_VERSION})" >&2
