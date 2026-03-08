//go:build darwin

package brotherql

import (
	"bytes"
	"fmt"

	ipp "github.com/phin1x/go-ipp"
)

// DarwinBackend provides macOS CUPS/IPP backend for Brother QL printers
type DarwinBackend struct {
	printerName string
	client      *ipp.IPPClient
}

// DarwinProvider implements BackendProvider for macOS
type DarwinProvider struct{}

// NewDarwinProvider creates a new DarwinProvider instance
func NewDarwinProvider() *DarwinProvider {
	return &DarwinProvider{}
}

// FindPrinters discovers Brother QL printers via CUPS API
func (p *DarwinProvider) FindPrinters() ([]PrinterInfo, error) {
	// Connect to CUPS on localhost:631
	client := ipp.NewIPPClient("localhost", 631, "", "", false)

	// Test CUPS connection
	if err := client.TestConnection(); err != nil {
		return nil, fmt.Errorf("CUPS connection failed: %w", err)
	}

	// Get all printers via CUPS
	// Note: go-ipp doesn't expose CUPS-Get-Printers directly
	// We'll need to enumerate printers using GetPrinterAttributes with wildcard
	// For now, return empty list and document that manual printer name is needed
	// TODO: Implement printer discovery via lower-level IPP requests or exec lpstat

	var printers []PrinterInfo

	// Alternative: Use lpstat command to discover printers
	// This would require exec.Command which we avoid for security
	// Users need to know their printer name from System Preferences

	return printers, nil
}

// Connect opens a connection to a printer via IPP
func (p *DarwinProvider) Connect(printer PrinterInfo) (Backend, error) {
	// Connect to CUPS on localhost:631
	// Note: printer.Name should be the CUPS printer name from System Preferences
	client := ipp.NewIPPClient("localhost", 631, "", "", false)

	// Test connection
	if err := client.TestConnection(); err != nil {
		return nil, fmt.Errorf("CUPS connection failed: %w", err)
	}

	return &DarwinBackend{
		printerName: printer.Name,
		client:      client,
	}, nil
}

// SupportsStatus returns true (macOS supports basic status via IPP)
func (p *DarwinProvider) SupportsStatus() bool {
	return true
}

// Write sends print job via IPP Print-Job operation
func (b *DarwinBackend) Write(data []byte) (int, error) {
	// Create document from raw Brother QL protocol data
	doc := ipp.Document{
		Document: bytes.NewReader(data),
		Size:     len(data),
		Name:     "brother_ql_label.bin",
		MimeType: "application/octet-stream",
	}

	// Job attributes (empty for raw printing)
	jobAttributes := map[string]any{}

	// Send print job to CUPS printer
	jobID, err := b.client.PrintJob(doc, b.printerName, jobAttributes)
	if err != nil {
		return 0, fmt.Errorf("IPP PrintJob failed: %w", err)
	}

	// Log job ID for debugging
	_ = jobID // Job submitted successfully

	return len(data), nil
}

// Read returns ErrStatusNotSupported (cannot read ESC i S via IPP)
func (b *DarwinBackend) Read(data []byte) (int, error) {
	// macOS CUPS/IPP does not support bidirectional ESC i S status queries
	// Use GetStatus() instead for basic printer state
	return 0, ErrStatusNotSupported
}

// Close releases the IPP connection (no-op for stateless IPP)
func (b *DarwinBackend) Close() error {
	// IPP is stateless HTTP-based, no persistent connection to close
	return nil
}

// GetStatus queries printer state via IPP Get-Printer-Attributes
func (b *DarwinBackend) GetStatus() (PrinterStatus, error) {
	// Query printer attributes from CUPS
	attributes := []string{
		"printer-state",
		"printer-state-reasons",
		"printer-state-message",
	}

	attrs, err := b.client.GetPrinterAttributes(b.printerName, attributes)
	if err != nil {
		return PrinterStatus{}, fmt.Errorf("GetPrinterAttributes failed: %w", err)
	}

	// Parse printer-state attribute
	status := PrinterStatus{}

	// Get printer-state (enum: 3=idle, 4=processing, 5=stopped)
	if stateAttrs, ok := attrs["printer-state"]; ok && len(stateAttrs) > 0 {
		if state, ok := stateAttrs[0].Value.(int); ok {
			switch state {
			case 3: // idle
				status.Ready = true
				status.Busy = false
			case 4: // processing
				status.Ready = true
				status.Busy = true
			case 5: // stopped
				status.Ready = false
				status.Busy = false
				status.Error = "stopped"
			}
		}
	}

	// Parse printer-state-reasons for detailed error information
	if reasonAttrs, ok := attrs["printer-state-reasons"]; ok {
		// printer-state-reasons is a 1setOf keyword
		// Common reasons: "none", "media-empty", "media-jam", "offline", "paused"
		for _, attr := range reasonAttrs {
			if reason, ok := attr.Value.(string); ok {
				switch reason {
				case "media-empty", "media-empty-error", "media-empty-warning":
					status.Error = "paper_out"
					status.Ready = false
				case "media-jam", "media-jam-error", "media-jam-warning":
					status.Error = "paper_jam"
					status.Ready = false
				case "offline", "offline-report":
					status.Error = "offline"
					status.Ready = false
				case "paused", "moving-to-paused":
					status.Error = "paused"
					status.Ready = false
				case "spool-area-full":
					status.Error = "error"
					status.Ready = false
				}
			}
		}
	}

	// Note: MediaType and MediaWidth are not available via IPP
	// Brother-specific status requires direct USB ESC i S communication
	status.MediaType = ""
	status.MediaWidth = 0

	return status, nil
}

// init registers DarwinProvider if building for macOS
func init() {
	// This could be used to auto-register the provider in a backend registry
	// For now, the provider must be explicitly created by the application
}
