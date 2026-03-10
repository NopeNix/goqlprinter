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
	client      *ipp.CUPSClient
}

// DarwinProvider implements BackendProvider for macOS
type DarwinProvider struct{}

// NewDarwinProvider creates a new DarwinProvider instance
func NewDarwinProvider() *DarwinProvider {
	return &DarwinProvider{}
}

// newCUPSClient creates a CUPS client using unix socket (preferred) with HTTP fallback
func newCUPSClient() (*ipp.CUPSClient, error) {
	// Try unix socket first (modern macOS default)
	socketAdapter := ipp.NewSocketAdapter("localhost", false)
	// Skip cert reading — printer discovery and printing don't require CUPS auth,
	// and the default cert paths (/etc/cups/certs/0) require root on macOS.
	socketAdapter.CertSearchPaths = nil
	if err := socketAdapter.TestConnection(); err == nil {
		return ipp.NewCUPSClientWithAdapter("", socketAdapter), nil
	}

	// Fall back to TCP (older macOS or custom CUPS config)
	httpClient := ipp.NewCUPSClient("localhost", 631, "", "", false)
	if err := httpClient.TestConnection(); err != nil {
		return nil, fmt.Errorf("CUPS connection failed (tried unix socket and TCP:631): %w", err)
	}
	return httpClient, nil
}

// FindPrinters discovers Brother QL printers via CUPS API
func (p *DarwinProvider) FindPrinters() ([]PrinterInfo, error) {
	client, err := newCUPSClient()
	if err != nil {
		return nil, err
	}

	// Use CUPS-Get-Printers to discover all printers
	cupsResp, err := client.GetPrinters([]string{"printer-name", "device-uri", "printer-info"})
	if err != nil {
		return nil, fmt.Errorf("CUPS GetPrinters failed: %w", err)
	}

	var printers []PrinterInfo
	for _, attrs := range cupsResp {
		name := ""
		if nameAttrs, ok := attrs["printer-name"]; ok && len(nameAttrs) > 0 {
			if n, ok := nameAttrs[0].Value.(string); ok {
				name = n
			}
		}
		if name == "" {
			continue
		}

		if !isBrotherPrinter(name) {
			// Also check device-uri for Brother vendor
			if uriAttrs, ok := attrs["device-uri"]; ok && len(uriAttrs) > 0 {
				if uri, ok := uriAttrs[0].Value.(string); ok {
					if !isBrotherPrinter(uri) {
						continue
					}
				}
			} else {
				continue
			}
		}

		model := extractModel(name)
		printers = append(printers, PrinterInfo{
			Name:    name,
			Model:   model,
			URI:     "cups://" + name,
			Backend: BackendNative,
		})
	}

	return printers, nil
}

// Connect opens a connection to a printer via IPP
func (p *DarwinProvider) Connect(printer PrinterInfo) (Backend, error) {
	client, err := newCUPSClient()
	if err != nil {
		return nil, err
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
