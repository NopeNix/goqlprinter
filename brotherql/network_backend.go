package brotherql

import (
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"time"
)

// Default network port for Brother "raw" / JetDirect-style printing.
// Every network-capable Brother QL model (QL-710W, QL-720NW, QL-810W,
// QL-820NWB, QL-1100, QL-1110NWB, QL-1115NWB) exposes this port by default.
const defaultNetworkPort = 9100

// NetworkBackend talks to a Brother QL printer over a raw TCP socket,
// typically on port 9100. The wire protocol is identical to the USB
// native backend — the Brother raster stream written to the socket
// produces the same physical output as the same bytes written to
// /dev/usb/lp*.
//
// What's NOT supported:
//   - Bidirectional status (Read always returns ErrStatusNotSupported;
//     Brother's `ESC i S` query needs the USB bulk-IN endpoint, which
//     isn't exposed over the network socket).
//   - mDNS/Bonjour discovery (FindPrinters returns empty).
//
// Both of these can be layered on later without changing this struct.
type NetworkBackend struct {
	conn net.Conn
	addr string
}

// NewNetworkBackend dials a network printer by URI and returns a ready
// backend. Accepted URI forms:
//
//	tcp://192.168.1.21:9100
//	tcp://192.168.1.21            (port defaults to 9100)
//	network://192.168.1.21:9100   (alias for tcp://)
//	192.168.1.21:9100             (bare host:port)
//
// The connection uses a 5-second TCP dial timeout. A 30-second write
// timeout is set on the socket so a half-open connection (e.g. printer
// powered off mid-job) surfaces as an error instead of hanging the
// caller forever.
func NewNetworkBackend(uri string) (*NetworkBackend, error) {
	addr, err := parseNetworkURI(uri)
	if err != nil {
		return nil, fmt.Errorf("invalid network printer URI %q: %w", uri, err)
	}

	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("network printer %s unreachable: %w", addr, err)
	}

	// Belt + suspenders. Brother printers will close the socket if you
	// violate the protocol; we want to time out before that if the
	// kernel buffer is full.
	if err := conn.SetWriteDeadline(time.Now().Add(30 * time.Second)); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to set write deadline on %s: %w", addr, err)
	}

	slog.Info("Network printer connected", "addr", addr)
	return &NetworkBackend{conn: conn, addr: addr}, nil
}

// Write pushes raster bytes to the printer over the network socket.
func (b *NetworkBackend) Write(data []byte) (int, error) {
	if b.conn == nil {
		return 0, fmt.Errorf("network printer not connected")
	}
	return b.conn.Write(data)
}

// Read is not supported on the network backend — port 9100 is a
// write-only socket on Brother network QL models. The Brother status
// query (`ESC i S`) requires the USB bulk-IN endpoint. Callers should
// check SupportsStatus() and skip status polling for network printers.
func (b *NetworkBackend) Read(data []byte) (int, error) {
	return 0, ErrStatusNotSupported
}

// Close shuts down the TCP connection.
func (b *NetworkBackend) Close() error {
	if b.conn == nil {
		return nil
	}
	err := b.conn.Close()
	b.conn = nil
	return err
}

// parseNetworkURI normalizes the four accepted forms into a single
// `host:port` string for net.Dial. Also tolerates bare IPv6 with
// brackets (tcp://[::1]:9100) and without (tcp://::1) where
// unambiguous.
func parseNetworkURI(uri string) (string, error) {
	if uri == "" {
		return "", fmt.Errorf("URI is empty")
	}

	host := ""

	// Strip scheme. Use HasPrefix (not length + slice) so URIs that
	// are *just* the scheme with no host ("tcp://") are correctly
	// detected as empty.
	switch {
	case strings.HasPrefix(uri, "tcp://"):
		host = uri[6:]
	case strings.HasPrefix(uri, "network://"):
		host = uri[10:]
	default:
		host = uri
	}

	if host == "" {
		return "", fmt.Errorf("URI %q is missing a host after the scheme", uri)
	}

	// IPv6 without brackets (e.g. "tcp://::1" or "tcp://fe80::1:9100")
	// is ambiguous because multiple colons trip up net.SplitHostPort.
	// Disambiguate by trying to parse the part before the last colon
	// as an IPv6 address: if it parses, the part after is the port;
	// otherwise the whole thing is the address (use default port).
	if strings.Count(host, ":") > 1 {
		// already bracketed: tcp://[::1]:9100 or tcp://[::1]
		if strings.HasPrefix(host, "[") {
			if !strings.Contains(host, "]") {
				return "", fmt.Errorf("malformed IPv6 URI %q: missing ']'", uri)
			}
			return host, nil
		}
		// Unbracketed: try to extract a trailing port.
		if i := strings.LastIndex(host, ":"); i > 0 {
			candidate := host[:i]
			portPart := host[i+1:]
			if net.ParseIP(candidate) != nil {
				// candidate is a valid IPv6 address; portPart is the port
				return "[" + candidate + "]:" + portPart, nil
			}
		}
		// No port or no valid IPv6 host found — wrap the whole thing.
		return "[" + host + "]:" + strconv.Itoa(defaultNetworkPort), nil
	}

	// Split host:port (handles "host:port" and plain "host")
	h, p, err := net.SplitHostPort(host)
	if err != nil {
		// No port given — treat the whole thing as host
		return fmt.Sprintf("%s:%d", host, defaultNetworkPort), nil
	}
	return fmt.Sprintf("%s:%s", h, p), nil
}
