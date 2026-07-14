package brotherql

import (
	"strings"
	"testing"
)

func TestParseNetworkURI(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		// happy paths
		{"tcp full", "tcp://192.168.1.21:9100", "192.168.1.21:9100", false},
		{"tcp no port", "tcp://192.168.1.21", "192.168.1.21:9100", false},
		{"network scheme", "network://10.0.0.5:9100", "10.0.0.5:9100", false},
		{"network no port", "network://10.0.0.5", "10.0.0.5:9100", false},
		{"bare host:port", "192.168.1.21:9100", "192.168.1.21:9100", false},
		{"bare host", "192.168.1.21", "192.168.1.21:9100", false},
		{"ipv6 tcp", "tcp://[fe80::1]:9100", "[fe80::1]:9100", false},
		{"ipv6 no port", "tcp://fe80::1", "[fe80::1]:9100", false},
		{"localhost", "tcp://127.0.0.1:9100", "127.0.0.1:9100", false},
		{"hostname", "tcp://ql-printer.local:9100", "ql-printer.local:9100", false},

		// error paths
		{"empty", "", "", true},
		{"tcp empty host", "tcp://", "", true},
		{"network empty host", "network://", "", true},
		{"garbage", "not a uri at all", "not a uri at all:9100", false}, // passes through
		{"just port", ":9100", ":9100", false},                         // passes through (weird but harmless)
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseNetworkURI(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (result=%q)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestNetworkBackendErrStatusNotSupported(t *testing.T) {
	b := &NetworkBackend{}
	buf := make([]byte, 16)
	n, err := b.Read(buf)
	if n != 0 {
		t.Errorf("Read should return 0 bytes, got %d", n)
	}
	if err == nil {
		t.Fatal("Read should return an error")
	}
	if !strings.Contains(err.Error(), "status") {
		t.Errorf("error should mention status, got: %v", err)
	}
}

func TestNetworkBackendCloseIdempotent(t *testing.T) {
	b := &NetworkBackend{} // no conn — Close should be a no-op
	if err := b.Close(); err != nil {
		t.Errorf("Close on unconnected backend should not error, got: %v", err)
	}
}

func TestNetworkBackendWriteNotConnected(t *testing.T) {
	b := &NetworkBackend{}
	n, err := b.Write([]byte("hello"))
	if n != 0 {
		t.Errorf("Write on unconnected backend should return 0 bytes, got %d", n)
	}
	if err == nil {
		t.Fatal("Write on unconnected backend should error")
	}
}
