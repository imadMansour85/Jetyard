# syntax=docker/dockerfile:1
#
# Single image: the Go backend serves the API and the statically exported
# frontend on one port.
#
#   docker build -t jetyard .
#   docker run -p 3000:3000 -e NATS_URL=nats://host:4222 -v jetyard-data:/data jetyard
#
# Both build stages run on the build machine's architecture and cross-compile,
# so a multi-arch build (linux/amd64,linux/arm64) needs no emulation.

# ---- frontend: static export -------------------------------------------------
FROM --platform=$BUILDPLATFORM node:20-alpine AS web
WORKDIR /web
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY frontend/ ./
# NEXT_PUBLIC_API_URL stays unset: the UI calls the API on its own origin.
ENV NEXT_OUTPUT=export NEXT_TELEMETRY_DISABLED=1
RUN npm run build

# ---- backend -----------------------------------------------------------------
FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS api
ARG TARGETOS TARGETARCH
ARG VERSION=dev
ARG COMMIT=""
ARG BUILD_TIME=""
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath \
      -ldflags "-s -w \
        -X github.com/imadMansour85/Jetyard/backend/buildinfo.Version=${VERSION} \
        -X github.com/imadMansour85/Jetyard/backend/buildinfo.Commit=${COMMIT} \
        -X github.com/imadMansour85/Jetyard/backend/buildinfo.BuildTime=${BUILD_TIME}" \
      -o /out/jetyard . \
 && mkdir -p /out/data

# ---- runtime: no shell, no package manager, non-root (uid 65532) -------------
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=api /out/jetyard /app/jetyard
COPY --from=web /web/out /app/web
# A new named volume copies this directory's ownership, so /data is writable.
COPY --from=api --chown=65532:65532 /out/data /data

ENV SERVER_PORT=3000 \
    STATIC_DIR=/app/web \
    AUTH_USERS_FILE=/data/auth-users.json \
    NATS_STORE_FILE=/data/nats-store.json \
    NATS_SETTINGS_FILE=/data/nats-settings.json \
    LOG_FORMAT=json

USER nonroot:nonroot
EXPOSE 3000
VOLUME /data
HEALTHCHECK --interval=30s --timeout=5s --start-period=90s --retries=3 \
    CMD ["/app/jetyard", "healthcheck"]
ENTRYPOINT ["/app/jetyard"]
