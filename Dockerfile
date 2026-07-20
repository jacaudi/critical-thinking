# syntax=docker/dockerfile:1.25@sha256:0adf442eae370b6087e08edc7c50b552d80ddf261576f4ebd6421006b2461f12

# ---- builder ----
FROM golang:1.26-alpine@sha256:0178a641fbb4858c5f1b48e34bdaabe0350a330a1b1149aabd498d0699ff5fb2 AS builder

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
FROM gcr.io/distroless/static-debian12:nonroot@sha256:f5b485ea962d9bd1186b2f6b3a061191539b905b82ec395de78cbfae51f20e35 AS release

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
