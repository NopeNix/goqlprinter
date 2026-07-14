package cmd

import (
	"log/slog"

	"goqlprinter/brotherql"
	icfg "goqlprinter/internal/config"
)

// InitBackendProvider selects and initializes the appropriate backend provider.
//
// Returns an error only for misconfiguration that prevents startup
// (e.g. `backend: "network"` set without `network_uri`). Runtime
// errors (printer unplugged, network unreachable) are deferred to
// the first print/status call so the server can still come up and
// surface the failure through the API.
func InitBackendProvider(cfg *icfg.Config) (brotherql.BackendProvider, error) {
	switch cfg.App.Backend {
	case "usb":
		slog.Info("Using USB backend (gousb/libusb)")
		return initUSBProvider(), nil
	case "native":
		slog.Info("Using native OS backend")
		return brotherql.NewNativeProvider(), nil
	case "network":
		slog.Info("Using network backend (raw TCP, port 9100)")
		if cfg.App.NetworkURI == "" {
			return nil, &BackendError{
				Backend: "network",
				Reason:  "app.network_uri (or LABELPRINTER_APP_NETWORK_URI) is required when app.backend=\"network\"",
			}
		}
		// The model name is the only thing the network backend can't
		// infer from the wire protocol. We require it via
		// app.default_printer so FindPrinters can advertise a
		// resolvable name. If the user leaves it blank, the CLI
		// still works with -m; only the web UI's printer dropdown
		// is empty.
		model := cfg.App.DefaultPrinter
		if model == "" {
			slog.Warn("app.default_printer is empty; web UI will show no printers, " +
				"but the CLI will still work via --printer/-m flags")
		}
		slog.Info("Network printer configured", "uri", cfg.App.NetworkURI, "model", model)
		return brotherql.NewNetworkProvider(cfg.App.NetworkURI, model)
	case "auto":
		return autoDetectProvider(), nil
	default:
		slog.Warn("Unknown backend, falling back to auto mode", "backend", cfg.App.Backend)
		return autoDetectProvider(), nil
	}
}

// BackendError signals a fatal backend misconfiguration. The CLI and
// server convert this into a non-zero exit / 500 response so the user
// sees the cause instead of a silent fallback.
type BackendError struct {
	Backend string
	Reason  string
}

func (e *BackendError) Error() string {
	return "backend " + e.Backend + ": " + e.Reason
}

func autoDetectProvider() brotherql.BackendProvider {
	slog.Info("Auto mode: trying USB backend first")
	usbProvider := initUSBProvider()
	if usbProvider != nil {
		printers, err := usbProvider.FindPrinters()
		if err == nil && len(printers) > 0 {
			slog.Info("USB backend found printers", "count", len(printers))
			return usbProvider
		}
		slog.Info("USB backend found no printers, trying native backend")
	}

	nativeProvider := brotherql.NewNativeProvider()
	printers, err := nativeProvider.FindPrinters()
	if err == nil && len(printers) > 0 {
		slog.Info("Native backend found printers", "count", len(printers))
		return nativeProvider
	}

	// Auto mode intentionally does NOT fall back to the network
	// backend: there is no discovery path for network printers, and
	// silently using whatever happens to be in `app.network_uri` is
	// surprising. Users with a network-only printer should set
	// `app.backend: "network"` explicitly.
	slog.Info("No printers found with USB or native backend; auto mode has no network fallback. " +
		"If your printer is network-attached, set app.backend=\"network\" and app.network_uri=...")
	return nativeProvider
}
