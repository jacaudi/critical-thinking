# syntax=docker/dockerfile:1.23@sha256:2780b5c3bab67f1f76c781860de469442999ed1a0d7992a5efdf2cffc0e3d769

# ---- builder ----
FROM golang:1.26-alpine AS builder

ARG VERSION=dev
ARG BUILDTIME=unknown
ARG REVISION=unknown

WORKDIR /src

# Cache module fetches separately from source for faster rebuilds.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build \
    -trimpath \
    -ldflags "-s -w -X main.version=${VERSION}" \
    -o /out/critical-thinking ./cmd/critical-thinking

# ---- final ----
FROM gcr.io/distroless/static-debian12:nonroot AS release

COPY --from=builder /out/critical-thinking /critical-thinking

LABEL org.opencontainers.image.title="Critical Thinking"
LABEL org.opencontainers.image.description="MCP server for critical, narrated, sequential thinking"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.created="${BUILDTIME}"
LABEL org.opencontainers.image.revision="${REVISION}"
LABEL org.opencontainers.image.source="https://github.com/jacaudi/critical-thinking-mcp"

ENV DOCKER=true
EXPOSE 3000

# distroless has no shell or curl; orchestrator-level health probes hit
# /health from the network. No HEALTHCHECK directive in the image.

USER nonroot:nonroot
ENTRYPOINT ["/critical-thinking"]
CMD ["-http", ":3000"]
