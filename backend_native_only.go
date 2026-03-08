//go:build !usb

package main

import (
	"goqlprinter/brotherql"
	"goqlprinter/logger"
)

// initUSBProvider is not available in native-only builds
// This stub ensures compilation works but returns native provider instead
func initUSBProvider() brotherql.BackendProvider {
	logger.Warning("USB backend requested but not available in this build, using native backend instead")
	return createNativeProvider()
}
