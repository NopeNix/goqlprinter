//go:build linux

package brotherql

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

// LinuxBackend implements Backend for Linux /dev/usb/lp* devices
type LinuxBackend struct {
	file        *os.File
	path        string
	readTimeout time.Duration // configurable read timeout, default 3s
}

// Write sends data to the printer device
func (b *LinuxBackend) Write(data []byte) (int, error) {
	if b.file == nil {
		return 0, fmt.Errorf("printer device not open")
	}
	return b.file.Write(data)
}

// Read receives data from the printer device with timeout.
// Uses poll(2) syscall to check for data availability before reading,
// avoiding both indefinite blocking and goroutine leaks.
func (b *LinuxBackend) Read(data []byte) (int, error) {
	if b.file == nil {
		return 0, fmt.Errorf("printer device not open")
	}

	timeout := b.readTimeout
	if timeout == 0 {
		timeout = 3 * time.Second
	}

	fd := int(b.file.Fd())
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		// Use poll(2) to check if data is available, with 100ms poll timeout
		remaining := time.Until(deadline)
		pollTimeoutMs := int(remaining.Milliseconds())
		if pollTimeoutMs > 100 {
			pollTimeoutMs = 100 // Poll in 100ms intervals
		}
		if pollTimeoutMs <= 0 {
			break
		}

		fds := []unix.PollFd{{
			Fd:     int32(fd),
			Events: unix.POLLIN,
		}}

		n, err := unix.Poll(fds, pollTimeoutMs)
		if err != nil {
			if err == unix.EINTR {
				continue // Interrupted by signal, retry
			}
			return 0, fmt.Errorf("poll error: %w", err)
		}

		if n > 0 && fds[0].Revents&unix.POLLIN != 0 {
			// Data is available, safe to read without blocking
			return b.file.Read(data)
		}
		// No data yet, poll again until deadline
	}

	return 0, fmt.Errorf("read timeout: no data from printer within %v", timeout)
}

// Close releases the device file handle
func (b *LinuxBackend) Close() error {
	if b.file != nil {
		err := b.file.Close()
		b.file = nil
		return err
	}
	return nil
}

// LinuxProvider implements BackendProvider for Linux native devices
type LinuxProvider struct{}

// NewLinuxProvider creates a new Linux backend provider
func NewLinuxProvider() *LinuxProvider {
	return &LinuxProvider{}
}

// FindPrinters discovers Brother printers via /dev/usb/lp* and sysfs
func (p *LinuxProvider) FindPrinters() ([]PrinterInfo, error) {
	var printers []PrinterInfo

	// Scan /dev/usb/lp* devices (lp0, lp1, etc.)
	devices, err := filepath.Glob("/dev/usb/lp*")
	if err != nil {
		return nil, fmt.Errorf("failed to scan /dev/usb/lp*: %w", err)
	}

	log.Printf("[Native] Scanning /dev/usb/lp* devices, found %d", len(devices))

	for _, devicePath := range devices {
		// Extract device name (e.g., "lp0" from "/dev/usb/lp0")
		deviceName := filepath.Base(devicePath)
		log.Printf("[Native] Checking device: %s", devicePath)

		// Check sysfs for vendor ID
		// The device symlink points to the USB interface (e.g., 1-1:1.0)
		// idVendor is one level up from the interface, at the USB device level
		deviceSymlink := filepath.Join("/sys/class/usbmisc", deviceName, "device")

		// Resolve the symlink to get the actual interface path
		interfacePath, err := filepath.EvalSymlinks(deviceSymlink)
		if err != nil {
			log.Printf("[Native] Failed to resolve symlink %s: %v", deviceSymlink, err)
			continue
		}
		log.Printf("[Native] Resolved symlink to: %s", interfacePath)

		// Go up one directory to get the USB device path (where idVendor lives)
		sysfsPath := filepath.Dir(interfacePath)
		log.Printf("[Native] USB device sysfs path: %s", sysfsPath)

		// Read vendor ID
		vendorIDPath := filepath.Join(sysfsPath, "idVendor")
		vendorID, err := readSysfsFile(vendorIDPath)
		if err != nil {
			log.Printf("[Native] Failed to read idVendor from %s: %v", vendorIDPath, err)
			continue
		}
		vendorID = strings.TrimSpace(vendorID)
		log.Printf("[Native] Vendor ID: %s", vendorID)

		// Check if it's a Brother device (vendor ID: 04f9)
		if !strings.EqualFold(vendorID, "04f9") {
			log.Printf("[Native] Not a Brother device (vendor ID %s != 04f9)", vendorID)
			continue
		}

		// Read product/model name
		product, err := readSysfsFile(filepath.Join(sysfsPath, "product"))
		if err != nil {
			product = "Unknown Brother Printer"
			log.Printf("[Native] Failed to read product, using default: %s", product)
		}
		product = strings.TrimSpace(product)
		log.Printf("[Native] Product name: %s", product)

		// Check if it's a Brother QL printer
		if !isBrotherPrinter(product) {
			log.Printf("[Native] Not a QL printer: %s", product)
			continue
		}

		// Extract model name
		model := extractModel(product)
		log.Printf("[Native] Found Brother QL printer: %s (model: %s) at %s", product, model, devicePath)

		printers = append(printers, PrinterInfo{
			Name:    product,
			Model:   model,
			URI:     devicePath, // /dev/usb/lpN
			Backend: BackendNative,
		})
	}

	log.Printf("[Native] Total Brother QL printers found: %d", len(printers))
	return printers, nil
}

// Connect opens a connection to the specified printer
func (p *LinuxProvider) Connect(printer PrinterInfo) (Backend, error) {
	if printer.Backend != BackendNative {
		return nil, fmt.Errorf("printer is not a native Linux device")
	}

	// Open the device file for read/write in blocking mode.
	// Blocking mode ensures Write() completes fully (critical for printer protocol).
	// Read() uses poll(2) syscall to avoid indefinite blocking.
	file, err := os.OpenFile(printer.URI, os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w (check permissions - you may need udev rules or sudo)", printer.URI, err)
	}

	return &LinuxBackend{
		file: file,
		path: printer.URI,
	}, nil
}

// SupportsStatus returns true because Linux /dev/usb/lp* devices are bidirectional
func (p *LinuxProvider) SupportsStatus() bool {
	return true
}

// readSysfsFile reads a single-line value from a sysfs file
func readSysfsFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
