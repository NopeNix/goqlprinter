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

// SetReadTimeout sets the timeout used by Read. Implements ReadTimeoutSetter.
func (b *LinuxBackend) SetReadTimeout(d time.Duration) {
	b.readTimeout = d
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
		remaining := time.Until(deadline)
		pollTimeoutMs := int(remaining.Milliseconds())
		if pollTimeoutMs > 100 {
			pollTimeoutMs = 100
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
				continue // interrupted by signal, retry
			}
			return 0, fmt.Errorf("poll error: %w", err)
		}

		if n > 0 && fds[0].Revents&unix.POLLIN != 0 {
			return b.file.Read(data)
		}
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

// FindPrinters discovers Brother printers via /dev/usb/lp* and sysfs.
func (p *LinuxProvider) FindPrinters() ([]PrinterInfo, error) {
	var printers []PrinterInfo

	devices, err := filepath.Glob("/dev/usb/lp*")
	if err != nil {
		return nil, fmt.Errorf("failed to scan /dev/usb/lp*: %w", err)
	}

	log.Printf("[Native] Scanning /dev/usb/lp* devices, found %d", len(devices))

	for _, devicePath := range devices {
		deviceName := filepath.Base(devicePath)
		log.Printf("[Native] Checking device: %s", devicePath)

		// /sys/class/usbmisc/<dev>/device points to the USB interface (e.g. 1-1:1.0).
		// idVendor lives one directory up at the USB device level.
		deviceSymlink := filepath.Join("/sys", "class", "usbmisc", deviceName, "device") //nolint:gocritic

		interfacePath, err := filepath.EvalSymlinks(deviceSymlink)
		if err != nil {
			log.Printf("[Native] Failed to resolve symlink %s: %v", deviceSymlink, err)
			continue
		}
		log.Printf("[Native] Resolved symlink to: %s", interfacePath)

		sysfsPath := filepath.Dir(interfacePath)
		log.Printf("[Native] USB device sysfs path: %s", sysfsPath)

		vendorIDPath := filepath.Join(sysfsPath, "idVendor")
		vendorID, err := readSysfsFile(vendorIDPath)
		if err != nil {
			log.Printf("[Native] Failed to read idVendor from %s: %v", vendorIDPath, err)
			continue
		}
		vendorID = strings.TrimSpace(vendorID)
		log.Printf("[Native] Vendor ID: %s", vendorID)

		if !strings.EqualFold(vendorID, "04f9") {
			log.Printf("[Native] Not a Brother device (vendor ID %s != 04f9)", vendorID)
			continue
		}

		product, err := readSysfsFile(filepath.Join(sysfsPath, "product"))
		if err != nil {
			product = "Unknown Brother Printer"
			log.Printf("[Native] Failed to read product, using default: %s", product)
		}
		product = strings.TrimSpace(product)
		log.Printf("[Native] Product name: %s", product)

		if !isBrotherPrinter(product) {
			log.Printf("[Native] Not a QL printer: %s", product)
			continue
		}

		model := extractModel(product)
		log.Printf("[Native] Found Brother QL printer: %s (model: %s) at %s", product, model, devicePath)

		printers = append(printers, PrinterInfo{
			Name:    product,
			Model:   model,
			URI:     devicePath,
			Backend: BackendNative,
		})
	}

	log.Printf("[Native] Total Brother QL printers found: %d", len(printers))
	return printers, nil
}

// Connect opens a connection to a Linux native printer device.
// The file is opened in blocking mode so Write completes fully;
// Read uses poll(2) to avoid indefinite blocking.
func (p *LinuxProvider) Connect(printer PrinterInfo) (Backend, error) {
	if printer.Backend != BackendNative {
		return nil, fmt.Errorf("printer is not a native Linux device")
	}

	file, err := os.OpenFile(printer.URI, os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w (check permissions - you may need udev rules or sudo)", printer.URI, err)
	}

	return &LinuxBackend{
		file: file,
		path: printer.URI,
	}, nil
}

// SupportsStatus returns true because Linux /dev/usb/lp* devices are bidirectional.
func (p *LinuxProvider) SupportsStatus() bool {
	return true
}

// readSysfsFile reads a single-line value from a sysfs file.
func readSysfsFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
