//go:build linux

package brotherql

// NewNativeProvider creates a Linux-specific native backend provider
// This function is available on all Linux builds regardless of CGO setting
func NewNativeProvider() BackendProvider {
	return NewLinuxProvider()
}
