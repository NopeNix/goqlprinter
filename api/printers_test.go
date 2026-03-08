package api_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"goqlprinter/brotherql"
)

func TestGetPrinters_Empty(t *testing.T) {
	t.Parallel()

	h := newTestHandlers(nil)
	r := gin.New()
	r.GET("/api/printers", h.GetPrinters)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/printers", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"printers"`) {
		t.Errorf("body missing 'printers' key: %s", body)
	}
	if !strings.Contains(body, `[]`) {
		t.Errorf("body should contain empty array, got: %s", body)
	}
}

func TestGetPrinters_WithPrinters(t *testing.T) {
	t.Parallel()

	printers := []brotherql.PrinterInfo{
		{Name: "QL-700 (USB)", Model: "QL-700", URI: "usb:001:005", Backend: brotherql.BackendUSB},
		{Name: "QL-800 (USB)", Model: "QL-800", URI: "usb:001:006", Backend: brotherql.BackendUSB},
	}
	h := newTestHandlers(printers)
	r := gin.New()
	r.GET("/api/printers", h.GetPrinters)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/printers", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "QL-700") {
		t.Errorf("body missing QL-700: %s", body)
	}
	if !strings.Contains(body, "QL-800") {
		t.Errorf("body missing QL-800: %s", body)
	}
}

func TestGetPrinters_ProviderError(t *testing.T) {
	t.Parallel()

	h := newTestHandlersWithError(errors.New("USB bus error"))
	r := gin.New()
	r.GET("/api/printers", h.GetPrinters)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/printers", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "error") {
		t.Errorf("body missing 'error' key: %s", body)
	}
}
