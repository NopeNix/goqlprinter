package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGetConfig(t *testing.T) {
	t.Parallel()

	h := newTestHandlers(nil)
	r := gin.New()
	r.GET("/api/config", h.GetConfig)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"config"`) {
		t.Errorf("body missing 'config' key: %s", body)
	}
}

func TestGetConfig_IsValidJSON(t *testing.T) {
	t.Parallel()

	h := newTestHandlers(nil)
	r := gin.New()
	r.GET("/api/config", h.GetConfig)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var result map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Errorf("response is not valid JSON: %v, body: %s", err, w.Body.String())
	}
	if _, ok := result["config"]; !ok {
		t.Errorf("parsed JSON missing 'config' key, keys present: %v", keysOf(result))
	}
}

func TestGetConfig_NoDefaultPrinter(t *testing.T) {
	t.Parallel()

	// No printers → default_printer should be null
	h := newTestHandlers(nil)
	r := gin.New()
	r.GET("/api/config", h.GetConfig)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"default_printer":null`) {
		t.Errorf("expected default_printer to be null when no printers available, got: %s", body)
	}
}

// keysOf returns the keys of a map for use in error messages.
func keysOf(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
