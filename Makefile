# Brother QL Printer Backend
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DATE = $(shell date +%Y-%m-%dT%H:%M:%S%z)
LDFLAGS = -ldflags "-X main.version=$(VERSION) -X main.buildDate=$(BUILD_DATE)"
OUT_DIR = dist

.PHONY: all build run clean help \
	build-linux-usb build-windows-usb \
	build-linux-native build-linux-arm-native build-windows-native build-darwin-native \
	build-all \
	install-frontend build-frontend dev

# Default: build for current platform
all: build

build:
	go build $(LDFLAGS) -o goqlprinter .

run:
	go run .

# Frontend targets
install-frontend:
	cd frontend && npm install

build-frontend:
	cd frontend && npm run build

# Dev: run Go backend + Vite devserver concurrently
dev:
	@trap 'kill 0' SIGINT; \
	go run . & \
	cd frontend && npm run dev & \
	wait

# Cross-compilation: USB backend (CGO required)
build-linux-usb:
	mkdir -p $(OUT_DIR)/linux-usb
	CGO_ENABLED=1 GOOS=linux go build $(LDFLAGS) -tags usb -o $(OUT_DIR)/linux-usb/goqlprinter .

build-windows-usb:
	mkdir -p $(OUT_DIR)/windows-usb
	CGO_ENABLED=1 GOOS=windows CC=x86_64-w64-mingw32-gcc go build $(LDFLAGS) -tags usb -o $(OUT_DIR)/windows-usb/goqlprinter.exe .

# Cross-compilation: Native backend (pure Go, no CGO)
build-linux-native:
	mkdir -p $(OUT_DIR)/linux-native
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -tags native -o $(OUT_DIR)/linux-native/goqlprinter .

build-linux-arm-native:
	mkdir -p $(OUT_DIR)/linux-arm-native
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -tags native -o $(OUT_DIR)/linux-arm-native/goqlprinter .

build-windows-native:
	mkdir -p $(OUT_DIR)/windows-native
	CGO_ENABLED=0 GOOS=windows go build $(LDFLAGS) -tags native -o $(OUT_DIR)/windows-native/goqlprinter.exe .

build-darwin-native:
	mkdir -p $(OUT_DIR)/darwin-native-amd64 $(OUT_DIR)/darwin-native-arm64
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -tags native -o $(OUT_DIR)/darwin-native-amd64/goqlprinter .
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -tags native -o $(OUT_DIR)/darwin-native-arm64/goqlprinter .

build-all: build-linux-usb build-windows-usb build-linux-native build-linux-arm-native build-windows-native build-darwin-native
	@echo "All targets built successfully"

clean:
	rm -rf $(OUT_DIR) goqlprinter goqlprinter.exe

help:
	@echo "Brother QL Printer Backend"
	@echo ""
	@echo "  make build              - Build for current platform"
	@echo "  make run                - Run development server"
	@echo "  make clean              - Remove build artifacts"
	@echo ""
	@echo "  USB Backend (CGO, requires libusb):"
	@echo "    make build-linux-usb    - Linux"
	@echo "    make build-windows-usb  - Windows"
	@echo ""
	@echo "  Native Backend (Pure Go):"
	@echo "    make build-linux-native     - Linux amd64"
	@echo "    make build-linux-arm-native - Linux arm64"
	@echo "    make build-windows-native   - Windows"
	@echo "    make build-darwin-native    - macOS (amd64 + arm64)"
	@echo ""
	@echo "  make build-all          - Build all cross-compilation targets"
	@echo ""
	@echo "  Frontend:"
	@echo "    make install-frontend   - npm install"
	@echo "    make build-frontend     - npm run build (output to frontend/dist)"
	@echo "    make dev                - Go backend + Vite devserver concurrently"
