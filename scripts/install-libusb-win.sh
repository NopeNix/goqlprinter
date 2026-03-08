#!/bin/bash
# Install pre-built libusb binaries for Windows cross-compilation (mingw-w64).
# Output: build_deps/libusb-win/  (relative to repo root)
# Faster alternative to setup-libusb-win.sh — no compilation required.
# Run once before: make build-windows-usb
set -euo pipefail

VERSION="1.0.29"
REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="${REPO_ROOT}/build_deps/libusb-win"
TMP=$(mktemp -d)
trap 'rm -rf "${TMP}"' EXIT

echo "Installing required packages..."
sudo apt-get update -q
sudo apt-get install -y mingw-w64 p7zip-full wget

echo "Downloading libusb ${VERSION} Windows binaries..."
wget -q -O "${TMP}/libusb-${VERSION}.7z" \
    "https://github.com/libusb/libusb/releases/download/v${VERSION}/libusb-${VERSION}.7z"
7z x "${TMP}/libusb-${VERSION}.7z" -o"${TMP}" -y >/dev/null

echo "Installing to ${OUT}..."
mkdir -p "${OUT}/lib" "${OUT}/include/libusb-1.0"
cp "${TMP}/MinGW64/dll/libusb-1.0.dll.a" "${OUT}/lib/"
cp "${TMP}/include/libusb-1.0/"*.h "${OUT}/include/libusb-1.0/"

echo ""
echo "Done: libusb ${VERSION} installed to ${OUT}"
echo "You can now run: make build-windows-usb"
