# Brother QL Printer Backend
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DATE = $(shell date +%Y-%m-%dT%H:%M:%S%z)
LDFLAGS = -ldflags "-X main.version=$(VERSION) -X main.buildDate=$(BUILD_DATE)"
OUT_DIR = dist
BUILD_DEPS = $(PWD)/build_deps

.PHONY: all build run clean help check-deps \
	install-frontend build-frontend dev \
	build-linux-usb build-windows-usb \
	build-linux-native build-linux-arm-native build-windows-native build-darwin-native \
	build-all package

# Default: build for current platform (requires frontend/dist to exist)
all: build

build:
	go build $(LDFLAGS) -o goqlprinter .

run:
	go run .

# Check that required tools are available
check-deps:
	@command -v go   >/dev/null 2>&1 || { echo "ERROR: go not found"; exit 1; }
	@command -v npm  >/dev/null 2>&1 || { echo "ERROR: npm not found"; exit 1; }
	@command -v make >/dev/null 2>&1 || { echo "ERROR: make not found"; exit 1; }
	@echo "OK: go $(shell go version | awk '{print $$3}'), npm $(shell npm --version)"

# Frontend
install-frontend:
	cd frontend && npm install

build-frontend:
	cd frontend && npm run build

# Dev: Go backend + Vite devserver concurrently (Ctrl+C stops both)
dev:
	@trap 'kill 0' SIGINT; \
	go run . & \
	cd frontend && npm run dev & \
	wait

# Cross-compilation: USB backend (CGO required)
# Run scripts/setup-libusb-arm.sh or scripts/install-libusb-win.sh first.
# Dependencies are stored in build_deps/ (gitignored).
build-linux-usb: build-frontend
	mkdir -p $(OUT_DIR)/linux-usb
	CGO_ENABLED=1 GOOS=linux \
	CGO_CFLAGS="-I$(BUILD_DEPS)/libusb-arm/include" \
	CGO_LDFLAGS="-L$(BUILD_DEPS)/libusb-arm/lib -lusb-1.0" \
	go build $(LDFLAGS) -tags usb -o $(OUT_DIR)/linux-usb/goqlprinter .

build-windows-usb: build-frontend
	mkdir -p $(OUT_DIR)/windows-usb
	CGO_ENABLED=1 GOOS=windows CC=x86_64-w64-mingw32-gcc \
	CGO_CFLAGS="-I$(BUILD_DEPS)/libusb-win/include" \
	CGO_LDFLAGS="-L$(BUILD_DEPS)/libusb-win/lib -lusb-1.0" \
	go build $(LDFLAGS) -tags usb -o $(OUT_DIR)/windows-usb/goqlprinter.exe .

# Cross-compilation: Native backend (pure Go, no CGO)
build-linux-native: build-frontend
	mkdir -p $(OUT_DIR)/linux-native
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -tags native -o $(OUT_DIR)/linux-native/goqlprinter .

build-linux-arm-native: build-frontend
	mkdir -p $(OUT_DIR)/linux-arm-native
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -tags native -o $(OUT_DIR)/linux-arm-native/goqlprinter .

build-windows-native: build-frontend
	mkdir -p $(OUT_DIR)/windows-native
	CGO_ENABLED=0 GOOS=windows go build $(LDFLAGS) -tags native -o $(OUT_DIR)/windows-native/goqlprinter.exe .

build-darwin-native: build-frontend
	mkdir -p $(OUT_DIR)/darwin-native-amd64 $(OUT_DIR)/darwin-native-arm64
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -tags native -o $(OUT_DIR)/darwin-native-amd64/goqlprinter .
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -tags native -o $(OUT_DIR)/darwin-native-arm64/goqlprinter .

build-all: build-linux-usb build-windows-usb build-linux-native build-linux-arm-native build-windows-native build-darwin-native
	@echo "All targets built successfully"

# Package dist/ binaries as .tar.gz (one per target directory)
package:
	@for dir in $(OUT_DIR)/*/; do \
		target=$$(basename "$$dir"); \
		archive="$(OUT_DIR)/goqlprinter-$(VERSION)-$${target}.tar.gz"; \
		echo "Packaging $$archive"; \
		tar -czf "$$archive" -C "$$dir" .; \
	done
	@echo "Packages written to $(OUT_DIR)/"

clean:
	rm -rf $(OUT_DIR) goqlprinter goqlprinter.exe

help:
	@echo "Brother QL Printer Backend"
	@echo ""
	@echo "  make build              - Build for current platform"
	@echo "  make run                - Run development server"
	@echo "  make check-deps         - Verify required tools (go, npm, make)"
	@echo "  make clean              - Remove build artifacts"
	@echo ""
	@echo "  Frontend:"
	@echo "    make install-frontend   - npm install"
	@echo "    make build-frontend     - npm run build  →  frontend/dist/"
	@echo "    make dev                - Go :8000 + Vite :5173 concurrently"
	@echo ""
	@echo "  USB Backend (CGO, requires libusb — see scripts/):"
	@echo "    make build-linux-usb    - Linux amd64"
	@echo "    make build-windows-usb  - Windows amd64"
	@echo ""
	@echo "  Native Backend (Pure Go):"
	@echo "    make build-linux-native     - Linux amd64"
	@echo "    make build-linux-arm-native - Linux arm64"
	@echo "    make build-windows-native   - Windows amd64"
	@echo "    make build-darwin-native    - macOS amd64 + arm64"
	@echo ""
	@echo "  make build-all          - All cross-compilation targets (builds frontend once)"
	@echo "  make package            - Package dist/ binaries as .tar.gz"
	@echo ""
	@echo "  libusb setup (run once before USB builds):"
	@echo "    scripts/setup-libusb-arm.sh    - Build libusb for ARM64 Linux"
	@echo "    scripts/setup-libusb-win.sh    - Build libusb for Windows (mingw)"
	@echo "    scripts/install-libusb-win.sh  - Install pre-built libusb for Windows"
