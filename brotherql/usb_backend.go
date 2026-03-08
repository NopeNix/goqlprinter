//go:build usb || cgo

package brotherql

import (
	"fmt"
	"log"
	"log/slog"

	"github.com/google/gousb"
)

// USBBackend provides a BrotherQL backend for USB printers.
type USBBackend struct {
	ctx         *gousb.Context
	dev         *gousb.Device
	cfg         *gousb.Config
	intf        *gousb.Interface
	inEndpoint  *gousb.InEndpoint
	outEndpoint *gousb.OutEndpoint
}

// NewUSBBackend creates a new USBBackend instance.
func NewUSBBackend(dev *gousb.Device) (*USBBackend, error) {
	// The dev.Reset() call is omitted as it can be unreliable, especially on Windows.

	// Log available configurations for debugging. Brother printers should have 1.
	log.Printf("Info: USB device has %d configuration(s).", len(dev.Desc.Configs))
	for _, cfgDesc := range dev.Desc.Configs {
		log.Printf("  - Configuration #%d", cfgDesc.Number)
	}

	// Auto-detach kernel driver (e.g. usblp) so libusb can claim the interface.
	if err := dev.SetAutoDetach(true); err != nil {
		slog.Warn("failed to set auto-detach on USB device", "error", err)
	}

	// The Brother QL printers use a single configuration and interface.
	cfg, err := dev.Config(1)
	if err != nil {
		return nil, err
	}

	intf, err := cfg.Interface(0, 0)
	if err != nil {
		if cerr := cfg.Close(); cerr != nil {
			slog.Warn("failed to close USB config", "error", cerr)
		}
		return nil, err
	}

	// Find the IN and OUT endpoints.
	var inEndpoint *gousb.InEndpoint
	var outEndpoint *gousb.OutEndpoint
	for _, desc := range intf.Setting.Endpoints {
		if desc.Direction == gousb.EndpointDirectionIn {
			inEndpoint, err = intf.InEndpoint(desc.Number)
		} else {
			outEndpoint, err = intf.OutEndpoint(desc.Number)
		}
		if err != nil {
			intf.Close()
			if cerr := cfg.Close(); cerr != nil {
				slog.Warn("failed to close USB config", "error", cerr)
			}
			return nil, err
		}
	}

	return &USBBackend{
		dev:         dev,
		cfg:         cfg,
		intf:        intf,
		inEndpoint:  inEndpoint,
		outEndpoint: outEndpoint,
	}, nil
}

// Write sends data to the printer.
func (b *USBBackend) Write(data []byte) (int, error) {
	return b.outEndpoint.Write(data)
}

// Read receives data from the printer.
func (b *USBBackend) Read(data []byte) (int, error) {
	return b.inEndpoint.Read(data)
}

// Close releases the USB device.
func (b *USBBackend) Close() error {
	b.intf.Close()
	if err := b.cfg.Close(); err != nil {
		slog.Warn("failed to close USB config", "error", err)
	}
	// The device itself is closed by the caller that opened it, to avoid double-closing.
	return nil
}

// USBProvider implements BackendProvider for USB printer discovery and connection
type USBProvider struct {
	// No state needed - stateless provider
}

// NewUSBProvider creates a new USBProvider instance
func NewUSBProvider() *USBProvider {
	return &USBProvider{}
}

// brotherVendorID is the USB vendor ID for Brother Industries
const brotherVendorID = 0x04f9

// printerModels maps model names to product IDs (from services/printer_service.go)
var printerModels = map[string]int{
	"QL-500":     0x2015,
	"QL-550":     0x2016,
	"QL-560":     0x2027,
	"QL-570":     0x2028,
	"QL-580N":    0x2029,
	"QL-650TD":   0x2017,
	"QL-700":     0x2048,
	"QL-710W":    0x2049,
	"QL-720NW":   0x204a,
	"QL-800":     0x2099,
	"QL-810W":    0x209a,
	"QL-820NWB":  0x209b,
	"QL-1050":    0x2019,
	"QL-1060N":   0x202a,
	"QL-1100":    0x20af,
	"QL-1110NWB": 0x20b0,
}

// FindPrinters discovers all connected Brother USB printers
func (p *USBProvider) FindPrinters() ([]PrinterInfo, error) {
	ctx := gousb.NewContext()
	defer func() {
		if cerr := ctx.Close(); cerr != nil {
			slog.Warn("failed to close USB context", "error", cerr)
		}
	}()

	log.Printf("USB: Scanning for Brother printers (VendorID: 0x%04x)...", brotherVendorID)

	// Open all Brother USB devices
	devices, err := ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		return desc.Vendor == brotherVendorID
	})
	if err != nil {
		log.Printf("USB: Failed to open devices: %v", err)
		return nil, fmt.Errorf("failed to enumerate USB devices: %w", err)
	}

	log.Printf("USB: Found %d Brother USB device(s)", len(devices))

	var printers []PrinterInfo

	// Match devices to known printer models
	for _, dev := range devices {
		productID := int(dev.Desc.Product)

		// Find matching model name
		for modelName, modelProductID := range printerModels {
			if productID == modelProductID {
				// Create URI in format "usb:bus:address" (e.g., "usb:003:025")
				uri := fmt.Sprintf("usb:%03d:%03d", dev.Desc.Bus, dev.Desc.Address)

				printer := PrinterInfo{
					Name:    fmt.Sprintf("%s (USB)", modelName), // Display name
					Model:   modelName,                          // Model identifier
					URI:     uri,                                // Connection identifier
					Backend: BackendUSB,                         // Backend type
				}

				printers = append(printers, printer)
				log.Printf("USB: Found %s at %s", modelName, uri)
				break // Only one model per product ID
			}
		}

		// Close device immediately to release resources
		if err := dev.Close(); err != nil {
			slog.Warn("failed to close USB device", "error", err)
		}
	}

	if len(printers) == 0 {
		log.Printf("USB: No Brother QL printers found. If your printer is connected but not detected (especially on Windows), you may need to install a generic USB driver like WinUSB using Zadig.")
	}

	return printers, nil
}

// Connect opens a connection to the specified printer
func (p *USBProvider) Connect(printer PrinterInfo) (Backend, error) {
	// Parse URI to extract bus and address
	var bus, address int
	_, err := fmt.Sscanf(printer.URI, "usb:%d:%d", &bus, &address)
	if err != nil {
		return nil, fmt.Errorf("invalid USB URI format '%s': %w", printer.URI, err)
	}

	log.Printf("USB: Connecting to %s at bus=%d address=%d", printer.Model, bus, address)

	// Open USB context
	ctx := gousb.NewContext()

	// Find the device by bus and address
	devices, err := ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		return desc.Vendor == brotherVendorID &&
			int(desc.Bus) == bus &&
			int(desc.Address) == address
	})
	if err != nil {
		if cerr := ctx.Close(); cerr != nil {
			slog.Warn("failed to close USB context", "error", cerr)
		}
		return nil, fmt.Errorf("failed to open USB device: %w", err)
	}

	if len(devices) == 0 {
		if cerr := ctx.Close(); cerr != nil {
			slog.Warn("failed to close USB context", "error", cerr)
		}
		return nil, fmt.Errorf("USB device not found at bus=%d address=%d", bus, address)
	}

	if len(devices) > 1 {
		// Close all devices
		for _, dev := range devices {
			if cerr := dev.Close(); cerr != nil {
				slog.Warn("failed to close USB device", "error", cerr)
			}
		}
		if cerr := ctx.Close(); cerr != nil {
			slog.Warn("failed to close USB context", "error", cerr)
		}
		return nil, fmt.Errorf("multiple devices found at bus=%d address=%d (unexpected)", bus, address)
	}

	// Use the first (and only) device
	dev := devices[0]

	// Create USBBackend using existing implementation
	backend, err := NewUSBBackend(dev)
	if err != nil {
		if cerr := dev.Close(); cerr != nil {
			slog.Warn("failed to close USB device", "error", cerr)
		}
		if cerr := ctx.Close(); cerr != nil {
			slog.Warn("failed to close USB context", "error", cerr)
		}
		return nil, fmt.Errorf("failed to initialize USB backend: %w", err)
	}

	// Store context in backend for cleanup
	backend.ctx = ctx

	log.Printf("USB: Successfully connected to %s", printer.Model)
	return backend, nil
}

// SupportsStatus returns true because USB backend supports ESC i S status queries
func (p *USBProvider) SupportsStatus() bool {
	return true
}
