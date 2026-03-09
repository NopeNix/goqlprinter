//go:build usb

package services

import (
	"fmt"
	"strconv"
	"strings"

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

	if !strings.HasPrefix(resolvedPrinter.UID, "usb:") {
		return fmt.Errorf("unsupported printer format: %s", resolvedPrinter.UID)
	}

	parts := strings.Split(resolvedPrinter.UID, ":")
	if len(parts) != 3 {
		return fmt.Errorf("invalid printer UID format: %s", resolvedPrinter.UID)
	}

	bus, err := strconv.Atoi(parts[1])
	if err != nil {
		return fmt.Errorf("invalid bus number: %s", parts[1])
	}

	address, err := strconv.Atoi(parts[2])
	if err != nil {
		return fmt.Errorf("invalid address number: %s", parts[2])
	}

	ctx := gousb.NewContext()
	defer ctx.Close()

	devices, err := ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		return desc.Bus == bus && desc.Address == address
	})
	if err != nil {
		return fmt.Errorf("failed to open USB device: %w", err)
	}
	if len(devices) == 0 {
		return fmt.Errorf("USB device not found: bus %d, address %d", bus, address)
	}

	dev := devices[0]
	defer dev.Close()

	for i, d := range devices {
		if i > 0 {
			d.Close()
		}
	}

	if err := dev.SetAutoDetach(true); err != nil {
		return fmt.Errorf("failed to set auto-detach: %w", err)
	}

	backend, err := brotherql.NewUSBBackend(dev)
	if err != nil {
		return fmt.Errorf("failed to create USB backend: %w", err)
	}
	defer backend.Close()

	if modelToUse == "" {
		return fmt.Errorf("printer model not specified")
	}

	return handler(backend, modelToUse)
}
