# API & CLI Reference

## Build Commands (justfile)

| Command | Purpose |
|---------|---------|
| `just serve` / `go run main.go` | Dev server on :8000 |
| `just build` | Build for current platform |
| `just build-frontend` | Bundle React frontend |
| `just build-linux-native` | Linux, CGO=0, pure Go |
| `just build-linux-usb` | Linux, CGO=1, gousb |
| `just build-windows-usb` | Windows cross-compile (mingw) |
| `just build-darwin-native` | macOS arm64+amd64 |
| `just build-all` | All platform targets |
| `just dev` | Concurrent Go + Vite dev server |
| `just package` | Create .tar.gz archives |

## CLI Commands (Cobra)

| Command | Purpose |
|---------|---------|
| `goqlprinter serve` | Start web server |
| `goqlprinter print` | Print from CLI |
| `goqlprinter printers` | List printers |
| `goqlprinter labels` | List label sizes |
| `goqlprinter fonts` | List fonts |
| `goqlprinter status` | Printer status |

## API Endpoints

| Method | Path | Handler | Purpose |
|--------|------|---------|---------|
| GET | `/api/config` | GetConfig | Server config + defaults |
| GET | `/api/printers` | GetPrinters | Discover Brother printers |
| GET | `/api/fonts` | GetFonts | List available fonts |
| GET | `/api/label-sizes` | GetLabelSizes | All label formats |
| GET | `/api/label-sizes/:id` | GetLabelSize | Single label details |
| POST | `/api/print` | PrintLabel | Print text/SVG label |
| POST | `/api/print_svg` | PrintSVG | Print SVG (needs rsvg-convert) |
| POST | `/api/print_png` | PrintPNGLabel | Print base64 PNG |
| POST | `/api/print_png_raw` | PrintPNGRaw | Print uploaded PNG file |
| POST | `/api/print_qr` | PrintQR | Generate + print QR code |
| POST | `/api/preview` | PreviewLabel | Return base64 PNG preview |
| POST | `/api/status` | GetStatus | Printer status query |
| POST | `/api/test/*` | Test* | Debug: invalidate/init/feed |

## Config Priority (low→high)

1. Defaults → 2. `/etc/labelprinter/config.json` → 3. `~/.labelprinter/config.json` → 4. `./config/config.json` → 5. `LABELPRINTER_*` env vars
