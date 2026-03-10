# System Architecture

## Components

| Package | Purpose |
|---------|---------|
| `main.go` | Entry point, `//go:embed` frontend, delegates to `cmd.Execute()` |
| `cmd/` | Cobra CLI: serve, print, printers, labels, fonts, status |
| `api/` | REST handlers (13 files): print, preview, printers, labels, fonts, config, status, test |
| `brotherql/` | Core printer protocol, rasterization, models, platform backends |
| `internal/services/` | Printer discovery, font management, connection logic |
| `internal/config/` | Viper-based config (JSON files + env vars) |
| `internal/logging/` | Logging infrastructure (log/slog) |
| `frontend/` | React 18 + TypeScript + Tailwind, built with Vite |

## Architecture Patterns

- **Strategy:** `Backend` + `BackendProvider` interfaces abstract USB vs native communication
- **DI:** `api.Handlers` struct holds `PrinterService`, `FontService`, `Config`
- **Mutex:** `services.PrinterLock` serializes all printer access
- **Handler func:** `PrinterHandler = func(backend, model) error` passed to `ConnectToPrinter()`
- **Embed:** Frontend SPA bundled into binary via `//go:embed all:frontend/dist`
- **Build tags:** `usb` for gousb (CGO), `!usb` for native (pure Go)

## Platform Dispatch

```
Build tags:
  usb     → gousb/libusb (CGO required)
  !usb    → native OS backends (pure Go)

Platform files:
  native_linux.go   → /dev/usb/lp* + poll(2)
  native_darwin.go  → IOKit APIs
  native_windows.go → WinUSB via alexbrainman/printer
```

## Startup Flow

```
main() → cmd.Execute() → root.PersistentPreRun:
  logger.Init → config.Load → selectBackend(auto|usb|native)
  → InitializeDefaultPrinter
serve subcommand:
  → setupGinRouter → embed frontend → ListenAndServe(:8000)
```

## Key Interfaces

```go
Backend interface { Write, Read, Close }
BackendProvider interface { FindPrinters, Connect, SupportsStatus }
StatusProvider interface { GetStatus() (PrinterStatus, error) }
```
