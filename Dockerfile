# syntax=docker/dockerfile:1.28@sha256:bb22d9815c728170f72750f4e5b0d672e06176142e1d602c7e66c050100b7e5b

# ---- builder ----
FROM golang:1.27-alpine@sha256:cf6fca6641884b8433441b2b0652976f975e1d0fdd26d177eaaf8596087f3125 AS builder

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
    -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${REVISION} -X main.date=${BUILDTIME}" \
    -o /out/critical-thinking ./cmd/critical-thinking

# ---- final ----
FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab AS release

COPY --from=builder /out/critical-thinking /critical-thinking

LABEL org.opencontainers.image.title="Critical Thinking"
LABEL org.opencontainers.image.description="MCP server for critical, narrated, sequential thinking"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.created="${BUILDTIME}"
LABEL org.opencontainers.image.revision="${REVISION}"
LABEL org.opencontainers.image.source="https://github.com/jacaudi/critical-thinking"

ENV CTHINK_HTTP_HOST=0.0.0.0
EXPOSE 3000

# distroless has no shell or curl; orchestrator-level health probes hit
# /health from the network. No HEALTHCHECK directive in the image.

USER nonroot:nonroot
ENTRYPOINT ["/critical-thinking"]
CMD ["serve", "--http", ":3000"]
