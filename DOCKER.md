# Docker

This fork adds a multi-stage `Dockerfile`, a `docker-compose.yml`, and a
GitHub Actions workflow that publishes the image to Docker Hub on every push
to `main` (and on `v*` tags).

The image uses the **native** Go backend (no CGO, no libusb). On Linux the
native backend talks to the printer through the OS driver via `/dev/usb/lp*`,
so you just pass the device into the container at runtime — no `--privileged`
flag, no libusb-in-container needed.

## Quick start

```bash
# 1. Drop your config / certs / fonts next to the compose file
mkdir -p config certs fonts
cp config/config.docker.json config/config.json     # edit token / TLS as needed

# 2. Find your printer device
ls /dev/usb/lp*

# 3. Edit docker-compose.yml → set the right /dev/usb/lpN device

# 4. Run it
docker compose up -d

# 5. Open the UI
open http://localhost:8000
```

## Pulling the prebuilt image

```bash
docker pull nopenix/goqlprinter:latest
docker run --rm -p 8000:8000 --device /dev/usb/lp0 nopenix/goqlprinter:latest
```

## Configuration

Everything in [README.md](README.md) still applies. Environment variables
override any mounted JSON config. The image already exports sensible defaults:

| Variable | Default |
|---|---|
| `LABELPRINTER_SERVER_HOST` | `0.0.0.0` |
| `LABELPRINTER_SERVER_PORT` | `8000` |
| `LABELPRINTER_APP_BACKEND` | `native` |
| `LABELPRINTER_APP_FONT_DIRS` | `/app/fonts,/fonts,/usr/share/fonts` |
| `LABELPRINTER_SERVER_CERT_FILE` | `/certs/dev.crt` |
| `LABELPRINTER_SERVER_KEY_FILE` | `/certs/dev.key` |

The image ships with:
* The bundled `KOMIKAX_.ttf` font baked in at `/app/fonts/`.
* The dev self-signed cert/key at `/certs/dev.{crt,key}` (TLS off by default).

Mount your own over the top:

```bash
-v $(pwd)/config/config.json:/config/config.json:ro
-v $(pwd)/certs:/certs:ro
-v $(pwd)/fonts:/fonts:ro
```

## USB device access

The container's default user (`goql`, uid 1000) can't open `/dev/usb/lp*`
unless the host's udev grants it. The two supported setups are:

**Option A — run as root** (simplest, recommended for a dedicated host):
```yaml
user: "0:0"
```
or on the CLI: `--user 0`.

**Option B — udev rule on the host** (production, multi-tenant):
```udev
# /etc/udev/rules.d/50-brother-ql.rules
SUBSYSTEM=="usb", ATTR{idVendor}=="04f9", MODE="0666", GROUP="goql"
```
then run with `--user 1000:1000` and `group_add: ["goql"]`.

## GitHub Actions — what gets built and pushed

The workflow at `.github/workflows/docker.yml`:

* Triggers on push to `main`, on `v*` git tags, and on manual dispatch.
* Builds a multi-arch image (`linux/amd64` + `linux/arm64`) with `docker buildx`.
* Pushes to `docker.io/nopenix/goqlprinter` (namespace configurable via
  the `DOCKERHUB_NAMESPACE` secret).
* Tags:
  * `latest` on every push to `main`
  * `vX.Y.Z`, `vX.Y`, `vX` on `vX.Y.Z` git tags
  * `<sha>` for traceability
* Caches layers in the GitHub Actions cache for fast rebuilds.
* Generates SBOM + provenance attestations.

### Required repository secrets

| Secret | Where to get it |
|---|---|
| `DOCKERHUB_USERNAME` | Your Docker Hub username |
| `DOCKERHUB_TOKEN`    | Docker Hub → Account Settings → Security → New access token |
| `DOCKERHUB_NAMESPACE` *(optional)* | Override the Docker Hub namespace; defaults to `DOCKERHUB_USERNAME` |

## Building locally

```bash
docker build -t nopenix/goqlprinter:dev .
docker run --rm -p 8000:8000 --device /dev/usb/lp0 nopenix/goqlprinter:dev
```

The first build takes a few minutes (downloads Go modules, builds the React
frontend, compiles the Go binary). Subsequent builds use cached layers.

## Build the USB backend image instead (advanced)

If you need the libusb backend (e.g. macOS-in-Linux, vendor-class printers,
or you want full bidirectional status on every platform), the `Dockerfile`
in this repo is the native-backend flavour only. To get a USB build, swap
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
