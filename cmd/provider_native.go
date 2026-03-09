//go:build !usb

package cmd

import (
	"log/slog"

	"goqlprinter/brotherql"
)

func initUSBProvider() brotherql.BackendProvider {
	slog.Info("USB backend not available in this build, using native backend")
	return brotherql.NewNativeProvider()
}
