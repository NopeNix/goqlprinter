//go:build !usb

package services

import (
	"goqlprinter/brotherql"
	"fmt"
	"strings"
)

// ConnectToPrinter handles printer connection using native OS backend
func ConnectToPrinter(printerIdentifier, modelOverride string, handler PrinterHandler) error {
	if defaultProvider == nil {
		return fmt.Errorf("no backend provider configured")
	}

	resolvedPrinter, err := ResolvePrinter(printerIdentifier)
	if err != nil {
		return fmt.Errorf("printer resolution error: %w", err)
	}

	if resolvedPrinter.UID == "file" {
		return fmt.Errorf("file printer cannot be connected")
	}

	// Convert FoundPrinter -> PrinterInfo
	printerInfo := brotherql.PrinterInfo{
		Name:    resolvedPrinter.Model,
		Model:   resolvedPrinter.Model,
		URI:     resolvedPrinter.UID,
		Backend: brotherql.BackendNative,
	}

	// If UID is usb:-formatted, try via native backend
	if strings.HasPrefix(resolvedPrinter.UID, "usb:") {
		// Native Linux uses /dev/usb/lp* paths
		// Map usb:bus:addr -> /dev/usb/lpN
		// Or find printer again via native provider
		printers, err := defaultProvider.FindPrinters()
		if err != nil {
			return fmt.Errorf("failed to find printers: %w", err)
		}

		// Find matching printer by model
		found := false
		for _, p := range printers {
			if p.Model == resolvedPrinter.Model {
				printerInfo = p
				found = true
				break
			}
		}
		if !found && len(printers) > 0 {
			printerInfo = printers[0] // Use first
		}
	}

	backend, err := defaultProvider.Connect(printerInfo)
	if err != nil {
		return fmt.Errorf("failed to connect to printer: %w", err)
	}
	defer backend.Close()

	modelToUse := resolvedPrinter.Model
	if modelOverride != "" {
		modelToUse = modelOverride
	}

	return handler(backend, modelToUse)
}
