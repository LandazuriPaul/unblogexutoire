package site_test

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/LandazuriPaul/unblogexutoire/internal/site"
)

func TestFormatDate(t *testing.T) {
	for in, want := range map[string]string{
		"2019-06-10T01:36:15Z": "10 juin 2019",
		"2019-12-31T23:30:00Z": "1er janvier 2020",
		"2020-08-27T10:00:00Z": "27 août 2020",
	} {
		d, _ := time.Parse(time.RFC3339, in)
		if got := site.FormatDate(d); got != want {
			t.Errorf("FormatDate(%s) = %q, want %q", in, got, want)
		}
	}
}

func TestHomeShowsAllPosts(t *testing.T) {
	body := do(newHandler(t, false), "GET", "/").Body.String()
	var order []int
	for _, n := range []string{"7", "6", "5", "4", "3", "2", "1"} {
		order = append(order, strings.Index(body, "Titre "+n+"<"))
	}
	for i, idx := range order {
		if idx < 0 || (i > 0 && idx < order[i-1]) {
			t.Errorf("post %d missing or out of order (indexes %v)", 7-i, order)
		}
	}
	if !strings.Contains(body, `href="/articles/"`) {
		t.Error("no link to /articles/")
	}
	if !strings.Contains(body, `href="/a-propos/"`) {
		t.Error("no link to the about page")
	}
}

func TestSecurityHeaders(t *testing.T) {
	h := newHandler(t, false)
	for _, rec := range []*http.Response{
		do(h, "GET", "/").Result(), do(h, "GET", "/nope/").Result(),
		do(h, "GET", "/healthz").Result(), do(h, "POST", "/").Result(), do(h, "GET", "/a-propos").Result(),
	} {
		for k, v := range map[string]string{
			"Content-Security-Policy": "default-src 'self'",
			"X-Content-Type-Options":  "nosniff",
			"Referrer-Policy":         "strict-origin-when-cross-origin",
		} {
			if got := rec.Header.Get(k); got != v {
				t.Errorf("%s %s: %s = %q, want %q", rec.Request.Method, rec.Request.URL, k, got, v)
			}
		}
	}
}

func TestNotFound(t *testing.T) {
	rec := do(newHandler(t, false), "GET", "/nope/")
	if rec.Code != 404 || !strings.Contains(rec.Body.String(), "Page introuvable") {
		t.Errorf("got %d %q", rec.Code, rec.Body.String())
	}
	if rec := do(newHandler(t, false), "GET", "//evil.example"); rec.Code != 404 {
		t.Errorf("//evil.example -> %d, want 404 (no open redirect)", rec.Code)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	rec := do(newHandler(t, false), "POST", "/")
	if rec.Code != 405 || rec.Header().Get("Allow") != "GET, HEAD" {
		t.Errorf("got %d Allow=%q", rec.Code, rec.Header().Get("Allow"))
	}
}

func TestTrailingSlashRedirect(t *testing.T) {
	rec := do(newHandler(t, false), "GET", "/a-propos?x=1")
	if rec.Code != 301 || rec.Header().Get("Location") != "/a-propos/?x=1" {
		t.Errorf("got %d Location=%q", rec.Code, rec.Header().Get("Location"))
	}
}

func TestETag304(t *testing.T) {
	h := newHandler(t, false)
	first := do(h, "GET", "/")
	etag := first.Header().Get("ETag")
	if etag == "" || first.Header().Get("Cache-Control") != "public, max-age=300" {
		t.Fatalf("ETag=%q Cache-Control=%q", etag, first.Header().Get("Cache-Control"))
	}
	second := do(h, "GET", "/", "If-None-Match", etag)
	if second.Code != 304 || second.Body.Len() != 0 {
		t.Errorf("got %d with %d bytes", second.Code, second.Body.Len())
	}
}

func TestGzip(t *testing.T) {
	h := newHandler(t, false)
	plain := do(h, "GET", "/").Body.Bytes()
	rec := do(h, "GET", "/", "Accept-Encoding", "gzip, br")
	if rec.Header().Get("Content-Encoding") != "gzip" || rec.Header().Get("Vary") != "Accept-Encoding" {
		t.Fatalf("headers = %v", rec.Header())
	}
	zr, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(zr)
	if !bytes.Equal(got, plain) {
		t.Error("gunzipped body differs from plain body")
	}
}

func TestHead(t *testing.T) {
	h := newHandler(t, false)
	get, head := do(h, "GET", "/"), do(h, "HEAD", "/")
	if head.Code != 200 || head.Body.Len() != 0 || head.Header().Get("ETag") != get.Header().Get("ETag") {
		t.Errorf("HEAD = %d, %d bytes, ETag %q vs %q", head.Code, head.Body.Len(), head.Header().Get("ETag"), get.Header().Get("ETag"))
	}
}

func TestCSSHashed(t *testing.T) {
	h := newHandler(t, false)
	css := regexp.MustCompile(`/static/style\.[0-9a-f]{10}\.css`).FindString(do(h, "GET", "/").Body.String())
	if css == "" {
		t.Fatal("no hashed stylesheet link")
	}
	rec := do(h, "GET", css)
	if rec.Code != 200 || !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/css") ||
		rec.Header().Get("Cache-Control") != "public, max-age=31536000, immutable" {
		t.Errorf("css = %d %v", rec.Code, rec.Header())
	}
}

func TestOpenGraph(t *testing.T) {
	body := do(newHandler(t, false), "GET", "/").Body.String()
	for _, want := range []string{
		`<html lang="fr">`, `property="og:title"`, `property="og:url" content="https://unblogexutoire.ninja/"`,
		`property="og:type" content="website"`, `name="description"`, `<title>Un blog exutoire</title>`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("home lacks %s", want)
		}
	}
}

func TestHealthz(t *testing.T) {
	rec := do(newHandler(t, false), "GET", "/healthz")
	if rec.Code != 200 || rec.Body.String() != "ok" {
		t.Errorf("got %d %q", rec.Code, rec.Body.String())
	}
}
