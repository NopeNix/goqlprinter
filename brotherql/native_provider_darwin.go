//go:build darwin

package brotherql

// NewNativeProvider creates a macOS-specific native backend provider
func NewNativeProvider() BackendProvider {
	return NewDarwinProvider()
}
