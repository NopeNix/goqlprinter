# goqlprinter

Lightweight web UI and REST API for Brother QL label printers. Turn any QL printer into a network printer, automate label printing from scripts, or just print labels faster than P-touch Editor. Single binary, no external dependencies.

![Brother QL series](https://img.shields.io/badge/Brother-QL_series-blue)
![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go)
![License](https://img.shields.io/badge/License-MIT-green)
![Docker](https://img.shields.io/badge/docker-ready-2496ED?logo=docker)
![Multi-arch](https://img.shields.io/badge/linux%2Famd64%20%7C%20linux%2Farm64-multi--arch-blue)

> **This fork** adds a multi-arch Docker image, a docker-compose setup that
> auto-publishes to [Docker Hub](https://hub.docker.com/r/nopenix/goqlprinter)
> via GitHub Actions, and a **network backend** so the QL-710W / QL-720NW /
> QL-810W / QL-820NWB / QL-1110NWB / QL-1115NWB can print over TCP/9100 — no
> USB cable, no driver, just WiFi or Ethernet.

## Features

- Web UI with live preview — print from any device on the network
- CLI for scripting and automation
- REST API for integration with other tools
- Text labels with font/alignment control, multiline support
- QR code generation and printing
- SVG and PNG image printing
- Auto-discovers connected Brother QL printers
- 20+ label sizes (endless tape, die-cut, round)
- **Network printing** — QL-810W and friends over raw TCP on port 9100
- HTTPS + token authentication for secure remote access
- Single binary with embedded frontend — no external dependencies
- Multi-arch Docker image (`linux/amd64` + `linux/arm64`) on Docker Hub

![goqlprinter screenshot](docs/screenshot.png)

## Docker — Quick Start

```bash
# Pull from Docker Hub
docker pull nopenix/goqlprinter:latest

# USB printer (Linux)
docker run -d --rm --name goqlprinter \
  --device /dev/usb/lp0 \
  -p 8000:8000 \
  nopenix/goqlprinter:latest

# Network printer (QL-810W over WiFi, no USB needed)
docker run -d --rm --name goqlprinter \
  -p 8000:8000 \
  -e LABELPRINTER_APP_BACKEND=network \
  -e LABELPRINTER_APP_DEFAULT_PRINTER=QL-810W \
  -e LABELPRINTER_APP_NETWORK_URI=tcp://192.168.1.42:9100 \
  nopenix/goqlprinter:latest
```

Open <http://localhost:8000>.

### docker-compose

```yaml
# compose.yaml
name: goqlprinter

services:
  goqlprinter:
    image: nopenix/goqlprinter:latest
    container_name: goqlprinter
    restart: unless-stopped

    # USB printer path (uncomment the device line):
    # devices:
    #   - /dev/usb/lp0:/dev/usb/lp0

    # Network printer path (WiFi / Ethernet QL-810W, QL-720NW, etc.):
    #   delete the `devices:` block above and use these env vars instead:
    environment:
      LABELPRINTER_SERVER_HOST: 0.0.0.0
      LABELPRINTER_SERVER_PORT: "8000"
      LABELPRINTER_APP_BACKEND: network        # or "native" / "usb" / "auto"
      LABELPRINTER_APP_DEFAULT_PRINTER: QL-810W
      LABELPRINTER_APP_NETWORK_URI: tcp://192.168.1.42:9100
      LABELPRINTER_SERVER_TOKEN: ""            # set a long random string to require Bearer auth

    ports:
      - "8000:8000"

    volumes:
      - ./config/config.json:/config/config.json:ro
      - ./certs:/certs:ro
      - ./fonts:/fonts:ro
      - ./debug_output:/data/debug_output

    user: "0:0"     # root so the container can open /dev/usb/lp* without udev
    healthcheck:
      test: ["CMD", "curl", "-fsS", "http://127.0.0.1:8000/api/label-sizes"]
      interval: 30s
      timeout: 5s
      retries: 3
      start_period: 10s
```

For more options (TLS, multi-arch, USB-backend, building from source, dev
overrides), see [DOCKER.md](DOCKER.md).

## Supported Printers

Brother QL-500, 550, 560, 570, 580N, 650TD, 700, **710W, 720NW, 810W,
820NWB, 1100, 1110NWB, 1115NWB** (WiFi/Ethernet models work over the
[network backend](#network-printing-over-tcp-9100) on port 9100), 1050, 1060N

## Requirements

- **Docker** (recommended) — just `docker run` from the image
- *OR* for building from source:
  - Go 1.24+
  - Node.js 18+ (frontend build)
  - [`just`](https://github.com/casey/just) (build tool)
  - USB access permissions for direct printer communication

## Getting Started (from source)

```bash
# Install frontend dependencies
just install-frontend

# Run development server (Go backend + Vite dev server)
just dev
```

This starts two servers concurrently:
- **<http://localhost:5173>** — Vite frontend with hot reload (use this in dev)
- **<http://localhost:8000>** — Go backend API

Open <http://localhost:5173> in your browser during development. In production,
the binary serves everything from port 8000.

## Building

```bash
# Build for current platform
just build

# Cross-compile for all platforms
just build-all
```

| Target | Command |
|--------|---------|
| Linux | `just build-linux-native` |
| Linux (ARM64) | `just build-linux-arm-native` |
| Windows | `just build-windows-native` |
| macOS (amd64 + arm64) | `just build-darwin-native` |

The binary embeds the frontend — no separate web server needed.

The native builds use OS printer interfaces and work with standard printer drivers. Printer status reporting varies by platform:

| Backend | Printing | Status reporting | Notes |
|---|---|---|---|
| Linux native (`/dev/usb/lp*`) | Full | Full (bidirectional) | `--device /dev/usb/lp0:/dev/usb/lp0` |
| Linux USB (libusb) | Full | Full | needs CGO, see below |
| macOS native (CUPS) | Full | Basic (CUPS/IPP, no media info) | |
| Windows native | Full | Minimal (connection status only) | |
| **Network (`tcp://host:9100`)** | **Full** | **None** (port 9100 is write-only) | new in this fork; works for QL-710W/720NW/810W/820NWB/1110NWB/1115NWB |

<details>
<summary>Advanced: USB (libusb) builds</summary>

USB builds communicate directly with the printer over USB, bypassing OS
drivers. This gives full bidirectional status reporting on all platforms,
but requires CGO, libusb-dev, and that no OS driver claims the device.

| Target | Command |
|--------|---------|
| Linux USB | `just build-linux-usb` |
| Windows USB | `just build-windows-usb` |

Windows USB builds require replacing the Brother driver with WinUSB using
[Zadig](https://zadig.akeo.ie/).
</details>

## Network Printing (over TCP/9100)

Network-capable Brother QL printers (710W, 720NW, 810W, 820NWB, 1110NWB,
1115NWB) listen on TCP port 9100 for raw print data — Brother's
"raw"/JetDirect-style protocol. The wire format is identical to USB, so
the same `goqlprinter` binary drives the printer over the network. **This
fork adds that backend; it's not in the upstream `mhavo/goqlprinter`.**

### Accepted URI formats for `LABELPRINTER_APP_NETWORK_URI`

| URI | Notes |
|---|---|
| `tcp://192.168.1.42:9100` | explicit port |
| `tcp://192.168.1.42` | port defaults to 9100 |
| `network://192.168.1.42:9100` | `network://` is an alias for `tcp://` |
| `192.168.1.42:9100` | bare host:port |
| `tcp://[::1]:9100` | IPv6 with brackets |
| `tcp://fe80::1:9100` | IPv6 without brackets (parsed heuristically) |

### Configure it

Set `app.backend: "network"` and provide the URI:

```json
{
  "app": {
    "backend": "network",
    "default_printer": "QL-810W",
    "network_uri": "tcp://192.168.198.64:9100"
  }
}
```

Or via env vars (the usual Docker path):

```bash
LABELPRINTER_APP_BACKEND=network
LABELPRINTER_APP_DEFAULT_PRINTER=QL-810W
LABELPRINTER_APP_NETWORK_URI=tcp://192.168.198.64:9100
```

The web UI's printer dropdown will then list `QL-810W` with
`id: tcp://192.168.198.64:9100`. Pick it, hit print, label comes out.

### CLI

```bash
# Print over the network
goqlprinter print "test label" \
  -p "tcp://192.168.198.64:9100" \
  -m "QL-810W" \
  -l 62
```

### Limitation

No bidirectional status over the network — port 9100 is write-only.
The web UI's status badge shows "unknown" for network printers; the
print pipeline itself is fully functional. The Python `brother_ql`
`network` backend has the same limitation. Can be layered on later via
SNMP if you ever need status.

## Configuration

Configuration is loaded in priority order (highest wins):

1. `LABELPRINTER_*` environment variables
2. `./config/config.json`
3. `~/.labelprinter/config.json`
4. `/etc/labelprinter/config.json`
5. Defaults

Example `config.json`:

```json
{
  "server": {
    "port": 8000,
    "host": "localhost",
    "tls": false,
    "cert_file": "./certs/dev.crt",
    "key_file": "./certs/dev.key",
    "token": ""
  },
  "app": {
    "backend": "auto",
    "default_printer": "QL-570",
    "font_dirs": ["./fonts", "~/.fonts"],
    "network_uri": ""
  }
}
```

| Field | Default | Notes |
|---|---|---|
| `server.host` | `localhost` | bind address |
| `server.port` | `8000` | HTTP port |
| `server.tls` | `false` | enable HTTPS (mount certs into `/certs`) |
| `server.cert_file` | *(empty)* | TLS cert path |
| `server.key_file` | *(empty)* | TLS key path |
| `server.token` | *(empty)* | Bearer token; if set, all `/api/*` require it |
| `app.backend` | `auto` | `native`, `usb`, `auto`, or `network` |
| `app.default_printer` | *(empty)* | model name, e.g. `QL-810W` |
| `app.font_dirs` | OS defaults | searched in order |
| `app.network_uri` | *(empty)* | required when `backend=network`; see [Network Printing](#network-printing-over-tcp-9100) |

## CLI Usage

```bash
# Start the web server
./goqlprinter serve

# List connected printers
./goqlprinter printers

# Print text directly
./goqlprinter print "Hello, World!" -l 62

# Print over the network
./goqlprinter print "Hello, World!" -p "tcp://192.168.1.42:9100" -m "QL-810W" -l 62

# List available label sizes and fonts
./goqlprinter labels
./goqlprinter fonts
```

## USB Permissions (Linux)

Add a udev rule so the printer is accessible without root:

```
SUBSYSTEM=="usb", ATTR{idVendor}=="04f9", MODE=="0666"
```

Save to `/etc/udev/rules.d/50-brother-ql.rules` and reload:
`sudo udevadm control --reload-rules`

For non-root inside the Docker container, the udev rule should also
`GROUP="goql"`, and compose should drop the `user: "0:0"` line in favour
of `group_add: ["goql"]`.

## macOS

The native build works if you have the Brother P-touch driver installed
(available on the App Store). For driverless operation, use the **USB
build** which communicates directly with the printer:

```bash
brew install libusb
just build-darwin-usb
sudo ./dist/darwin-usb-arm64/goqlprinter serve
```

Root access (`sudo`) is required because macOS restricts direct USB
device access.

## Windows

Works with the standard Brother printer driver — no extra setup needed.

## API

The server exposes a REST API at port 8000 (default):

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/printers` | List connected printers |
| GET | `/api/label-sizes` | List supported label sizes |
| GET | `/api/fonts` | List available fonts |
| GET | `/api/config` | Server + app config (no secrets) |
| GET | `/swagger/index.html` | Swagger UI for all endpoints |
| POST | `/api/print` | Print text label |
| POST | `/api/print_qr` | Print QR code |
| POST | `/api/print_svg` | Print SVG (needs `rsvg-convert` on the host) |
| POST | `/api/print_png` | Print base64 PNG |
| POST | `/api/print_png_raw` | Print uploaded PNG |
| POST | `/api/preview` | Get PNG preview (no print) |
| POST | `/api/status` | Query printer status |
| POST | `/api/test/invalidate` | Debug: clear raster compression state |
| POST | `/api/test/init` | Debug: re-init the printer |
| POST | `/api/test/feed` | Debug: feed paper |

Swagger UI available at `/swagger/index.html` (interactive docs).

## HTTPS & API Security

goqlprinter supports HTTPS and token-based API authentication, making it
safe to expose on a network for automated printing from other systems —
patient label printing from hospital information systems, asset tags
from inventory databases, shipping labels from warehouse management, etc.

### Quick Start (Development)

A self-signed certificate is included in `certs/` for development. Enable HTTPS:

```json
{
  "server": {
    "tls": true,
    "cert_file": "./certs/dev.crt",
    "key_file": "./certs/dev.key",
    "token": "my-secret-token"
  }
}
```

```bash
# Print from a script
curl -k https://localhost:8000/api/print \
  -H "Authorization: Bearer my-secret-token" \
  -H "Content-Type: application/json" \
  -d '{"text": "Patient: John Doe\nDOB: 1990-01-15", "label_size": "62"}'
```

### Production Setup

For production, use a proper certificate from your CA or Let's Encrypt:

```json
{
  "server": {
    "host": "0.0.0.0",
    "port": 8443,
    "tls": true,
    "cert_file": "/etc/ssl/certs/labelprinter.pem",
    "key_file": "/etc/ssl/private/labelprinter.key",
    "token": "your-secure-token-here"
  }
}
```

All settings can also be set via environment variables:

```bash
LABELPRINTER_SERVER_TLS=true
LABELPRINTER_SERVER_CERT_FILE=/path/to/cert.pem
LABELPRINTER_SERVER_KEY_FILE=/path/to/key.pem
LABELPRINTER_SERVER_TOKEN=your-secure-token-here
```

### Token Authentication

When `server.token` is set, all `/api/*` endpoints require a `Bearer`
token in the `Authorization` header. The token is compared using
constant-time comparison to prevent timing attacks. The token is never
exposed via the `/api/config` endpoint.

Without a token, the API is open — suitable for local use or when
behind a reverse proxy that handles authentication.

### Using with a Reverse Proxy

If you prefer to handle TLS termination externally (e.g., with Caddy
or nginx), run goqlprinter in plain HTTP mode and let the proxy handle
encryption:

```caddy
# Caddyfile
labels.example.com {
    reverse_proxy localhost:8000
}
```

## Debug Mode

Set `"printer": "file"` in any print request to write the rasterized
image to `debug_output/` instead of sending it to a printer.

## Acknowledgements

This project was inspired by and builds on ideas from:

- [brother_ql_web](https://github.com/pklaus/brother_ql_web) — Web
  interface for Brother QL printers (Python)
- [brother_ql](https://github.com/pklaus/brother_ql) — Brother QL
  printer protocol library (Python). The **network backend** in this
  fork is a Go port of their `network` backend; same wire protocol,
  same `tcp://host:9100` URI scheme, same bidirectional-status
  limitation. Credit to Pascal Klaeus for figuring out the protocol
  years ago.

This fork adds:

- Multi-stage `Dockerfile` with multi-arch build (linux/amd64 + linux/arm64)
- GitHub Actions workflow that builds and publishes the image to Docker Hub
  on every push to `main` and on `v*` git tags
- `docker-compose.yml` + a heavily-documented example + a local-dev override
- **Network backend** for raw TCP/9100 printing on QL-710W / 720NW /
  810W / 820NWB / 1100 / 1110NWB / 1115NWB (the WiFi/Ethernet models)

Proudly built with [Claude](https://claude.ai/).

## License

MIT — see [LICENSE](LICENSE)
