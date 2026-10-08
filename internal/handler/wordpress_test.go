package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestWordPressRequestValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, body := range []string{
		`{}`, `{"license_key":"KEY"}`, `{"site_url":"https://example.test"}`,
		`{"license_key":"` + strings.Repeat("a", 17*1024) + `","site_url":"https://example.test"}`,
		`not-json`,
	} {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/wordpress/test/verify", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		if _, ok := bindWordPressRequest(c); ok {
			t.Errorf("invalid body accepted (length %d)", len(body))
		}
		if w.Code != http.StatusBadRequest || w.Header().Get("Cache-Control") != "private, no-store" {
			t.Errorf("validation failure status/cache policy: %d %v", w.Code, w.Header())
		}
	}
}
