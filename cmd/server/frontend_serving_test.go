package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

// The SPA shell answers routes, not files. A tab that still holds an
// older index.html asks for the asset that build named; if the server
// answers with HTML, the browser refuses a script it was told is
// JavaScript and the page comes up blank with nothing in the log.
// And the shell itself is re-read, so a frontend rebuilt under a
// running server stops pointing at assets it has deleted.
func TestServeFrontend(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()
	dist := filepath.Join(dir, "web", "dist")
	if err := os.MkdirAll(filepath.Join(dist, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dist, filepath.FromSlash(name)), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("index.html", `<script src="/assets/app-v1.js"></script>`)
	write("assets/app-v1.js", "console.log(1)")
	if err := os.MkdirAll(filepath.Join(dist, "downloads"), 0o755); err != nil {
		t.Fatal(err)
	}
	write("downloads/free-1.0.0.zip", "PK-test-installer")

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	r := gin.New()
	r.GET("/api/v1/health", func(c *gin.Context) { c.String(http.StatusOK, "api") })
	serveFrontend(r)

	get := func(path string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		return w
	}

	if w := get("/assets/app-v1.js"); w.Code != http.StatusOK || w.Body.String() != "console.log(1)" {
		t.Fatalf("asset: %d %q", w.Code, w.Body.String())
	} else if cc := w.Header().Get("Cache-Control"); cc != "public, max-age=31536000, immutable" {
		t.Fatalf("hashed asset should be cacheable forever, got %q", cc)
	}
	// The app's own routes get the shell, and it must not be cached.
	w := get("/licenses/abc")
	if w.Code != http.StatusOK || w.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("route: %d %q", w.Code, w.Header().Get("Content-Type"))
	}
	if cc := w.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Fatalf("the shell must be revalidated, got %q", cc)
	}
	if got := w.Header().Get("X-Robots-Tag"); got != "noindex, nofollow" {
		t.Fatalf("purchase/account pages should stay out of search results, got %q", got)
	}
	// Previously shared product links and the plugin's View plans URL
	// return to the appropriate part of the only public purchase page.
	for path, target := range map[string]string{
		"/pricing":                       "/#store-plans",
		"/pricing/":                      "/#store-plans",
		"/guide":                         "/#installation",
		"/products/accessible-forms":     "/",
		"/products/accessible-forms-pro": "/#comparison",
	} {
		for _, method := range []string{http.MethodGet, http.MethodHead} {
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(method, path, nil))
			if w.Code != http.StatusFound || w.Header().Get("Location") != target {
				t.Fatalf("legacy link %s %s: status=%d location=%q", method, path, w.Code, w.Header().Get("Location"))
			}
		}
	}
	// A route that happens to end in something dot-like is still a
	// route, not a file.
	if w := get("/licenses/1.0"); w.Code != http.StatusOK || w.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("versioned route: %d %q", w.Code, w.Header().Get("Content-Type"))
	}
	// The asset a previous build named is gone: say so.
	if w := get("/assets/app-v0.js"); w.Code != http.StatusNotFound {
		t.Fatalf("stale asset: %d, want 404", w.Code)
	}
	if w := get("/favicon.ico"); w.Code != http.StatusNotFound {
		t.Fatalf("missing file: %d, want 404", w.Code)
	}
	if w := get("/downloads/free-1.0.0.zip"); w.Code != http.StatusOK || w.Body.String() != "PK-test-installer" || w.Header().Get("Content-Disposition") != `attachment; filename="free-1.0.0.zip"` {
		t.Fatal("Free installer was not served as a download")
	}
	if w := get("/downloads/missing.zip"); w.Code != http.StatusNotFound {
		t.Fatal("missing installer received the SPA shell")
	}
	// API routes are never touched by the fallback.
	if w := get("/api/v1/health"); w.Code != http.StatusOK || w.Body.String() != "api" {
		t.Fatalf("api passthrough: %d %q", w.Code, w.Body.String())
	} else if w.Header().Get("X-Robots-Tag") != "" {
		t.Fatal("frontend indexing headers reached the API")
	}
	// A rebuild lands while the server runs: the next page load has
	// to name the new asset, not the deleted one.
	write("index.html", `<script src="/assets/app-v2.js"></script>`)
	if err := os.Remove(filepath.Join(dist, "assets", "app-v1.js")); err != nil {
		t.Fatal(err)
	}
	write("assets/app-v2.js", "console.log(2)")
	if w := get("/"); w.Body.String() != `<script src="/assets/app-v2.js"></script>` {
		t.Fatalf("shell not re-read after a rebuild: %q", w.Body.String())
	}
}
