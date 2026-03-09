# Brother QL Printer Backend
version := `git describe --tags --always --dirty 2>/dev/null || echo "dev"`
build_date := `date +%Y-%m-%dT%H:%M:%S%z`
ldflags := "-ldflags \"-X main.version=" + version + " -X main.buildDate=" + build_date + "\""
out_dir := "dist"
build_deps := justfile_directory() / "build_deps"

# Default: build for current platform
default: build

build:
    go build {{ldflags}} -o goqlprinter .

run:
    go run .

# Check that required tools are available
check-deps:
    @command -v go   >/dev/null 2>&1 || { echo "ERROR: go not found"; exit 1; }
    @command -v npm  >/dev/null 2>&1 || { echo "ERROR: npm not found"; exit 1; }
    @command -v just >/dev/null 2>&1 || { echo "ERROR: just not found"; exit 1; }
    @echo "OK: go $(go version | awk '{print $3}'), npm $(npm --version)"

# Frontend
install-frontend:
    cd frontend && npm install

build-frontend:
    cd frontend && npm run build

# Dev: Go backend + Vite devserver concurrently (Ctrl+C stops both)
dev:
    #!/usr/bin/env bash
    trap 'kill 0' SIGINT
    go run . &
    cd frontend && npm run dev &
    wait

# Cross-compilation: USB backend (CGO required)
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

# Cross-compilation: Native backend (pure Go, no CGO)
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

build-all: build-linux-usb build-windows-usb build-linux-native build-linux-arm-native build-windows-native build-darwin-native
    @echo "All targets built successfully"

# Package dist/ binaries as .tar.gz (one per target directory)
package:
    #!/usr/bin/env bash
    for dir in {{out_dir}}/*/; do
        target=$(basename "$dir")
        archive="{{out_dir}}/goqlprinter-{{version}}-${target}.tar.gz"
        echo "Packaging $archive"
        tar -czf "$archive" -C "$dir" .
    done
    echo "Packages written to {{out_dir}}/"

clean:
    rm -rf {{out_dir}} goqlprinter goqlprinter.exe
