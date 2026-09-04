# Dev container for critical-thinking: the Go toolchain, Task (the project's
# command runner), and the two shell tools the plugin lint/tests need. Every
# verification target lives in taskfile.yml so this container and CI run the
# same commands.
FROM golang:1.26-bookworm@sha256:9fdc884aacc3bec89b20ffc69f4bb369c78210e3e4f600387b5128b12c199f81

RUN apt-get update \
 && apt-get install -y --no-install-recommends jq shellcheck \
 && rm -rf /var/lib/apt/lists/*

RUN go install github.com/go-task/task/v3/cmd/task@v3.53.1
