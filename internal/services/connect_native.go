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

	// Convert FoundPrinter -> PrinterInfo. Infer the backend type from
	// the URI scheme so a network URI uses the network backend, a
	// /dev/ path uses the native backend, and an usb: URI looks up
	// the actual discovered device.
	printerInfo := brotherql.PrinterInfo{
		Name:    resolvedPrinter.Model,
		Model:   resolvedPrinter.Model,
		URI:     resolvedPrinter.UID,
		Backend: brotherql.BackendNative,
	}

	switch {
	case strings.HasPrefix(resolvedPrinter.UID, "tcp://"),
		strings.HasPrefix(resolvedPrinter.UID, "network://"):
		printerInfo.Backend = brotherql.BackendNetwork
	case strings.HasPrefix(resolvedPrinter.UID, "usb:"):
		// usb: URIs need to be re-resolved against actual discovery
		// because bus/address can change between scans.
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
