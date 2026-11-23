package site

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
)

// entry is a response body ready to send, with its gzip version.
type entry struct {
	body, gz    []byte
	etag        string
	contentType string
	cache       string
}

func newEntry(contentType, cache string, body []byte) *entry {
	sum := sha256.Sum256(body)
	return &entry{
		body:        body,
		gz:          gzipBytes(body),
		etag:        `"` + hex.EncodeToString(sum[:])[:16] + `"`,
		contentType: contentType,
		cache:       cache,
	}
}

func gzipBytes(b []byte) []byte {
	var buf bytes.Buffer
	zw, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	_, _ = zw.Write(b)
	_ = zw.Close()
	return buf.Bytes()
}

// Handler serves the site.
func (s *Site) Handler() http.Handler {
	return http.HandlerFunc(s.serveHTTP)
}

func (s *Site) serveHTTP(w http.ResponseWriter, r *http.Request) {
	h := w.Header()
	h.Set("Content-Security-Policy", "default-src 'self'")
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Referrer-Policy", "strict-origin-when-cross-origin")

	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		h.Set("Allow", "GET, HEAD")
		http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	path := r.URL.Path
	switch {
	case path == "/healthz":
		h.Set("Content-Type", "text/plain; charset=utf-8")
		h.Set("Cache-Control", "no-store")
		_, _ = w.Write([]byte("ok"))
	case path == s.cssPath:
		s.serveEntry(w, r, s.css, http.StatusOK)
	case s.routes[path] != nil:
		s.serveEntry(w, r, s.routes[path], http.StatusOK)
	case path == articlesPath:
		s.serveArticles(w, r)
	case !strings.HasSuffix(path, "/") && (s.routes[path+"/"] != nil || path+"/" == articlesPath):
		target := path + "/"
		if r.URL.RawQuery != "" {
			target += "?" + r.URL.RawQuery
		}
		h.Set("Location", target)
		w.WriteHeader(http.StatusMovedPermanently)
	default:
		s.serveEntry(w, r, s.notFound, http.StatusNotFound)
	}
}

// serveEntry writes e, honouring If-None-Match, Accept-Encoding and HEAD.
func (s *Site) serveEntry(w http.ResponseWriter, r *http.Request, e *entry, status int) {
	h := w.Header()
	h.Set("Content-Type", e.contentType)
	h.Set("Cache-Control", e.cache)
	h.Set("Vary", "Accept-Encoding")
	if status == http.StatusOK {
		h.Set("ETag", e.etag)
		if match := r.Header.Get("If-None-Match"); match != "" && strings.Contains(match, e.etag) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}
	writeBody(w, r, status, e.body, e.gz)
}

// writeBody sends body, or gz when the client accepts gzip; HEAD sends headers only.
func writeBody(w http.ResponseWriter, r *http.Request, status int, body, gz []byte) {
	if gz != nil && acceptsGzip(r) {
		w.Header().Set("Content-Encoding", "gzip")
		body = gz
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(status)
	if r.Method != http.MethodHead {
		_, _ = w.Write(body)
	}
}

func acceptsGzip(r *http.Request) bool {
	for _, part := range strings.Split(r.Header.Get("Accept-Encoding"), ",") {
		name, params, _ := strings.Cut(strings.TrimSpace(part), ";")
		if strings.TrimSpace(name) == "gzip" && strings.ReplaceAll(params, " ", "") != "q=0" {
			return true
		}
	}
	return false
}
