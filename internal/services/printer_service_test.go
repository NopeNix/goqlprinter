package services_test

import (
	"errors"
	"io"
	"testing"

	"goqlprinter/brotherql"
	"goqlprinter/internal/services"
)

// mockProvider implements brotherql.BackendProvider for testing.
type mockProvider struct {
	printers []brotherql.PrinterInfo
	err      error
}

func (m *mockProvider) FindPrinters() ([]brotherql.PrinterInfo, error) {
	return m.printers, m.err
}

func (m *mockProvider) Connect(printer brotherql.PrinterInfo) (brotherql.Backend, error) {
	return &mockBackend{}, nil
}

func (m *mockProvider) SupportsStatus() bool {
	return false
}

// mockBackend implements brotherql.Backend for testing.
type mockBackend struct{}

func (m *mockBackend) Write(p []byte) (int, error) { return len(p), nil }
func (m *mockBackend) Read(p []byte) (int, error)  { return 0, io.EOF }
func (m *mockBackend) Close() error                { return nil }

func TestNewPrinterService(t *testing.T) {
	t.Parallel()

	provider := &mockProvider{}
	svc := services.NewPrinterService(provider)

	if svc == nil {
		t.Fatal("expected non-nil PrinterService")
	}
}

func TestFindPrinters_WithPrinters(t *testing.T) {
	t.Parallel()

	provider := &mockProvider{
		printers: []brotherql.PrinterInfo{
			{Name: "QL-700 (USB)", Model: "QL-700", URI: "usb:001:001", Backend: brotherql.BackendUSB},
			{Name: "QL-800 (USB)", Model: "QL-800", URI: "usb:001:002", Backend: brotherql.BackendUSB},
		},
	}
	svc := services.NewPrinterService(provider)

	found, err := svc.FindPrinters()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(found) != 2 {
		t.Fatalf("expected 2 printers, got %d", len(found))
	}

	cases := []struct {
		idx           int
		expectedModel string
		expectedUID   string
	}{
		{0, "QL-700", "usb:001:001"},
		{1, "QL-800", "usb:001:002"},
	}
	for _, tc := range cases {
		p := found[tc.idx]
		if p.Model != tc.expectedModel {
			t.Errorf("printer[%d].Model = %q, want %q", tc.idx, p.Model, tc.expectedModel)
		}
		if p.UID != tc.expectedUID {
			t.Errorf("printer[%d].UID = %q, want %q", tc.idx, p.UID, tc.expectedUID)
		}
	}
}

func TestFindPrinters_ProviderError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("USB bus failure")
	provider := &mockProvider{err: wantErr}
	svc := services.NewPrinterService(provider)

	_, err := svc.FindPrinters()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFindPrinters_Empty(t *testing.T) {
	t.Parallel()

	provider := &mockProvider{printers: []brotherql.PrinterInfo{}}
	svc := services.NewPrinterService(provider)

	found, err := svc.FindPrinters()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(found) != 0 {
		t.Fatalf("expected 0 printers, got %d", len(found))
	}
}

func TestResolvePrinter_EmptyID_NoDefault(t *testing.T) {
	t.Parallel()

	provider := &mockProvider{}
	svc := services.NewPrinterService(provider)

	_, err := svc.ResolvePrinter("")
	if err == nil {
		t.Fatal("expected error for empty identifier with no default, got nil")
	}
}

func TestResolvePrinter_EmptyID_WithDefault(t *testing.T) {
	t.Parallel()

	provider := &mockProvider{
		printers: []brotherql.PrinterInfo{
			{Name: "QL-700 (USB)", Model: "QL-700", URI: "usb:001:001", Backend: brotherql.BackendUSB},
		},
	}
	svc := services.NewPrinterService(provider)
	svc.InitializeDefaultPrinter("")

	got, err := svc.ResolvePrinter("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Model != "QL-700" {
		t.Errorf("expected default model QL-700, got %q", got.Model)
	}
}

func TestResolvePrinter_ByModel(t *testing.T) {
	t.Parallel()

	provider := &mockProvider{
		printers: []brotherql.PrinterInfo{
			{Name: "QL-800 (USB)", Model: "QL-800", URI: "usb:001:002", Backend: brotherql.BackendUSB},
		},
	}
	svc := services.NewPrinterService(provider)

	got, err := svc.ResolvePrinter("QL-800")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Model != "QL-800" {
		t.Errorf("expected model QL-800, got %q", got.Model)
	}
}

func TestResolvePrinter_NotFound(t *testing.T) {
	t.Parallel()

	provider := &mockProvider{
		printers: []brotherql.PrinterInfo{
			{Name: "QL-800 (USB)", Model: "QL-800", URI: "usb:001:002", Backend: brotherql.BackendUSB},
		},
	}
	svc := services.NewPrinterService(provider)

	_, err := svc.ResolvePrinter("QL-NOTEXIST")
	if err == nil {
		t.Fatal("expected error for unknown model, got nil")
	}
}

func TestResolvePrinter_ByURI(t *testing.T) {
	t.Parallel()

	provider := &mockProvider{
		printers: []brotherql.PrinterInfo{
			{Name: "QL-800 (USB)", Model: "QL-800", URI: "usb:001:002", Backend: brotherql.BackendUSB},
		},
	}
	svc := services.NewPrinterService(provider)

	got, err := svc.ResolvePrinter("usb:001:002")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.UID != "usb:001:002" {
		t.Errorf("expected UID usb:001:002, got %q", got.UID)
	}
}

func TestInitializeDefaultPrinter_SetsFirst(t *testing.T) {
	t.Parallel()

	provider := &mockProvider{
		printers: []brotherql.PrinterInfo{
			{Name: "QL-700 (USB)", Model: "QL-700", URI: "usb:001:001", Backend: brotherql.BackendUSB},
			{Name: "QL-800 (USB)", Model: "QL-800", URI: "usb:001:002", Backend: brotherql.BackendUSB},
		},
	}
	svc := services.NewPrinterService(provider)
	svc.InitializeDefaultPrinter("")

	def := svc.GetDefaultPrinter()
	if def == nil {
		t.Fatal("expected a default printer to be set")
	}
	if def.Model != "QL-700" {
		t.Errorf("expected first printer QL-700 as default, got %q", def.Model)
	}
}

func TestInitializeDefaultPrinter_SetsConfigured(t *testing.T) {
	t.Parallel()

	provider := &mockProvider{
		printers: []brotherql.PrinterInfo{
			{Name: "QL-800 (USB)", Model: "QL-800", URI: "usb:001:002", Backend: brotherql.BackendUSB},
			{Name: "QL-700 (USB)", Model: "QL-700", URI: "usb:001:001", Backend: brotherql.BackendUSB},
		},
	}
	svc := services.NewPrinterService(provider)
	svc.InitializeDefaultPrinter("QL-700")

	def := svc.GetDefaultPrinter()
	if def == nil {
		t.Fatal("expected a default printer to be set")
	}
	if def.Model != "QL-700" {
		t.Errorf("expected configured printer QL-700, got %q", def.Model)
	}
}

func TestInitializeDefaultPrinter_FallsBackToFirst(t *testing.T) {
	t.Parallel()

	provider := &mockProvider{
		printers: []brotherql.PrinterInfo{
			{Name: "QL-800 (USB)", Model: "QL-800", URI: "usb:001:002", Backend: brotherql.BackendUSB},
		},
	}
	svc := services.NewPrinterService(provider)
	svc.InitializeDefaultPrinter("QL-MISSING")

	def := svc.GetDefaultPrinter()
	if def == nil {
		t.Fatal("expected a default printer to be set")
	}
	if def.Model != "QL-800" {
		t.Errorf("expected fallback to first printer QL-800, got %q", def.Model)
	}
}
