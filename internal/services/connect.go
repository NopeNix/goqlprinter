package services

import (
	"fmt"
	"sync"
)

// printerLock prevents concurrent access to the same printer.
// Two concurrent POST /api/print requests must not interleave commands.
var printerLock sync.Mutex

// connectCommon handles the shared logic for ConnectToPrinter:
// resolve printer, validate, apply model override.
// It returns the resolved printer info and the model to use.
func connectCommon(svc *PrinterService, printerIdentifier, modelOverride string) (FoundPrinter, string, error) {
	resolvedPrinter, err := svc.ResolvePrinter(printerIdentifier)
	if err != nil {
		return FoundPrinter{}, "", fmt.Errorf("printer resolution error: %w", err)
	}

	if resolvedPrinter.UID == "file" {
		return FoundPrinter{}, "", fmt.Errorf("file printer cannot be connected")
	}

	modelToUse := resolvedPrinter.Model
	if modelOverride != "" {
		modelToUse = modelOverride
	}

	return resolvedPrinter, modelToUse, nil
}
