package storage

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Filesystem keeps private artifacts on the server's persistent volume. Signed
// URLs authenticate the method, key, expiry, filename and exact upload length.
// No directory or unsigned download is exposed. Existing bytes are immutable;
// a repeated identical upload succeeds, a different upload cannot replace it.
type Filesystem struct {
	root, base string
	secret     []byte
	maxSize    int64
}

const FilesystemRoute = "/api/v1/artifacts/object"

func NewFilesystem(root, base string, secret []byte, maxSize int64) (*Filesystem, error) {
	if len(secret) < 32 || root == "" {
		return nil, errors.New("filesystem storage requires a path and a 32-byte signing secret")
	}
	u, err := url.Parse(base)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, errors.New("invalid filesystem storage base URL")
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err = os.MkdirAll(absolute, 0700); err != nil {
		return nil, err
	}
	if maxSize <= 0 {
		maxSize = 500 * 1024 * 1024
	}
	return &Filesystem{absolute, strings.TrimRight(base, "/"), append([]byte(nil), secret...), maxSize}, nil
}
func (s *Filesystem) filename(key string) (string, error) {
	if key == "" || path.Clean(key) != key || strings.HasPrefix(key, "/") || key == "." || strings.HasPrefix(key, "../") || strings.ContainsAny(key, "\\\x00\r\n") {
		return "", errors.New("invalid storage key")
	}
	return filepath.Join(s.root, filepath.FromSlash(key)), nil
}
func (s *Filesystem) signature(method string, q url.Values) string {
	mac := hmac.New(sha256.New, s.secret)
	io.WriteString(mac, "keygate-artifact-v1\n"+method+"\n"+q.Encode())
	return hex.EncodeToString(mac.Sum(nil))
}
func (s *Filesystem) presign(method, key, filename string, size int64, ttl time.Duration) (string, error) {
	if _, err := s.filename(key); err != nil {
		return "", err
	}
	q := url.Values{"key": {key}, "expires": {strconv.FormatInt(time.Now().Add(clampPresignTTL(ttl, 10*time.Minute)).Unix(), 10)}, "filename": {filename}, "size": {strconv.FormatInt(size, 10)}}
	q.Set("signature", s.signature(method, q))
	return s.base + FilesystemRoute + "?" + q.Encode(), nil
}
func (s *Filesystem) PresignedPut(_ context.Context, key, contentType string, expectedSize int64, expires time.Duration) (string, error) {
	if expectedSize <= 0 || expectedSize > s.maxSize {
		return "", fmt.Errorf("upload length must be between 1 and %d bytes", s.maxSize)
	}
	return s.presign("PUT", key, "", expectedSize, expires)
}
func (s *Filesystem) PresignedGet(_ context.Context, key, filename string, expires time.Duration) (string, error) {
	return s.presign("GET", key, filename, 0, expires)
}
func (s *Filesystem) Head(_ context.Context, key string) (*ObjectInfo, error) {
	name, err := s.filename(key)
	if err != nil {
		return nil, err
	}
	st, err := os.Lstat(name)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrObjectNotFound
	}
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() {
		return nil, ErrObjectNotFound
	}
	contentType := mime.TypeByExtension(path.Ext(key))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return &ObjectInfo{Size: st.Size(), ContentType: contentType}, nil
}
func (s *Filesystem) Exists(ctx context.Context, key string) (bool, error) {
	_, err := s.Head(ctx, key)
	if errors.Is(err, ErrObjectNotFound) {
		return false, nil
	}
	return err == nil, err
}
func (s *Filesystem) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	if _, err := s.Head(ctx, key); err != nil {
		return nil, err
	}
	name, _ := s.filename(key)
	return os.Open(name)
}
func (s *Filesystem) Delete(_ context.Context, key string) error {
	name, err := s.filename(key)
	if err != nil {
		return err
	}
	err = os.Remove(name)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
func (s *Filesystem) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	q := r.URL.Query()
	signature := q.Get("signature")
	q.Del("signature")
	for _, key := range []string{"key", "expires", "filename", "size"} {
		if len(q[key]) != 1 {
			http.Error(w, "invalid signed request", 400)
			return
		}
	}
	if len(q) != 4 {
		http.Error(w, "invalid signed request", 400)
		return
	}
	method := r.Method
	if method == "HEAD" {
		method = "GET"
	}
	if method != "GET" && method != "PUT" {
		http.Error(w, "method not allowed", 405)
		return
	}
	expires, err := strconv.ParseInt(q.Get("expires"), 10, 64)
	if err != nil || expires <= time.Now().Unix() || !hmac.Equal([]byte(signature), []byte(s.signature(method, q))) {
		http.Error(w, "signed link is invalid or expired", 403)
		return
	}
	name, err := s.filename(q.Get("key"))
	if err != nil {
		http.Error(w, "invalid storage key", 400)
		return
	}
	if method == "PUT" {
		s.upload(w, r, name, q.Get("size"))
		return
	}
	info, err := s.Head(r.Context(), q.Get("key"))
	if err != nil {
		http.Error(w, "artifact not found", 404)
		return
	}
	f, err := os.Open(name)
	if err != nil {
		http.Error(w, "artifact unavailable", 503)
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", info.ContentType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Length", strconv.FormatInt(info.Size, 10))
	if hint := q.Get("filename"); hint != "" {
		w.Header().Set("Content-Disposition", `attachment; filename="`+sanitizeASCII(hint)+`"; filename*=UTF-8''`+rfc5987Escape(hint))
	}
	if r.Method != "HEAD" {
		_, _ = io.Copy(w, f)
	}
}
func (s *Filesystem) upload(w http.ResponseWriter, r *http.Request, name, sizeText string) {
	size, err := strconv.ParseInt(sizeText, 10, 64)
	if err != nil || size <= 0 || size > s.maxSize {
		http.Error(w, "invalid upload length", 400)
		return
	}
	if r.ContentLength >= 0 && r.ContentLength != size {
		http.Error(w, "upload length mismatch", 400)
		return
	}
	if err = os.MkdirAll(filepath.Dir(name), 0700); err != nil {
		http.Error(w, "storage unavailable", 503)
		return
	}
	temp, err := os.CreateTemp(filepath.Dir(name), ".upload-")
	if err != nil {
		http.Error(w, "storage unavailable", 503)
		return
	}
	defer os.Remove(temp.Name())
	defer temp.Close()
	written, err := io.Copy(temp, http.MaxBytesReader(w, r.Body, size))
	if err != nil || written != size {
		http.Error(w, "upload length mismatch", 400)
		return
	}
	if err = temp.Sync(); err != nil {
		http.Error(w, "storage unavailable", 503)
		return
	}
	if err = temp.Close(); err != nil {
		http.Error(w, "storage unavailable", 503)
		return
	}
	if err = os.Link(temp.Name(), name); err != nil {
		if !errors.Is(err, os.ErrExist) {
			http.Error(w, "storage unavailable", 503)
			return
		}
		incoming, err1 := os.Open(temp.Name())
		existing, err2 := os.Open(name)
		if err1 != nil || err2 != nil {
			if incoming != nil {
				incoming.Close()
			}
			if existing != nil {
				existing.Close()
			}
			http.Error(w, "storage unavailable", 503)
			return
		}
		first, second := sha256.New(), sha256.New()
		_, e1 := io.Copy(first, incoming)
		_, e2 := io.Copy(second, existing)
		incoming.Close()
		existing.Close()
		if e1 != nil || e2 != nil || !hmac.Equal(first.Sum(nil), second.Sum(nil)) {
			http.Error(w, "artifact bytes are immutable; create a new release", 409)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
}
