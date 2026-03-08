//go:build !usb

package main

import (
	"log/slog"

	"goqlprinter/brotherql"
)

// initUSBProvider is not available in native-only builds
// This stub ensures compilation works but returns native provider instead
func initUSBProvider() brotherql.BackendProvider {
	slog.Warn("USB backend requested but not available in this build, using native backend instead")
	return createNativeProvider()
}
