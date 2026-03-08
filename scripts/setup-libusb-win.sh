#!/bin/bash
# Build libusb from source for Windows cross-compilation (mingw-w64).
# Output: build_deps/libusb-win/  (relative to repo root)
# Run once before: make build-windows-usb
set -euo pipefail

VERSION="1.0.29"
REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="${REPO_ROOT}/build_deps/libusb-win"
TMP=$(mktemp -d)
trap 'rm -rf "${TMP}"' EXIT

echo "Installing required packages..."
sudo apt-get update -q
sudo apt-get install -y autoconf automake libtool pkg-config mingw-w64 wget

echo "Downloading libusb ${VERSION}..."
wget -q -O "${TMP}/libusb-${VERSION}.tar.bz2" \
    "https://github.com/libusb/libusb/releases/download/v${VERSION}/libusb-${VERSION}.tar.bz2"
tar -xf "${TMP}/libusb-${VERSION}.tar.bz2" -C "${TMP}"
cd "${TMP}/libusb-${VERSION}"

echo "Configuring for x86_64-w64-mingw32..."
./configure --host=x86_64-w64-mingw32 --prefix="${OUT}" --enable-shared
make -j"$(nproc)"
make install

echo ""
echo "Done: libusb ${VERSION} installed to ${OUT}"
echo "You can now run: make build-windows-usb"
