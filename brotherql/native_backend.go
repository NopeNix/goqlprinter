package brotherql

import (
	"errors"
	"strings"
)

var (
	// ErrStatusNotSupported indicates the backend doesn't support status queries
	ErrStatusNotSupported = errors.New("status queries not supported by this backend")
)

// isBrotherPrinter checks if a printer name indicates a Brother QL printer
// Note: At this point, vendor ID (04f9) has already been verified as Brother,
// so we only need to check for "ql" in the product name.
// USB sysfs 'product' file typically contains just "QL-570", not "Brother QL-570"
func isBrotherPrinter(name string) bool {
	nameLower := strings.ToLower(name)
	return strings.Contains(nameLower, "ql")
}

// extractModel attempts to extract the model name from a printer string
// E.g., "Brother QL-800" -> "QL-800"
func extractModel(name string) string {
	// Look for "QL-" pattern
	idx := strings.Index(strings.ToUpper(name), "QL-")
	if idx == -1 {
		return "Unknown"
	}

	// Extract "QL-XXX" or "QL-XXXXX"
	remainder := name[idx:]
	parts := strings.Fields(remainder)
	if len(parts) > 0 {
		return parts[0]
	}

	return "Unknown"
}
