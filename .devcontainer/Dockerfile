# Dev container for critical-thinking: the Go toolchain, Task (the project's
# command runner) and every lint tool `task ci` needs, at the SAME pinned
# versions .github/actions/setup/action.yml installs on the runner. Every
# verification target lives in taskfile.yml so this container and CI run the
# same commands and the same tool versions.
FROM golang:1.27-bookworm@sha256:648f440f42a0958804efb24df176f806f9d353b41f1c0627f666428e40310f6b

# shellcheck and jq: plugin lint. yamllint: repo:yaml. curl/ca-certificates: the
# pinned tarball installs below. (Versions are apt's; none of these is pinned in
# CI either — shellcheck, jq and yamllint ship on ubuntu-latest.)
RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates curl jq shellcheck yamllint \
 && rm -rf /var/lib/apt/lists/*

# renovate: datasource=go depName=github.com/go-task/task/v3
ARG TASK_VERSION=v3.53.1
# renovate: datasource=go depName=golang.org/x/vuln
ARG GOVULNCHECK_VERSION=v1.7.0
RUN go install "github.com/go-task/task/v3/cmd/task@${TASK_VERSION}" \
 && go install "golang.org/x/vuln/cmd/govulncheck@${GOVULNCHECK_VERSION}"

# Pinned release tarballs, not a piped install script — mirrors the setup action.
# Architecture-aware: this container is built natively (arm64 on Apple silicon,
# amd64 on the runner), and every project below publishes both.
# renovate: datasource=github-releases depName=golangci/golangci-lint
ARG GOLANGCI_LINT_VERSION=2.13.2
# renovate: datasource=github-releases depName=rhysd/actionlint
ARG ACTIONLINT_VERSION=1.7.12
# renovate: datasource=github-releases depName=hadolint/hadolint
ARG HADOLINT_VERSION=2.15.1
SHELL ["/bin/bash", "-o", "pipefail", "-c"]
RUN set -eu; arch="$(dpkg --print-architecture)"; \
    curl -sSfL "https://github.com/golangci/golangci-lint/releases/download/v${GOLANGCI_LINT_VERSION}/golangci-lint-${GOLANGCI_LINT_VERSION}-linux-${arch}.tar.gz" \
      | tar xz --strip-components=1 -C /usr/local/bin "golangci-lint-${GOLANGCI_LINT_VERSION}-linux-${arch}/golangci-lint"; \
    curl -sSfL "https://github.com/rhysd/actionlint/releases/download/v${ACTIONLINT_VERSION}/actionlint_${ACTIONLINT_VERSION}_linux_${arch}.tar.gz" \
      | tar xz -C /usr/local/bin actionlint; \
    case "${arch}" in amd64) hl=x86_64 ;; arm64) hl=arm64 ;; *) echo "unsupported arch ${arch}" >&2; exit 1 ;; esac; \
    curl -sSfLo /usr/local/bin/hadolint "https://github.com/hadolint/hadolint/releases/download/v${HADOLINT_VERSION}/hadolint-Linux-${hl}"; \
    chmod +x /usr/local/bin/hadolint; \
    golangci-lint --version; actionlint --version; hadolint --version
