// Package brotherql implements the Brother QL label printer protocol and hardware communication.
package brotherql

// Backend abstracts printer communication
type Backend interface {
	Write(data []byte) (int, error)
	Read(data []byte) (int, error) // ESC i S response (USB/Linux native)
	Close() error
}

// StatusProvider for backends that support status queries
type StatusProvider interface {
	// GetStatus returns printer status (Windows/macOS use OS APIs)
	GetStatus() (PrinterStatus, error)
}

// PrinterStatus contains status information.
// JSON field names match what the frontend expects (LabelStatus interface).
type PrinterStatus struct {
	Ready bool   `json:"ready"`
	Busy  bool   `json:"busy"`
	Error string `json:"-"` // internal use only, mapped to Errors slice for frontend

	// Fields expected by frontend LabelStatus interface:
	ModelName   string   `json:"model_name"`
	MediaType   string   `json:"media_type"`
	MediaWidth  int      `json:"media_width"`
	MediaLength int      `json:"media_length"`
	StatusType  string   `json:"status_type"`
	PhaseType   string   `json:"phase_type"`
	Errors      []string `json:"errors"`
}

// BackendType defines available backend implementations
type BackendType string

const (
	BackendUSB    BackendType = "usb"    // gousb/libusb (CGO)
	BackendNative BackendType = "native" // OS native (Pure Go)
)

// PrinterInfo contains discovered printer information
type PrinterInfo struct {
	Name    string      // Display name
	Model   string      // "QL-570", "QL-800", etc.
	URI     string      // Connection identifier
	Backend BackendType // Which backend found it
}

// BackendProvider creates backend connections
type BackendProvider interface {
	// FindPrinters discovers available Brother printers
	FindPrinters() ([]PrinterInfo, error)

	// Connect opens a connection to a printer
	Connect(printer PrinterInfo) (Backend, error)

	// SupportsStatus returns true if status queries work
	SupportsStatus() bool
}
