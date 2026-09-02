# syntax=docker/dockerfile:1.27@sha256:bde3983e9c939224420ddaf6b784cc30e09b035a4dea01f581230c50809f372e

# ---- builder ----
FROM golang:1.26-alpine@sha256:ce864e7223ac17b1775e6fd0b4c0db580c2eb50e7953a427916379e4b92a1628 AS builder

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
