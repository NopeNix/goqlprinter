//go:build windows

package brotherql

import (
	"errors"
	"fmt"
	"strings"

	"github.com/alexbrainman/printer"
)

var (
	// ErrNoActiveJob indicates Write() hasn't been called to start a print job
	ErrNoActiveJob = errors.New("no active print job")
)

// WindowsBackend implements Backend and StatusProvider using Windows WinSpool API
type WindowsBackend struct {
	printerHandle *printer.Printer
	printerName   string
	jobID         int // Track active job ID for potential bidirectional communication
}

// WindowsProvider implements BackendProvider for Windows WinSpool API
type WindowsProvider struct{}

// NewWindowsProvider creates a new Windows backend provider
func NewWindowsProvider() *WindowsProvider {
	return &WindowsProvider{}
}

// Write sends RAW data to the printer via WinSpool API
func (b *WindowsBackend) Write(data []byte) (int, error) {
	if b.printerHandle == nil {
		return 0, errors.New("printer not opened")
	}

	// Write raw data to printer
	n, err := b.printerHandle.Write(data)
	if err != nil {
		return n, fmt.Errorf("failed to write to printer: %w", err)
	}

	return n, nil
}

// Read attempts to read status from printer
// Windows WinSpool API doesn't support bidirectional ESC i S protocol directly
// Status must be queried via GetStatus() instead
func (b *WindowsBackend) Read(data []byte) (int, error) {
	// Windows doesn't support direct bidirectional Read() for ESC i S protocol
	// Use GetStatus() method instead which queries WinSpool API
	return 0, ErrStatusNotSupported
}

// Close closes the printer connection
func (b *WindowsBackend) Close() error {
	if b.printerHandle == nil {
		return nil
	}

	err := b.printerHandle.Close()
	b.printerHandle = nil
	b.jobID = 0
	return err
}

// GetStatus retrieves printer status from Windows WinSpool API
// Implements StatusProvider interface
// Note: The alexbrainman/printer package doesn't support querying printer status directly.
// Windows WinSpool would require direct syscalls to GetPrinter() API.
// For now, we return a basic "ready" status since the printer connection is open.
func (b *WindowsBackend) GetStatus() (PrinterStatus, error) {
	if b.printerHandle == nil {
		return PrinterStatus{}, errors.New("printer not opened")
	}

	// The alexbrainman/printer package doesn't expose GetPrinter() or status queries.
	// Return basic status - if we have an open handle, assume printer is ready.
	// Detailed status (paper_out, paper_jam, etc.) would require:
	// 1. Direct Windows API calls via syscall/windows package, or
	// 2. Bidirectional ESC i S protocol (not supported over WinSpool)
	status := PrinterStatus{
		Ready:      true,
		Busy:       false,
		Error:      "",
		MediaType:  "", // Not available via WinSpool
		MediaWidth: 0,  // Not available via WinSpool
	}

	return status, nil
}

// FindPrinters discovers Brother QL printers installed in Windows
// Implements BackendProvider interface
func (p *WindowsProvider) FindPrinters() ([]PrinterInfo, error) {
	// Enumerate all installed printers via WinSpool API
	printerNames, err := printer.ReadNames()
	if err != nil {
		return nil, fmt.Errorf("failed to enumerate printers: %w", err)
	}

	var printers []PrinterInfo
	for _, name := range printerNames {
		// Filter for Brother QL printers
		if isBrotherPrinter(name) {
			printers = append(printers, PrinterInfo{
				Name:    name,
				Model:   extractModel(name),
				URI:     name, // Windows uses printer name as URI
				Backend: BackendNative,
			})
		}
	}

	return printers, nil
}

// Connect opens a connection to a Windows printer
// Implements BackendProvider interface
func (p *WindowsProvider) Connect(printerInfo PrinterInfo) (Backend, error) {
	// Open printer using WinSpool API
	handle, err := printer.Open(printerInfo.URI)
	if err != nil {
		return nil, fmt.Errorf("failed to open printer '%s': %w", printerInfo.URI, err)
	}

	// Start a RAW print job
	// This is required for sending binary Brother QL protocol commands
	err = handle.StartDocument("Brother QL Print Job", "RAW")
	if err != nil {
		handle.Close()
		return nil, fmt.Errorf("failed to start print job: %w", err)
	}

	backend := &WindowsBackend{
		printerHandle: handle,
		printerName:   printerInfo.Name,
		jobID:         0, // Job ID not currently tracked
	}

	return backend, nil
}

// SupportsStatus indicates Windows backend supports status queries via WinSpool
// Implements BackendProvider interface
func (p *WindowsProvider) SupportsStatus() bool {
	// Windows supports basic status via GetStatus() method
	// However, it doesn't support bidirectional Read() for ESC i S protocol
	return true
}

// Helper functions for Windows backend

// isBrotherPrinter is imported from native_backend.go
// extractModel is imported from native_backend.go

// Additional helper to normalize printer names
func normalizePrinterName(name string) string {
	// Remove common suffixes that might interfere with model detection
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "  ", " ")
	return name
}
