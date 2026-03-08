# System Architecture

## Components

| Package | Purpose |
|---------|---------|
| `main.go` | Entry point, Gin router, backend init, embedded frontend |
| `api/` | REST handlers (13 files): print, preview, printers, labels, fonts, config, status, test |
| `brotherql/` | Core printer protocol, rasterization, models, platform backends |
| `services/` | Printer discovery, font management, connection logic |
| `config/` | Viper-based config (JSON files + env vars) |
| `logger/` | Logging infrastructure |

## Architecture Patterns

- **Strategy:** `Backend` + `BackendProvider` interfaces abstract USB vs native communication
- **DI:** Global `BackendProvider` injected at startup via `SetDefaultProvider()`
- **Mutex:** `services.PrinterLock` serializes all printer access
- **Handler func:** `PrinterHandler = func(backend, model) error` passed to `ConnectToPrinter()`
- **Embed:** Frontend SPA bundled into binary via `//go:embed`

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
main() → logger.Init → config.Load → selectBackend(auto|usb|native)
       → InitializeDefaultPrinter → setupGinRouter → ListenAndServe(:8000)
```
