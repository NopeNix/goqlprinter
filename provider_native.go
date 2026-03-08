//go:build !usb

package main

import (
	"log/slog"

	"goqlprinter/brotherql"
)

// initUSBProvider is unavailable in native-only builds; falls back to the native provider.
func initUSBProvider() brotherql.BackendProvider {
	slog.Warn("USB backend requested but not available in this build, using native backend instead")
	return createNativeProvider()
}
