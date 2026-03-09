# Brother QL Printer Backend
version := `git describe --tags --always --dirty 2>/dev/null || echo "dev"`
build_date := `date +%Y-%m-%dT%H:%M:%S%z`
ldflags := "-ldflags \"-X main.version=" + version + " -X main.buildDate=" + build_date + "\""
out_dir := "dist"
build_deps := justfile_directory() / "build_deps"

# Show available commands
default:
    @just --list

# ─── Development ────────────────────────────────────────

# Build binary for current platform
build:
    go build {{ldflags}} -o goqlprinter .

# Start web server (go run)
serve:
    go run . serve

# Dev: Go backend + Vite devserver concurrently (Ctrl+C stops both)
dev:
    #!/usr/bin/env bash
    trap 'kill 0' SIGINT
    go run . serve &
    cd frontend && npm run dev &
    wait

# ─── CLI shortcuts ──────────────────────────────────────

# List connected printers
printers:
    go run . printers

# List available label sizes
labels:
    go run . labels

# List available fonts
fonts:
    go run . fonts

# Query printer status
status:
    go run . status

# Print text (usage: just print "Hello" -- -l 62)
print *ARGS:
    go run . print {{ARGS}}

# ─── Frontend ───────────────────────────────────────────

# Install frontend dependencies
install-frontend:
    cd frontend && npm install

# Build frontend for embedding
build-frontend:
    cd frontend && npm run build

# ─── Cross-compilation: USB backend (CGO required) ─────

build-linux-usb: build-frontend
    mkdir -p {{out_dir}}/linux-usb
    CGO_ENABLED=1 GOOS=linux \
    CGO_CFLAGS="-I{{build_deps}}/libusb-arm/include" \
    CGO_LDFLAGS="-L{{build_deps}}/libusb-arm/lib -lusb-1.0" \
    go build {{ldflags}} -tags usb -o {{out_dir}}/linux-usb/goqlprinter .

build-windows-usb: build-frontend
    mkdir -p {{out_dir}}/windows-usb
    CGO_ENABLED=1 GOOS=windows CC=x86_64-w64-mingw32-gcc \
    CGO_CFLAGS="-I{{build_deps}}/libusb-win/include" \
    CGO_LDFLAGS="-L{{build_deps}}/libusb-win/lib -lusb-1.0" \
    go build {{ldflags}} -tags usb -o {{out_dir}}/windows-usb/goqlprinter.exe .

# ─── Cross-compilation: Native backend (pure Go) ───────

build-linux-native: build-frontend
    mkdir -p {{out_dir}}/linux-native
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build {{ldflags}} -tags native -o {{out_dir}}/linux-native/goqlprinter .

build-linux-arm-native: build-frontend
    mkdir -p {{out_dir}}/linux-arm-native
    CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build {{ldflags}} -tags native -o {{out_dir}}/linux-arm-native/goqlprinter .

build-windows-native: build-frontend
    mkdir -p {{out_dir}}/windows-native
    CGO_ENABLED=0 GOOS=windows go build {{ldflags}} -tags native -o {{out_dir}}/windows-native/goqlprinter.exe .

build-darwin-native: build-frontend
    mkdir -p {{out_dir}}/darwin-native-amd64 {{out_dir}}/darwin-native-arm64
    CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build {{ldflags}} -tags native -o {{out_dir}}/darwin-native-amd64/goqlprinter .
    CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build {{ldflags}} -tags native -o {{out_dir}}/darwin-native-arm64/goqlprinter .

# Build all platform targets
build-all: build-linux-usb build-windows-usb build-linux-native build-linux-arm-native build-windows-native build-darwin-native
    @echo "All targets built successfully"

# ─── Packaging & Cleanup ───────────────────────────────

# Package dist/ binaries as .tar.gz
package:
    #!/usr/bin/env bash
    for dir in {{out_dir}}/*/; do
        target=$(basename "$dir")
        archive="{{out_dir}}/goqlprinter-{{version}}-${target}.tar.gz"
        echo "Packaging $archive"
        tar -czf "$archive" -C "$dir" .
    done
    echo "Packages written to {{out_dir}}/"

# Remove build artifacts
clean:
    rm -rf {{out_dir}} goqlprinter goqlprinter.exe

# ─── Utilities ──────────────────────────────────────────

# Check that required tools are available
check-deps:
    @command -v go   >/dev/null 2>&1 || { echo "ERROR: go not found"; exit 1; }
    @command -v npm  >/dev/null 2>&1 || { echo "ERROR: npm not found"; exit 1; }
    @command -v just >/dev/null 2>&1 || { echo "ERROR: just not found"; exit 1; }
    @echo "OK: go $(go version | awk '{print $3}'), npm $(npm --version)"
