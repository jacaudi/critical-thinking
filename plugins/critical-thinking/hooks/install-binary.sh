#!/usr/bin/env bash
# Download the critical-thinking MCP server binary from the latest GitHub
# Release for the current platform and stash it under ${CLAUDE_PLUGIN_ROOT}/bin/
# so the plugin's .mcp.json can launch it without a Go toolchain on the host.
#
# Idempotent: re-runs are no-ops once the binary is in place. Re-download by
# deleting ${CLAUDE_PLUGIN_ROOT}/bin/critical-thinking.
#
# Windows note: requires Git Bash, WSL, or another POSIX shell environment.

set -euo pipefail

REPO="jacaudi/critical-thinking-plugin"
PROJECT="critical-thinking"

PLUGIN_ROOT="${CLAUDE_PLUGIN_ROOT:-$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]:-$0}" 2>/dev/null || echo "${BASH_SOURCE[0]:-$0}")")"/.. && pwd)}"
BIN_DIR="${PLUGIN_ROOT}/bin"
BIN_PATH="${BIN_DIR}/${PROJECT}"

if [[ -x "${BIN_PATH}" ]]; then
  exit 0
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

# Resolve latest release tag (no jq dependency).
LATEST_TAG=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
  | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -1)

if [[ -z "${LATEST_TAG}" ]]; then
  echo "critical-thinking: could not resolve latest release tag from https://api.github.com/repos/${REPO}/releases/latest" >&2
  echo "critical-thinking: ensure a release exists, or install the binary manually and put it on PATH." >&2
  exit 1
fi

VERSION="${LATEST_TAG#v}"

EXT="tar.gz"
[[ "${OS}" == "windows" ]] && EXT="zip"

URL="https://github.com/${REPO}/releases/download/${LATEST_TAG}/${PROJECT}_${VERSION}_${OS}_${ARCH}.${EXT}"

echo "critical-thinking: downloading ${URL}" >&2

mkdir -p "${BIN_DIR}"
TMPDIR="$(mktemp -d)"
trap 'rm -rf "${TMPDIR}"' EXIT

if [[ "${EXT}" == "tar.gz" ]]; then
  curl -fsSL "${URL}" | tar -xzC "${TMPDIR}"
else
  curl -fsSL -o "${TMPDIR}/release.zip" "${URL}"
  unzip -q "${TMPDIR}/release.zip" -d "${TMPDIR}"
fi

# Goreleaser archives place the binary at the archive root.
SRC="${TMPDIR}/${PROJECT}"
[[ "${OS}" == "windows" ]] && SRC="${TMPDIR}/${PROJECT}.exe"

if [[ ! -f "${SRC}" ]]; then
  echo "critical-thinking: archive did not contain ${SRC}" >&2
  exit 1
fi

mv "${SRC}" "${BIN_PATH}"
chmod +x "${BIN_PATH}"

echo "critical-thinking: installed ${BIN_PATH} (release ${LATEST_TAG})" >&2
