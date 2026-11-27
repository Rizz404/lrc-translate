# Single-container image: builds the React/Vite frontend, builds the Go
# backend, then ships one small image where the Go binary also serves the
# built frontend (static files + SPA fallback — see
# backend/internal/httpapi/router.go). Build context must be the repo root
# (it needs both backend/ and frontend/).
#
#   docker build -t lrc-translate .
#   docker run -p 8080:8080 -v lrc-translate-data:/app/data lrc-translate

# ---- Stage 1: build the frontend -------------------------------------------
FROM docker.io/library/node:26-alpine@sha256:6bd9ab43c0ec0190ca7b2d193906cb8b7edf6e7f4afa43ddeccd6452710bffb5 AS frontend-build
WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# ---- Stage 2: build the backend --------------------------------------------
FROM docker.io/library/golang:1.27-alpine@sha256:70bd7ddea8b6cbe5d6943cde279d6b67feb10cc924fb9100d1d8757c1575b280 AS backend-build
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download && go mod verify
COPY backend/ ./
# CGO_ENABLED=0: the sqlite driver (glebarez/sqlite -> modernc.org/sqlite) is
# pure Go, so a static binary with no C toolchain is enough.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/server ./cmd/server

# ---- Stage 3: final runtime image ------------------------------------------
FROM docker.io/library/alpine:3.24@sha256:f59abe656b692d901e184be3445e084786602108599d2a51d229c0f5a48c71bf
# ca-certificates: needed for outbound HTTPS calls (LRCLIB, LibreTranslate,
# scrape targets). tzdata: correct local time in logs.
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S app \
    && adduser -S app -G app

WORKDIR /app
COPY --from=backend-build /out/server ./server
COPY --from=frontend-build /src/frontend/dist ./web

ENV PORT=8080 \
    GIN_MODE=release \
    STATIC_DIR=/app/web \
    DB_DRIVER=sqlite \
    DB_DSN=/app/data/db.sqlite

RUN mkdir -p /app/data && chown -R app:app /app
USER app

EXPOSE 8080
VOLUME ["/app/data"]

HEALTHCHECK --interval=30s --timeout=5s --start-period=60s --retries=3 \
    CMD wget -q -T 3 -O /dev/null "http://127.0.0.1:${PORT}/api/health" || exit 1

ENTRYPOINT ["/app/server"]
