# syntax=docker/dockerfile:1

# ── Stage 1: Build ──
FROM golang:1.25-bookworm AS builder

WORKDIR /src

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build args
ARG ENABLE_OTEL=false
ARG ENABLE_TSNET=false
ARG VERSION=dev

# Build static binary (CGO disabled for scratch/alpine compatibility)
RUN set -eux; \
    TAGS=""; \
    if [ "$ENABLE_OTEL" = "true" ]; then TAGS="otel"; fi; \
    if [ "$ENABLE_TSNET" = "true" ]; then \
        if [ -n "$TAGS" ]; then TAGS="$TAGS,tsnet"; else TAGS="tsnet"; fi; \
    fi; \
    if [ -n "$TAGS" ]; then TAGS="-tags $TAGS"; fi; \
    CGO_ENABLED=0 GOOS=linux \
    go build -ldflags="-s -w -X github.com/Minh-Tam-Solution/MTClaw/cmd.Version=${VERSION}" \
    ${TAGS} -o /out/mtclaw .

# ── Stage 2: Runtime ──
FROM alpine:3.22

ARG ENABLE_SANDBOX=false

# Install ca-certificates + wget (healthcheck) + optionally docker-cli (sandbox)
RUN set -eux; \
    apk add --no-cache ca-certificates wget; \
    if [ "$ENABLE_SANDBOX" = "true" ]; then \
        apk add --no-cache docker-cli; \
    fi

# Non-root user
RUN adduser -D -u 1000 -h /app mtclaw
WORKDIR /app

# Copy binary and migrations
COPY --from=builder /out/mtclaw /app/mtclaw
COPY --from=builder /src/migrations/ /app/migrations/
COPY docker-entrypoint.sh /app/docker-entrypoint.sh
RUN chmod +x /app/docker-entrypoint.sh

# Create data directories (owned by mtclaw user)
RUN mkdir -p /app/workspace /app/data /app/sessions /app/skills /app/tsnet-state /app/.mtclaw \
    && chown -R mtclaw:mtclaw /app

# Default environment
ENV MTCLAW_CONFIG=/app/config.json \
    MTCLAW_WORKSPACE=/app/workspace \
    MTCLAW_DATA_DIR=/app/data \
    MTCLAW_SESSIONS_STORAGE=/app/sessions \
    MTCLAW_SKILLS_DIR=/app/skills \
    MTCLAW_MIGRATIONS_DIR=/app/migrations \
    MTCLAW_HOST=0.0.0.0 \
    MTCLAW_PORT=18790

USER mtclaw

EXPOSE 18790

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:18790/health || exit 1

ENTRYPOINT ["/app/docker-entrypoint.sh"]
CMD ["serve"]
