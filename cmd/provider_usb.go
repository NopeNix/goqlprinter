//go:build usb

package cmd

import "goqlprinter/brotherql"

func initUSBProvider() brotherql.BackendProvider {
	return brotherql.NewUSBProvider()
}
