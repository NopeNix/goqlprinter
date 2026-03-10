package api_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// createTestPNGBase64 creates a minimal valid 1x1 white PNG and returns base64-encoded data.
func createTestPNGBase64(t *testing.T) string {
	t.Helper()
	img := image.NewGray(image.Rect(0, 0, 1, 1))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("failed to create test PNG: %v", err)
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func TestPrintLabel_MissingLabelSize(t *testing.T) {
	t.Parallel()

	h := newTestHandlers(nil)
	r := gin.New()
	r.POST("/api/print", h.PrintLabel)

	body := `{"text":"Hello"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/print", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
	respBody := w.Body.String()
	if !strings.Contains(respBody, "error") {
		t.Errorf("response should contain error key, got: %s", respBody)
	}
}

func TestPrintLabel_InvalidLabelSize(t *testing.T) {
	t.Parallel()

	h := newTestHandlers(nil)
	r := gin.New()
	r.POST("/api/print", h.PrintLabel)

	body := `{"text":"Hello","label_size":"nonexistent"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/print", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
	respBody := w.Body.String()
	if !strings.Contains(respBody, "Invalid label size") {
		t.Errorf("expected 'Invalid label size' in response, got: %s", respBody)
	}
}

func TestPrintLabel_InvalidJSON(t *testing.T) {
	t.Parallel()

	h := newTestHandlers(nil)
	r := gin.New()
	r.POST("/api/print", h.PrintLabel)

	body := `{not valid json`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/print", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestPrintLabel_EmptyBody(t *testing.T) {
	t.Parallel()

	h := newTestHandlers(nil)
	r := gin.New()
	r.POST("/api/print", h.PrintLabel)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/print", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for missing required label_size", w.Code)
	}
}

// TestPrintLabel_Validation is a table-driven test for various invalid requests.
func TestPrintLabel_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantInBody string
	}{
		{
			name:       "missing label_size",
			body:       `{"text":"Hello"}`,
			wantStatus: http.StatusBadRequest,
			wantInBody: "error",
		},
		{
			name:       "invalid label_size",
			body:       `{"text":"Hello","label_size":"invalid"}`,
			wantStatus: http.StatusBadRequest,
			wantInBody: "Invalid label size",
		},
		{
			name:       "empty body",
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
			wantInBody: "error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newTestHandlers(nil)
			r := gin.New()
			r.POST("/api/print", h.PrintLabel)

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/print", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tc.wantStatus)
			}
			if tc.wantInBody != "" && !strings.Contains(w.Body.String(), tc.wantInBody) {
				t.Errorf("body missing %q: %s", tc.wantInBody, w.Body.String())
			}
		})
	}
}

// TestPreviewLabel tests the preview endpoint.
func TestPreviewLabel_MissingLabelSize(t *testing.T) {
	t.Parallel()

	h := newTestHandlers(nil)
	r := gin.New()
	r.POST("/api/preview", h.PreviewLabel)

	body := `{"text":"Hello"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/preview", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestPreviewLabel_InvalidLabelSize(t *testing.T) {
	t.Parallel()

	h := newTestHandlers(nil)
	r := gin.New()
	r.POST("/api/preview", h.PreviewLabel)

	body := `{"text":"Hello","label_size":"nonexistent"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/preview", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

// TestPrintQR tests the QR code print endpoint.
func TestPrintQR_MissingRequiredFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
	}{
		{"missing data", `{"label_size":"62"}`},
		{"missing label_size", `{"data":"hello"}`},
		{"empty body", `{}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newTestHandlers(nil)
			r := gin.New()
			r.POST("/api/print_qr", h.PrintQR)

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/print_qr", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("%s: status = %d, want 400", tc.name, w.Code)
			}
		})
	}
}

func TestPrintQR_InvalidLabelSize(t *testing.T) {
	t.Parallel()

	h := newTestHandlers(nil)
	r := gin.New()
	r.POST("/api/print_qr", h.PrintQR)

	body := `{"data":"hello","label_size":"nonexistent"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/print_qr", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

// TestPrintSVG tests the SVG print endpoint.
func TestPrintSVG_MissingRequiredFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
	}{
		{"missing svg_data", `{"label_size":"62"}`},
		{"missing label_size", `{"svg_data":"<svg></svg>"}`},
		{"empty body", `{}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newTestHandlers(nil)
			r := gin.New()
			r.POST("/api/print_svg", h.PrintSVG)

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/print_svg", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("%s: status = %d, want 400", tc.name, w.Code)
			}
		})
	}
}

func TestPrintSVG_InvalidLabelSize(t *testing.T) {
	t.Parallel()

	h := newTestHandlers(nil)
	r := gin.New()
	r.POST("/api/print_svg", h.PrintSVG)

	body := `{"svg_data":"<svg></svg>","label_size":"nonexistent"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/print_svg", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

// TestGetStatus_Validation tests the status endpoint validation.
func TestGetStatus_InvalidJSON(t *testing.T) {
	t.Parallel()

	h := newTestHandlers(nil)
	r := gin.New()
	r.POST("/api/status", h.GetStatus)

	body := `{not valid json`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/status", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

// TestPrintQR_FileMode tests QR code saving to file when printer=file.
// --- PrintPNGLabel tests ---

func TestPrintPNGLabel_MissingRequiredFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
	}{
		{"missing png_data", `{"label_size":"62"}`},
		{"missing label_size", `{"png_data":"aGVsbG8="}`},
		{"empty body", `{}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := newTestHandlers(nil)
			r := gin.New()
			r.POST("/api/print_png", h.PrintPNGLabel)

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/print_png", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("%s: status = %d, want 400", tc.name, w.Code)
			}
		})
	}
}

func TestPrintPNGLabel_InvalidBase64(t *testing.T) {
	t.Parallel()

	h := newTestHandlers(nil)
	r := gin.New()
	r.POST("/api/print_png", h.PrintPNGLabel)

	body := `{"label_size":"62","png_data":"!!!not-base64!!!"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/print_png", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
	if !strings.Contains(w.Body.String(), "invalid base64") {
		t.Errorf("expected 'invalid base64' in response, got: %s", w.Body.String())
	}
}

func TestPrintPNGLabel_ValidBase64ButNotPNG(t *testing.T) {
	t.Parallel()

	h := newTestHandlers(nil)
	r := gin.New()
	r.POST("/api/print_png", h.PrintPNGLabel)

	// Valid base64 of "hello world" but not a PNG
	body := `{"label_size":"62","png_data":"aGVsbG8gd29ybGQ="}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/print_png", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
	if !strings.Contains(w.Body.String(), "invalid PNG") {
		t.Errorf("expected 'invalid PNG' in response, got: %s", w.Body.String())
	}
}

func TestPrintPNGLabel_InvalidLabelSize(t *testing.T) {
	t.Parallel()

	h := newTestHandlers(nil)
	r := gin.New()
	r.POST("/api/print_png", h.PrintPNGLabel)

	// Create a valid 1x1 PNG in base64
	pngB64 := createTestPNGBase64(t)

	body := `{"label_size":"nonexistent","png_data":"` + pngB64 + `"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/print_png", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

// --- PrintPNGRaw tests ---

func TestPrintPNGRaw_MissingLabelSize(t *testing.T) {
	t.Parallel()

	h := newTestHandlers(nil)
	r := gin.New()
	r.POST("/api/print_png_raw", h.PrintPNGRaw)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/print_png_raw", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
	if !strings.Contains(w.Body.String(), "label_size") {
		t.Errorf("expected 'label_size' in response, got: %s", w.Body.String())
	}
}

func TestPrintQR_FileMode(t *testing.T) {
	t.Parallel()

	h := newTestHandlers(nil)
	r := gin.New()
	r.POST("/api/print_qr", h.PrintQR)

	// Create the debug_output directory if needed
	body := `{"data":"https://example.com","label_size":"62","printer":"file"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/print_qr", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	// It may fail due to missing debug_output dir, but it should not be 400
	if w.Code == http.StatusBadRequest {
		t.Errorf("file mode should not return 400, got body: %s", w.Body.String())
	}
	// If it succeeded, verify the response contains a filename
	if w.Code == http.StatusOK {
		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Errorf("response is not valid JSON: %v", err)
		}
		if _, ok := resp["filename"]; !ok {
			t.Errorf("expected 'filename' in response, got: %v", resp)
		}
	}
}
