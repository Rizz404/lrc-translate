# Single-container image: builds the React/Vite frontend, builds the Go
# backend, then ships one small image where the Go binary also serves the
# built frontend (static files + SPA fallback — see
# backend/internal/httpapi/router.go). Build context must be the repo root
# (it needs both backend/ and frontend/).
#
#   docker build -t lrc-translate .
#   docker run -p 8080:8080 -v lrc-translate-data:/app/data lrc-translate

# ---- Stage 1: build the frontend -------------------------------------------
FROM node:22-alpine AS frontend-build
WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# ---- Stage 2: build the backend --------------------------------------------
FROM golang:1.26-alpine AS backend-build
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
# CGO_ENABLED=0: the sqlite driver (glebarez/sqlite -> modernc.org/sqlite) is
# pure Go, so a static binary with no C toolchain is enough.
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/server ./cmd/server

# ---- Stage 3: final runtime image ------------------------------------------
FROM alpine:3.20
# ca-certificates: needed for outbound HTTPS calls (LRCLIB, LibreTranslate,
# scrape targets). tzdata: correct local time in logs.
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S app && adduser -S app -G app

WORKDIR /app
COPY --from=backend-build /out/server ./server
COPY --from=frontend-build /src/frontend/dist ./web

ENV PORT=8080 \
    STATIC_DIR=/app/web \
    DB_DRIVER=sqlite \
    DB_DSN=/app/data/db.sqlite

RUN mkdir -p /app/data && chown -R app:app /app
USER app

EXPOSE 8080
VOLUME ["/app/data"]

ENTRYPOINT ["/app/server"]
