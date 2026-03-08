# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go-based backend for a Brother QL label printer web GUI. It replaces a Python-based backend and provides RESTful APIs for discovering Brother QL printers, printing labels with text/SVG content, and managing printer configurations.

## Architecture

### Core Components
- **brotherql/**: Core printer protocol implementation
  - `brotherql.go`: Main printing logic with Brother QL protocol
  - `models.go`: Printer models, label sizes, and configuration data
  - `raster.go`: Image processing and rasterization
  - `usb_backend.go`: USB communication interface
- **api/**: REST API endpoints (Gin framework)
  - `print.go`: Main printing endpoint with text/SVG support
  - `printers.go`: Printer discovery
  - `labels.go`: Label size management
  - `fonts.go`: Font management
- **services/**: Business logic services
  - `printer_service.go`: USB printer discovery and management
  - `font_service.go`: Font path resolution
- **config/**: Configuration management with Viper

### Key Technologies
- **Gin**: HTTP framework for REST APIs
- **gousb**: USB device communication for printers
- **imaging**: Image processing and manipulation
- **freetype**: Text rendering
- **Viper**: Configuration management
- **Embed**: Static frontend serving

## Development Commands

### Build and Run
```bash
# Development server
go run main.go

# Build binary
go build -o backend_go main.go
```

### Available Commands
- `go run main.go` - Start the server on localhost:8000 (or configured port)
- `go build` - Build the binary
- `go test ./...` - Run tests (if any)
- `go mod tidy` - Clean up dependencies
- `go mod download` - Download dependencies

### Configuration
Configuration loaded from:
1. `/etc/labelprinter/config.json`
2. `$HOME/.labelprinter/config.json`
3. `./config/config.json`
4. Defaults (see config/config.go)

### Debug Features
- Print to file: Set `printer: "file"` in API requests to save as PNG
- Debug images saved to `debug_output/` directory
- Verbose logging enabled by default

## API Endpoints

### Printing
- `POST /api/print` - Print text labels
- `POST /api/print_png` - Print PNG images
- `POST /api/print_png_raw` - Print raw PNG data
- `POST /api/print_qr` - Print QR codes
- `POST /api/print_svg` - Print SVG labels (NEW)

### Discovery
- `GET /api/config` - Get server configuration
- `GET /api/label-sizes` - Get supported label sizes
- `GET /api/label-sizes/:id` - Get specific label size details
- `GET /api/printers` - List connected Brother printers
- `GET /api/fonts` - List available fonts
- `POST /api/status` - Get printer status

### Testing
- `POST /api/test/invalidate` - Test printer buffer invalidation
- `POST /api/test/initialize` - Test printer initialization
- `POST /api/test/feed` - Test media feed
- `POST /api/test/set_media_and_feed` - Test media configuration

### Documentation
- `GET /swagger/*any` - Swagger UI documentation

## Printer Models Supported

Complete support for Brother QL series: QL-500, 550, 560, 570, 580N, 650TD, 700, 710W, 720NW, 800, 810W, 820NWB, 1050, 1060N, 1100, 1110NWB.

Each model has specific protocol requirements stored in `brotherql/models.go`:
- Raster width bytes (90 for standard, 162 for wide)
- Protocol feature support (compression, high-res mode)
- Initialization parameters

## Label Sizes

20+ predefined label formats including:
- **Endless tape**: 12mm, 18mm, 29mm, 38mm, 50mm, 54mm, 62mm, 102mm, 104mm
- **Die-cut labels**: Various rectangular and round sizes
- Custom margins and feed settings per label type

## Font Management

Fonts automatically discovered from:
- `./fonts/` directory
- `./custom_fonts/` directory
- Additional paths via configuration

Roboto font family included by default.

## Development Notes

### USB Printer Discovery
- Uses gousb library for cross-platform USB communication
- Vendor ID: `0x04f9` (Brother)
- USB devices formatted as `usb:bus:address` strings
- Windows users may need WinUSB driver via Zadig

### Image Processing Pipeline
1. Input text/SVG/PNG → grayscale conversion
2. Canvas sizing based on label dimensions
3. Content positioning (center/start/end alignment)
4. Raster byte conversion (300 DPI)
5. Brother QL protocol commands
6. USB transmission

### Common Issues
- **USB permissions**: Ensure user has USB device access
- **rsgv-convert missing**: Required for SVG processing (`apt install librsvg2-bin`)
- **Image distortion**: Check label size DPI settings vs actual printer
- **Fonts missing**: Verify font paths in configuration

## AI Context Docs

Compressed documentation for AI assistants in `docs/ai-*.md`:

| Doc | Content |
|-----|---------|
| `ai-system.md` | Architecture, components, patterns |
| `ai-dataflow.md` | Print pipeline, data structures |
| `ai-cli-reference.md` | API endpoints, build commands |
| `ai-coding-guide.md` | Protocol rules, MUST/MUST NOT |
| `ai-tech-stack.md` | Dependencies, hardware support |

### Load Full Context
```bash
claude --append-system-prompt "$(cat docs/ai-*.md)"
```