package api_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGetLabelSizes(t *testing.T) {
	t.Parallel()

	h := newTestHandlers(nil)
	r := gin.New()
	r.GET("/api/label-sizes", h.GetLabelSizes)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/label-sizes", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"label_sizes"`) {
		t.Errorf("body missing 'label_sizes' key: %s", body)
	}
	// Confirm the list is non-empty (there are many predefined labels)
	if strings.Contains(body, `"label_sizes":[]`) || strings.Contains(body, `"label_sizes":null`) {
		t.Errorf("label_sizes should be non-empty, got: %s", body)
	}
}

func TestGetLabelSize_Known(t *testing.T) {
	t.Parallel()

	h := newTestHandlers(nil)
	r := gin.New()
	r.GET("/api/label-sizes/:id", h.GetLabelSize)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/label-sizes/62", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"62"`) {
		t.Errorf("body missing id '62': %s", body)
	}
}

func TestGetLabelSize_Unknown(t *testing.T) {
	t.Parallel()

	h := newTestHandlers(nil)
	r := gin.New()
	r.GET("/api/label-sizes/:id", h.GetLabelSize)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/label-sizes/notexist", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

// TestGetLabelSize_KnownIDs is a table-driven test for various known label IDs.
func TestGetLabelSize_KnownIDs(t *testing.T) {
	t.Parallel()

	knownIDs := []string{"12", "29", "62", "17x54", "d12", "d58", "102x152"}
	for _, id := range knownIDs {
		t.Run(id, func(t *testing.T) {
			t.Parallel()
			h := newTestHandlers(nil)
			r := gin.New()
			r.GET("/api/label-sizes/:id", h.GetLabelSize)

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/label-sizes/"+id, nil)
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("id=%s: status = %d, want 200", id, w.Code)
			}
		})
	}
}
