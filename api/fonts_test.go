package api_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGetFonts_Empty(t *testing.T) {
	t.Parallel()

	// newTestHandlers uses empty FontDirs, so no fonts will be discovered.
	h := newTestHandlers(nil)
	r := gin.New()
	r.GET("/api/fonts", h.GetFonts)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/fonts", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"fonts"`) {
		t.Errorf("body missing 'fonts' key: %s", body)
	}
	// fonts list must be present; when no font dirs are configured the handler
	// returns a nil slice which JSON-encodes as null.
	if !strings.Contains(body, `"fonts":[]`) && !strings.Contains(body, `"fonts":null`) {
		t.Errorf("expected fonts key with empty/null value, got: %s", body)
	}
}
