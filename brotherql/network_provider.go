package brotherql

import "log/slog"

// NetworkProvider is a BackendProvider that connects to a single
// network-attached Brother QL printer by URI.
//
// In the typical deployment the URI comes from the config file
// (`app.network_uri` / `LABELPRINTER_APP_NETWORK_URI`) and the
// provider is instantiated once at server start. The web UI and CLI
// then refer to the printer by model name (e.g. "QL-810W"), and the
// provider hands out fresh TCP connections per print job.
//
// If you need to talk to multiple network printers, run multiple
// goqlprinter instances with different `app.network_uri` values and
// front them with a reverse proxy — or, in a future revision, this
// provider can be taught to round-robin across a list of URIs.
type NetworkProvider struct {
	uri   string
	model string
}

// NewNetworkProvider creates a provider bound to a single network
// printer URI. The URI is validated here so misconfiguration fails
// at startup, not on the first print job.
//
// `model` is the model name (e.g. "QL-810W") that FindPrinters will
// advertise. It MUST be set; the network backend has no way to
// auto-detect the model from the wire protocol. If `model` is empty
// FindPrinters returns no entries and the printer won't be usable
// from the web UI (though the CLI will still work via -m flag).
func NewNetworkProvider(uri, model string) (*NetworkProvider, error) {
	if _, err := parseNetworkURI(uri); err != nil {
		return nil, err
	}
	return &NetworkProvider{uri: uri, model: model}, nil
}

// FindPrinters returns a single synthetic printer entry so the
// PrinterService can resolve the model name from API requests. This
// is the network equivalent of a discovery scan — the printer isn't
// "found" via broadcast, it's "known" because the user configured
// its address.
//
// If no model was provided to the constructor, returns an empty list
// (the user can still use the CLI with -p tcp://... -m MODEL).
func (p *NetworkProvider) FindPrinters() ([]PrinterInfo, error) {
	if p.model == "" {
		return nil, nil
	}
	return []PrinterInfo{{
		Name:    p.model + " (network)",
		Model:   p.model,
		URI:     p.uri,
		Backend: BackendNetwork,
	}}, nil
}

// Connect opens a fresh TCP connection to the configured printer.
func (p *NetworkProvider) Connect(_ PrinterInfo) (Backend, error) {
	return NewNetworkBackend(p.uri)
}

// SupportsStatus returns false. Port 9100 is write-only; status
// queries (`ESC i S`) require the USB bulk-IN endpoint.
func (p *NetworkProvider) SupportsStatus() bool {
	return false
}

// URI returns the configured network address. Useful for logging and
// for surfacing it in the /api/config endpoint.
func (p *NetworkProvider) URI() string {
	return p.uri
}

// ensure interface compliance at compile time
var _ BackendProvider = (*NetworkProvider)(nil)

// keep the slog import live in case it's needed for debug logs later;
// (currently we delegate all logging to NewNetworkBackend / Connect).
var _ = slog.Default
