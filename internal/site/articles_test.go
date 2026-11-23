package site_test

import (
	"compress/gzip"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

// manyHandler: the base fixture plus 16 extra posts, all by alice, so alice has 23 posts.
func manyHandler(t *testing.T) http.Handler {
	files := baseFiles()
	for i := 0; i < 16; i++ {
		files[fmt.Sprintf("posts/x%02d.md", i)] = fmt.Sprintf(
			"---\ntitle: Extra %d\nslug: x%02d\ndate: 2020-03-%02dT12:00:00Z\nauthors: [alice]\n---\nTexte %d.\n", i, i, i+1, i)
	}
	return handler(t, load(t, files, false))
}

func get(t *testing.T, h http.Handler, target string) string {
	t.Helper()
	rec := do(h, "GET", target)
	if rec.Code != 200 {
		t.Fatalf("GET %s = %d", target, rec.Code)
	}
	return rec.Body.String()
}

var resultLink = regexp.MustCompile(`<h2><a href="/[a-z0-9-]+/">`)

func TestArticlesDefault(t *testing.T) {
	b := get(t, manyHandler(t), "/articles/")
	if n := len(resultLink.FindAllString(b, -1)); n != 10 {
		t.Errorf("%d results, want 10", n)
	}
	if !strings.Contains(b, "23 articles") || !strings.Contains(b, `href="/articles/?page=2"`) {
		t.Error("missing count or next link")
	}
	if strings.Contains(b, "Plus récents") {
		t.Error("first page has a previous link")
	}
}

func TestArticlesFilterKeepsParams(t *testing.T) {
	b := get(t, manyHandler(t), "/articles/?auteur=alice&page=2")
	if !strings.Contains(b, `href="/articles/?auteur=alice&amp;page=3"`) ||
		!strings.Contains(b, `href="/articles/?auteur=alice&amp;page=1"`) {
		t.Error("prev/next links drop the filters")
	}
	if !strings.Contains(b, "Plus récents") || !strings.Contains(b, "Plus anciens") {
		t.Error("missing prev/next labels")
	}
}

func TestArticlesNoResults(t *testing.T) {
	b := get(t, newHandler(t, false), "/articles/?q=zzzz")
	if !strings.Contains(b, "Aucun article ne correspond à ces critères.") || !strings.Contains(b, "0 article<") {
		t.Error("missing empty-state text or singular count")
	}
	if one := get(t, newHandler(t, false), "/articles/?q=titre+3"); !strings.Contains(one, "1 article<") {
		t.Error("count for one result is not singular")
	}
}

func TestArticlesHostileInput(t *testing.T) {
	h := newHandler(t, false)
	for _, q := range []string{"page=-3", "page=abc", "annee=99999", "mois=13", "auteur=nope",
		"q=" + strings.Repeat("a", 10000), "page=999999999999999999999"} {
		if rec := do(h, "GET", "/articles/?"+q); rec.Code != 200 {
			t.Errorf("?%.40s = %d", q, rec.Code)
		}
	}
}

func TestArticlesEscapesQuery(t *testing.T) {
	b := get(t, newHandler(t, false), "/articles/?q="+url.QueryEscape("<script>alert(1)</script>"))
	if strings.Contains(b, "<script>alert") || !strings.Contains(b, "&lt;script&gt;") {
		t.Error("query not escaped")
	}
}

func TestArticlesSelected(t *testing.T) {
	b := get(t, newHandler(t, false), "/articles/?annee=2019&auteur=alice&tag=recit&mois=6&q=corps")
	for _, want := range []string{
		`<option value="2019" selected>`, `<option value="alice" selected>`,
		`<option value="recit" selected>`, `<option value="6" selected>juin`, `value="corps"`,
		`href="/articles/">Réinitialiser`,
	} {
		if !strings.Contains(b, want) {
			t.Errorf("form lacks %s", want)
		}
	}
	if strings.Contains(get(t, newHandler(t, false), "/articles/"), "Réinitialiser") {
		t.Error("reset link shown without filters")
	}
}

func TestArticlesDraftsHidden(t *testing.T) {
	b := get(t, newHandler(t, false), "/articles/?q=brouillon")
	if strings.Contains(b, "/brouillon/") || strings.Contains(b, `value="secret"`) || strings.Contains(b, `value="bob"`) {
		t.Error("draft, draft-only tag or draft-only author visible")
	}
	if drafts := get(t, newHandler(t, true), "/articles/?q=brouillon"); !strings.Contains(drafts, "/brouillon/") {
		t.Error("drafts on: draft not found")
	}
}

func TestArticlesCacheAndGzip(t *testing.T) {
	h := newHandler(t, false)
	rec := do(h, "GET", "/articles/", "Accept-Encoding", "gzip")
	if rec.Header().Get("Cache-Control") != "public, max-age=60" || rec.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("headers = %v", rec.Header())
	}
	if _, err := gzip.NewReader(rec.Body); err != nil {
		t.Error(err)
	}
	if r := do(h, "GET", "/articles?q=x"); r.Code != 301 || r.Header().Get("Location") != "/articles/?q=x" {
		t.Errorf("/articles -> %d %q", r.Code, r.Header().Get("Location"))
	}
	if r := do(h, "HEAD", "/articles/"); r.Code != 200 || r.Body.Len() != 0 {
		t.Errorf("HEAD = %d, %d bytes", r.Code, r.Body.Len())
	}
}
