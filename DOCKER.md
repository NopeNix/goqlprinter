# Docker

This fork adds a multi-stage `Dockerfile`, a `docker-compose.yml`, and a
GitHub Actions workflow that publishes the image to Docker Hub on every push
to `main` (and on `v*` tags).

The image uses the **native** Go backend (no CGO, no libusb). On Linux the
native backend talks to the printer through the OS driver via `/dev/usb/lp*`,
so you just pass the device into the container at runtime — no `--privileged`
flag, no libusb shim.

## Network mode (Brother QL-810W / 720NW / 820NWB / 1110NWB over WiFi or Ethernet)

The network backend talks to a network-capable Brother QL printer over a
**raw TCP socket on port 9100** (Brother's "raw" / JetDirect-style protocol).
The wire format is identical to USB — same raster bytes, same model-specific
header — so the same `goqlprinter` binary works. The only thing that does
NOT work over the network is bidirectional status queries (`ESC i S`), since
port 9100 is write-only. The web UI's status badge will show "unknown" for
network printers; the print pipeline is fully functional.

```yaml
# compose.yaml — network mode
services:
  goqlprinter:
    image: nopenix/goqlprinter:latest
    environment:
      LABELPRINTER_APP_BACKEND: network
      LABELPRINTER_APP_DEFAULT_PRINTER: QL-810W
      LABELPRINTER_APP_NETWORK_URI: tcp://192.168.1.42:9100
    # no --device /dev/usb/lp0 needed
    ports:
      - "8000:8000"
```

Accepted URI forms for `LABELPRINTER_APP_NETWORK_URI`:

| URI | Notes |
|---|---|
| `tcp://192.168.1.42:9100` | explicit port |
| `tcp://192.168.1.42` | port defaults to 9100 |
| `network://192.168.1.42:9100` | `network://` is an alias for `tcp://` |
| `192.168.1.42:9100` | bare host:port works too |
| `tcp://[::1]:9100` | IPv6 with brackets |
| `tcp://fe80::1:9100` | IPv6 without brackets (parsed heuristically) |

If you need to talk to multiple network printers, run multiple goqlprinter
containers with different `LABELPRINTER_APP_NETWORK_URI` values and front them
with a reverse proxy (Caddy, nginx, traefik). One container = one printer
for now; a future revision could round-robin across a list.

## Quick start (USB)

```bash
docker pull nopenix/goqlprinter:latest
docker run -d --rm --name goqlprinter \
  --device /dev/usb/lp0 \
  -p 8000:8000 \
  -e LABELPRINTER_SERVER_TOKEN="$(openssl rand -hex 32)" \
  nopenix/goqlprinter:latest
```

## Pulling the prebuilt image

```bash
docker pull nopenix/goqlprinter:latest
docker run --rm -p 8000:8000 --device /dev/usb/lp0 nopenix/goqlprinter:latest
```

## Configuration

Everything in [README.md](README.md) still applies. Environment variables
override any mounted JSON config. The image already exports sensible defaults:

| Variable | Default | Notes |
|---|---|---|
| `LABELPRINTER_SERVER_HOST` | `0.0.0.0` | |
| `LABELPRINTER_SERVER_PORT` | `8000` | |
| `LABELPRINTER_APP_BACKEND` | `native` | `native`, `usb`, `auto`, or `network` |
| `LABELPRINTER_APP_NETWORK_URI` | *(empty)* | required when backend=network |
| `LABELPRINTER_APP_DEFAULT_PRINTER` | *(empty)* | model name, e.g. `QL-810W` |
| `LABELPRINTER_APP_FONT_DIRS` | `/app/fonts,/fonts` | searched in order |
| `LABELPRINTER_SERVER_TLS` | `false` | enable HTTPS (mount certs into /certs) |
| `LABELPRINTER_SERVER_TOKEN` | *(empty)* | Bearer token; if set, all `/api/*` requires it |

## USB device access

The container's default user is `goql` (uid 1000). The default `docker-compose.yml`
flips to `root` so the container can open `/dev/usb/lp*` without udev work.

For non-root USB access (the more secure option), add a host-side udev rule:

```udev
# /etc/udev/rules.d/50-brother-ql.rules
SUBSYSTEM=="usb", ATTR{idVendor}=="04f9", MODE="0666", GROUP="goql"
```

then `sudo udevadm control --reload-rules && sudo udevadm trigger`. Drop the
`user: "0:0"` line from compose and add `group_add: ["goql"]`.

## Build the USB backend image instead (advanced)

If you need the libusb backend (e.g. macOS-in-Linux, vendor-class printers,
or you want full bidirectional status on every platform), this repo's
`Dockerfile` is the native-backend flavour only. To get a USB build, swap
the `gobuild` stage for:

```dockerfile
FROM golang:1.24.2-bookworm AS gobuild
RUN apt-get update && apt-get install -y --no-install-recommends \
        libusb-1.0-0-dev pkg-config ca-certificates \
 && rm -rf /var/lib/apt/lists/*
ENV CGO_ENABLED=1
RUN go install github.com/swaggo/swag/cmd/swag@v1.16.3 \
 && swag init -g main.go -o docs --parseDependency --parseInternal
RUN go build -trimpath -tags usb \
    -ldflags="-s -w -X 'main.version=${VERSION}' -X 'main.buildDate=${BUILD_DATE}'" \
    -o /out/goqlprinter .
```

## Building locally

```bash
docker buildx build --platform linux/amd64,linux/arm64 -t nopenix/goqlprinter:dev .
```

The first build takes a few minutes (downloads Go modules, builds the React
frontend, compiles the Go binary). Subsequent builds use cached layers.
