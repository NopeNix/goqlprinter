package cmd

import (
	"log/slog"

	"goqlprinter/brotherql"
	icfg "goqlprinter/internal/config"
)

// InitBackendProvider selects and initializes the appropriate backend provider.
func InitBackendProvider(cfg *icfg.Config) brotherql.BackendProvider {
	switch cfg.App.Backend {
	case "usb":
		slog.Info("Using USB backend (gousb/libusb)")
		return initUSBProvider()
	case "native":
		slog.Info("Using native OS backend")
		return brotherql.NewNativeProvider()
	case "auto":
		return autoDetectProvider()
	default:
		slog.Warn("Unknown backend, falling back to auto mode", "backend", cfg.App.Backend)
		return autoDetectProvider()
	}
}

func autoDetectProvider() brotherql.BackendProvider {
	slog.Info("Auto mode: trying USB backend first")
	usbProvider := initUSBProvider()
	if usbProvider != nil {
		printers, err := usbProvider.FindPrinters()
		if err == nil && len(printers) > 0 {
			slog.Info("USB backend found printers", "count", len(printers))
			return usbProvider
		}
		slog.Info("USB backend found no printers, trying native backend")
	}

	nativeProvider := brotherql.NewNativeProvider()
	printers, err := nativeProvider.FindPrinters()
	if err == nil && len(printers) > 0 {
		slog.Info("Native backend found printers", "count", len(printers))
		return nativeProvider
	}

	slog.Info("No printers found with any backend")
	return nativeProvider
}
