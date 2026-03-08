//go:build usb

package main

import "goqlprinter/brotherql"

// initUSBProvider creates a USB backend provider (only available in USB builds)
func initUSBProvider() brotherql.BackendProvider {
	return brotherql.NewUSBProvider()
}
