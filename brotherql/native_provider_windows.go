//go:build windows

package brotherql

// NewNativeProvider creates a Windows-specific native backend provider
func NewNativeProvider() BackendProvider {
	return NewWindowsProvider()
}
