//go:build usb

package services

import (
	"fmt"
	"log/slog"

	"github.com/google/gousb"

	"goqlprinter/brotherql"
)

// ConnectToPrinter handles USB printer connection using gousb.
// It uses the PrinterService to resolve the printer identifier.
func ConnectToPrinter(svc *PrinterService, printerIdentifier, modelOverride string, handler PrinterHandler) error {
	printerLock.Lock()
	defer printerLock.Unlock()

	resolvedPrinter, modelToUse, err := connectCommon(svc, printerIdentifier, modelOverride)
	if err != nil {
		return err
	}

	if modelToUse == "" {
		return fmt.Errorf("printer model not specified")
	}

	// Look up the USB product ID from the model name.
	// This is more reliable than bus/address which can change on macOS
	// when kernel drivers are detached/reattached.
	productID, ok := brotherql.PrinterProductIDs[resolvedPrinter.Model]
	if !ok {
		return fmt.Errorf("unknown printer model: %s", resolvedPrinter.Model)
	}

	ctx := gousb.NewContext()
	defer func() {
		if cerr := ctx.Close(); cerr != nil {
			slog.Warn("failed to close USB context", "error", cerr)
		}
	}()

	devices, err := ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		return desc.Vendor == brotherql.BrotherVendorID &&
			desc.Product == gousb.ID(productID)
	})
	if err != nil {
		return fmt.Errorf("failed to open USB device: %w", err)
	}
	if len(devices) == 0 {
		return fmt.Errorf("USB device %s not found", resolvedPrinter.Model)
	}

	dev := devices[0]
	defer func() {
		if cerr := dev.Close(); cerr != nil {
			slog.Warn("failed to close USB device", "error", cerr)
		}
	}()

	for _, d := range devices[1:] {
		if cerr := d.Close(); cerr != nil {
			slog.Warn("failed to close extra USB device", "error", cerr)
		}
	}

	if err := dev.SetAutoDetach(true); err != nil {
		return fmt.Errorf("failed to set auto-detach: %w", err)
	}

	backend, err := brotherql.NewUSBBackend(dev)
	if err != nil {
		return fmt.Errorf("failed to create USB backend: %w", err)
	}
	defer func() {
		if cerr := backend.Close(); cerr != nil {
			slog.Warn("failed to close USB backend", "error", cerr)
		}
	}()

	return handler(backend, modelToUse)
}
