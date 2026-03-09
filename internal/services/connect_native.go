//go:build !usb

package services

import (
	"fmt"
	"log/slog"
	"strings"

	"goqlprinter/brotherql"
)

// ConnectToPrinter handles printer connection using native OS backend.
// It uses the PrinterService to resolve the printer and the embedded provider to connect.
func ConnectToPrinter(svc *PrinterService, printerIdentifier, modelOverride string, handler PrinterHandler) error {
	printerLock.Lock()
	defer printerLock.Unlock()

	if svc == nil || svc.provider == nil {
		return fmt.Errorf("no backend provider configured")
	}

	resolvedPrinter, modelToUse, err := connectCommon(svc, printerIdentifier, modelOverride)
	if err != nil {
		return err
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
		printers, err := svc.provider.FindPrinters()
		if err != nil {
			return fmt.Errorf("failed to find printers: %w", err)
		}

		found := false
		for _, p := range printers {
			if p.Model == resolvedPrinter.Model {
				printerInfo = p
				found = true
				break
			}
		}
		if !found && len(printers) > 0 {
			printerInfo = printers[0]
		}
	}

	backend, err := svc.provider.Connect(printerInfo)
	if err != nil {
		return fmt.Errorf("failed to connect to printer: %w", err)
	}
	defer func() {
		if cerr := backend.Close(); cerr != nil {
			slog.Warn("failed to close backend", "error", cerr)
		}
	}()

	return handler(backend, modelToUse)
}
