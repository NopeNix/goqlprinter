package services

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"goqlprinter/brotherql"
)

// FoundPrinter holds information about a discovered printer.
type FoundPrinter struct {
	Model string `json:"name"`
	UID   string `json:"id"`
}

// PrinterService manages printer discovery and connection, with all state encapsulated.
type PrinterService struct {
	mu             sync.Mutex
	provider       brotherql.BackendProvider
	defaultPrinter *FoundPrinter
}

// NewPrinterService creates a new PrinterService with the given BackendProvider.
func NewPrinterService(provider brotherql.BackendProvider) *PrinterService {
	return &PrinterService{provider: provider}
}

// InitializeDefaultPrinter finds and sets the default printer for the session.
func (s *PrinterService) InitializeDefaultPrinter(configuredName string) {
	printers, err := s.FindPrinters()
	if err != nil {
		slog.Error("Error finding printers during initialization", "error", err)
		return
	}
	if len(printers) == 0 {
		slog.Warn("No Brother QL printers found")
		return
	}

	// 1. Try to find the printer from config
	if configuredName != "" {
		for _, p := range printers {
			if p.Model == configuredName {
				slog.Info("Default printer set from config", "model", p.Model, "uid", p.UID)
				s.mu.Lock()
				s.defaultPrinter = &p
				s.mu.Unlock()
				return
			}
		}
		slog.Warn("Configured default printer not found, falling back to first available", "configured", configuredName)
	}

	// 2. Fallback: use the first available printer
	p := printers[0]
	slog.Info("Default printer set to first available", "model", p.Model, "uid", p.UID)
	s.mu.Lock()
	s.defaultPrinter = &p
	s.mu.Unlock()
}

// FindPrinters scans for connected Brother printers.
func (s *PrinterService) FindPrinters() ([]FoundPrinter, error) {
	slog.Debug("Discovering printers using provider", "type", fmt.Sprintf("%T", s.provider))

	printerInfos, err := s.provider.FindPrinters()
	if err != nil {
		slog.Error("Provider failed to find printers", "error", err)
		return nil, fmt.Errorf("failed to discover printers: %w", err)
	}

	slog.Info("Found printers via provider", "count", len(printerInfos))

	foundPrinters := make([]FoundPrinter, 0, len(printerInfos))
	for _, info := range printerInfos {
		foundPrinter := FoundPrinter{
			Model: info.Model,
			UID:   info.URI,
		}
		foundPrinters = append(foundPrinters, foundPrinter)
		slog.Debug("Found printer", "model", info.Model, "uri", info.URI, "backend", info.Backend)
	}

	return foundPrinters, nil
}

// ResolvePrinter finds a printer by identifier (name or UID). Empty → default.
func (s *PrinterService) ResolvePrinter(identifier string) (FoundPrinter, error) {
	if identifier == "" {
		s.mu.Lock()
		def := s.defaultPrinter
		s.mu.Unlock()
		if def != nil {
			return *def, nil
		}
		return FoundPrinter{}, errors.New("no printer specified and no default printer is configured or connected")
	}

	// Check if identifier looks like a URI (USB backend or native /dev/ path)
	if strings.HasPrefix(identifier, "usb:") || strings.HasPrefix(identifier, "/dev/") {
		printers, _ := s.FindPrinters()
		for _, p := range printers {
			if p.UID == identifier {
				return p, nil
			}
		}
		// Can't find model, but can still try to print.
		return FoundPrinter{UID: identifier, Model: "Unknown"}, nil
	}

	// Assume identifier is a model name
	printers, err := s.FindPrinters()
	if err != nil {
		return FoundPrinter{}, fmt.Errorf("could not list printers to resolve name '%s': %w", identifier, err)
	}

	for _, p := range printers {
		if p.Model == identifier {
			return p, nil
		}
	}

	return FoundPrinter{}, fmt.Errorf("printer with name '%s' not found", identifier)
}

// GetDefaultPrinter returns the current default printer (thread-safe).
func (s *PrinterService) GetDefaultPrinter() *FoundPrinter {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.defaultPrinter
}
