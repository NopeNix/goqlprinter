# syntax=docker/dockerfile:1.7
# ─────────────────────────────────────────────────────────────────────────────
# goqlprinter — multi-stage Docker build
#
# Produces a minimal image that runs the native-backend binary (no CGO,
# no libusb). On Linux the native backend talks to the printer through the
# OS driver (/dev/usb/lp*), so you only need to pass the USB device into
# the container at runtime — no --privileged, no libusb shim.
#
# Build:
#   docker build -t nopenix/goqlprinter:dev .
#
# Run (replace /dev/usb/lp0 with the actual device on the host):
#   docker run --rm -p 8000:8000 \
#     --device /dev/usb/lp0 \
#     -v $(pwd)/config:/config:ro \
#     -v $(pwd)/fonts:/fonts:ro \
#     -e LABELPRINTER_SERVER_HOST=0.0.0.0 \
#     nopenix/goqlprinter
# ─────────────────────────────────────────────────────────────────────────────

ARG GO_VERSION=1.24.2
ARG NODE_VERSION=22-alpine

# ─── Stage 1: build the React frontend ──────────────────────────────────────
FROM node:${NODE_VERSION} AS frontend

WORKDIR /src/frontend

# Cache npm install layer
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm ci --no-audit --no-fund || npm install --no-audit --no-fund

COPY frontend/ ./
RUN npm run build


# ─── Stage 2: build the Go binary (native backend, no CGO) ─────────────────
FROM golang:${GO_VERSION}-alpine AS gobuild

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG VERSION=dev
ARG BUILD_DATE

# git is required so `go install` for swag can pull its module
RUN apk add --no-cache git ca-certificates

WORKDIR /src

# Cache go module download layer
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source
COPY . .

# Copy the built frontend into the embed path expected by main.go
COPY --from=frontend /src/frontend/dist ./frontend/dist

# Install swag and generate Swagger docs (matches `just swagger` in the repo)
RUN go install github.com/swaggo/swag/cmd/swag@v1.16.3
RUN swag init -g main.go -o docs --parseDependency --parseInternal

# Build the native-backend binary (CGO disabled, no libusb needed)
# ldflags: strip debug info + bake in version metadata
ENV CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH}
RUN go build \
    -trimpath \
    -tags native \
    -ldflags="-s -w \
      -X 'main.version=${VERSION}' \
      -X 'main.buildDate=${BUILD_DATE}'" \
    -o /out/goqlprinter \
    .


# ─── Stage 3: minimal runtime ───────────────────────────────────────────────
# Alpine: small, has ca-certs for HTTPS to printers / Let's Encrypt,
# and the entrypoint shell makes USB device mapping easy to debug.
FROM alpine:3.20

# tini for proper signal handling, ca-certs for TLS to upstream services,
# curl for the docker-healthcheck, shadow for `adduser`.
RUN apk add --no-cache ca-certificates tini curl shadow \
 && addgroup -S goql && adduser -S -G goql -u 1000 goql

WORKDIR /app

# Static assets shipped in the repo (fonts bundled with the project)
COPY --from=gobuild /out/goqlprinter      /app/goqlprinter
COPY fonts                                /app/fonts
COPY certs                                /app/certs
COPY config                               /app/config

# Runtime layout: the user mounts their own config/certs/fonts on top of these
RUN mkdir -p /config /certs /fonts /data \
 && chown -R goql:goql /app /config /certs /fonts /data \
 && chmod 755 /app/goqlprinter

ENV LABELPRINTER_SERVER_HOST=0.0.0.0 \
    LABELPRINTER_SERVER_PORT=8000 \
    LABELPRINTER_APP_FONT_DIRS="/app/fonts,/fonts,/usr/share/fonts" \
    LABELPRINTER_SERVER_CERT_FILE=/certs/dev.crt \
    LABELPRINTER_SERVER_KEY_FILE=/certs/dev.key

EXPOSE 8000

# Container-level health check — hits the unauthenticated /api/printers
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -fsS http://127.0.0.1:8000/api/label-sizes || exit 1

# tini reaps zombies and forwards signals so the server shuts down cleanly
ENTRYPOINT ["/sbin/tini", "--", "/app/goqlprinter", "serve"]

# By default, run as a non-root user. Override with `--user 0` if the host
# has no udev rule granting the printer to the in-container user, or
# mount `--device /dev/usb/lp0` and add the matching udev rule on the host.
USER goql
