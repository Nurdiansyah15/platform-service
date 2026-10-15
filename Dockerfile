# ── Build stage ───────────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/app     ./cmd/app
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/seeder  ./cmd/seeder
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/worker  ./cmd/worker

# ── App image ─────────────────────────────────────────────────────────────────
FROM alpine:3.20 AS app
RUN apk --no-cache add ca-certificates tzdata && adduser -D -u 10001 platform
COPY --from=builder /out/app /app
# i18n bundle dibaca dari LOCALES_DIR (default "locales" relatif ke working dir).
COPY --from=builder /build/locales /locales
WORKDIR /
USER platform
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/healthz || exit 1
CMD ["/app"]

# ── Migrate image ─────────────────────────────────────────────────────────────
# File migrasi di-embed ke binary.
FROM alpine:3.20 AS migrate
RUN apk --no-cache add ca-certificates tzdata
COPY --from=builder /out/migrate /migrate
ENTRYPOINT ["/migrate"]

# ── Seeder image ──────────────────────────────────────────────────────────────
FROM alpine:3.20 AS seeder
RUN apk --no-cache add ca-certificates tzdata
COPY --from=builder /out/seeder /seeder
ENTRYPOINT ["/seeder"]

# ── Worker image ──────────────────────────────────────────────────────────────
# Pengirim webhook (outbox → webhook_deliveries → HTTP).
FROM alpine:3.20 AS worker
RUN apk --no-cache add ca-certificates tzdata && adduser -D -u 10001 platform
COPY --from=builder /out/worker /worker
USER platform
ENTRYPOINT ["/worker"]
