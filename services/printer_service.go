package services

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"goqlprinter/brotherql"
)

var (
	// PrinterLock ensures only one operation at a time can access the printer.
	PrinterLock sync.Mutex

	// activeDefaultPrinter is the default printer found at startup
	activeDefaultPrinter *FoundPrinter

	// defaultProvider is the BackendProvider used for printer discovery.
	// This is initialized by SetDefaultProvider() and used by legacy functions
	// that don't accept a provider parameter.
	defaultProvider brotherql.BackendProvider
)

// FoundPrinter holds information about a discovered printer.
// This struct is maintained for backward compatibility with existing API responses.
// Internally, it's mapped from brotherql.PrinterInfo.
type FoundPrinter struct {
	Model string `json:"name"` // E.g. "QL-800"
	UID   string `json:"id"`   // Unique identifier for the device (URI)
}

// SetDefaultProvider sets the BackendProvider to use for printer discovery.
// This should be called during application initialization before any printer
// operations. If not set, FindPrinters() and related functions will return an error.
func SetDefaultProvider(provider brotherql.BackendProvider) {
	defaultProvider = provider
	slog.Info("Default backend provider set", "type", fmt.Sprintf("%T", provider))
}

// FindPrinters scans for connected Brother printers using the default provider.
// This is a legacy function maintained for backward compatibility. New code should
// use FindPrintersWithProvider() to explicitly pass a BackendProvider.
func FindPrinters() ([]FoundPrinter, error) {
	if defaultProvider == nil {
		return nil, errors.New("no backend provider configured - call SetDefaultProvider() first")
	}
	return FindPrintersWithProvider(defaultProvider)
}

// FindPrintersWithProvider scans for connected Brother printers using the specified provider.
// This function uses the BackendProvider interface to discover printers, which allows
// for different backend implementations (USB, native OS, etc.).
//
// The returned FoundPrinter structs are converted from PrinterInfo for API compatibility.
func FindPrintersWithProvider(provider brotherql.BackendProvider) ([]FoundPrinter, error) {
	slog.Debug("Discovering printers using provider", "type", fmt.Sprintf("%T", provider))

	// Use the BackendProvider to find printers
	printerInfos, err := provider.FindPrinters()
	if err != nil {
		slog.Error("Provider failed to find printers", "error", err)
		return nil, fmt.Errorf("failed to discover printers: %w", err)
	}

	slog.Info("Found printers via provider", "count", len(printerInfos))

	// Convert PrinterInfo to FoundPrinter for backward compatibility
	foundPrinters := make([]FoundPrinter, 0, len(printerInfos))
	for _, info := range printerInfos {
		foundPrinter := FoundPrinter{
			Model: info.Model, // Model name (e.g., "QL-800")
			UID:   info.URI,   // Connection URI (e.g., "usb:003:025")
		}
		foundPrinters = append(foundPrinters, foundPrinter)
		slog.Debug("Found printer", "model", info.Model, "uri", info.URI, "backend", info.Backend)
	}

	return foundPrinters, nil
}

// InitializeDefaultPrinter finds and sets the active default printer for the session.
// This function uses the default BackendProvider set via SetDefaultProvider().
// If no provider is configured, it logs a warning and returns without setting a default.
func InitializeDefaultPrinter(configuredName string) {
	if defaultProvider == nil {
		slog.Warn("Cannot initialize default printer: no backend provider configured")
		return
	}

	printers, err := FindPrinters()
	if err != nil {
		slog.Error("Error finding printers during initialization", "error", err)
		return
	}
	if len(printers) == 0 {
		slog.Warn("No Brother QL printers found")
		return
	}

	// 1. Try to find the printer from config.json
	if configuredName != "" {
		for _, p := range printers {
			if p.Model == configuredName {
				slog.Info("Default printer set from config", "model", p.Model, "uid", p.UID)
				activeDefaultPrinter = &p
				return
			}
		}
		slog.Warn("Configured default printer not found, falling back to first available", "configured", configuredName)
	}

	// 2. Fallback: use the first available printer
	p := printers[0]
	slog.Info("Default printer set to first available", "model", p.Model, "uid", p.UID)
	activeDefaultPrinter = &p
}

// GetActiveDefaultPrinter returns the default printer found at startup.
func GetActiveDefaultPrinter() *FoundPrinter {
	return activeDefaultPrinter
}

// ResolvePrinter finds a printer by its identifier (name or UID).
// If identifier is empty, it returns the default printer.
func ResolvePrinter(identifier string) (FoundPrinter, error) {
	if identifier == "" {
		if activeDefaultPrinter != nil {
			return *activeDefaultPrinter, nil
		}
		return FoundPrinter{}, errors.New("no printer specified and no default printer is configured or connected")
	}

	// Check if identifier looks like a URI (USB backend or native /dev/ path)
	if strings.HasPrefix(identifier, "usb:") || strings.HasPrefix(identifier, "/dev/") {
		// It's a URI, find the matching printer
		printers, _ := FindPrinters()
		for _, p := range printers {
			if p.UID == identifier {
				return p, nil
			}
		}
		// Can't find model, but can still try to print. Return a partial struct.
		return FoundPrinter{UID: identifier, Model: "Unknown"}, nil // Model might be provided in request anyway
	}

	// Assume identifier is a model name
	printers, err := FindPrinters()
	if err != nil {
		return FoundPrinter{}, fmt.Errorf("could not list printers to resolve name '%s': %w", identifier, err)
	}

	for _, p := range printers {
		if p.Model == identifier {
			return p, nil // Return first match for the model name
		}
	}

	return FoundPrinter{}, fmt.Errorf("printer with name '%s' not found", identifier)
}
