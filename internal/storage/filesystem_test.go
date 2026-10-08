package storage

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestFilesystemSignedLifecycle(t *testing.T) {
	var fs *Filesystem
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fs.ServeHTTP(w, r) }))
	defer server.Close()
	var err error
	fs, err = NewFilesystem(t.TempDir(), server.URL, bytes.Repeat([]byte{1}, 32), 1024)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	content := []byte("private-plugin-zip")
	put, err := fs.PresignedPut(ctx, "product/1.0.0/plugin.zip", "application/zip", int64(len(content)), time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	send := func(method, address string, body []byte) int {
		t.Helper()
		request, _ := http.NewRequest(method, address, bytes.NewReader(body))
		response, err := server.Client().Do(request)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		io.Copy(io.Discard, response.Body)
		return response.StatusCode
	}
	if status := send("PUT", put, content); status != 200 {
		t.Fatalf("upload status %d", status)
	}
	if status := send("PUT", put, content); status != 200 {
		t.Fatalf("repeat upload status %d", status)
	}
	changed := bytes.Repeat([]byte{'x'}, len(content))
	if status := send("PUT", put, changed); status != 409 {
		t.Fatalf("replacement status %d", status)
	}
	get, _ := fs.PresignedGet(ctx, "product/1.0.0/plugin.zip", "Accessible Pro.zip", time.Minute)
	response, err := server.Client().Get(get)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(response.Body)
	response.Body.Close()
	if !bytes.Equal(data, content) || response.Header.Get("Cache-Control") != "private, no-store" || !strings.Contains(response.Header.Get("Content-Disposition"), "Accessible Pro.zip") {
		t.Fatalf("incorrect download response: %s", data)
	}
	if status := send("HEAD", get, nil); status != 200 {
		t.Fatalf("head status %d", status)
	}
	if status := send("GET", put, nil); status != 403 {
		t.Fatalf("upload token used for read: %d", status)
	}
	u, _ := url.Parse(get)
	q := u.Query()
	q.Set("key", "product/other.zip")
	u.RawQuery = q.Encode()
	if status := send("GET", u.String(), nil); status != 403 {
		t.Fatalf("changed key status %d", status)
	}
	q = u.Query()
	q.Set("expires", "1")
	q.Del("signature")
	q.Set("signature", fs.signature("GET", q))
	u.RawQuery = q.Encode()
	if status := send("GET", u.String(), nil); status != 403 {
		t.Fatalf("expired status %d", status)
	}
	if status := send("GET", server.URL+FilesystemRoute+"?key=product/1.0.0/plugin.zip", nil); status != 400 {
		t.Fatalf("unsigned download status %d", status)
	}
	if _, err = fs.PresignedGet(ctx, "../escape", "", time.Minute); err == nil {
		t.Fatal("traversal accepted")
	}
	if _, err = fs.PresignedPut(ctx, "oversized.zip", "", 1025, time.Minute); err == nil {
		t.Fatal("oversize accepted")
	}
	if err = fs.Delete(ctx, "product/1.0.0/plugin.zip"); err != nil {
		t.Fatal(err)
	}
	if exists, err := fs.Exists(ctx, "product/1.0.0/plugin.zip"); err != nil || exists {
		t.Fatalf("delete failed %v", err)
	}
}
func TestFilesystemRejectsPartialUpload(t *testing.T) {
	fs, err := NewFilesystem(t.TempDir(), "https://example.test", bytes.Repeat([]byte{2}, 32), 100)
	if err != nil {
		t.Fatal(err)
	}
	signed, _ := fs.PresignedPut(context.Background(), "partial.zip", "", 10, time.Minute)
	request := httptest.NewRequest("PUT", signed, strings.NewReader("short"))
	response := httptest.NewRecorder()
	fs.ServeHTTP(response, request)
	if response.Code != 400 {
		t.Fatalf("partial status %d", response.Code)
	}
	if exists, _ := fs.Exists(context.Background(), "partial.zip"); exists {
		t.Fatal("partial upload became visible")
	}
}
