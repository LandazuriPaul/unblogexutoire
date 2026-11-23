package site_test

import (
	"strings"
	"testing"
)

func body(t *testing.T, drafts bool, path string) string {
	t.Helper()
	rec := do(newHandler(t, drafts), "GET", path)
	if rec.Code != 200 {
		t.Fatalf("GET %s = %d", path, rec.Code)
	}
	return rec.Body.String()
}

func TestPostPage(t *testing.T) {
	b := body(t, false, "/p3/")
	for _, want := range []string{
		"<h1>Titre 3</h1>", "10 juin 2019", `href="/author/alice/"`, ">Alice<",
		`href="/tag/recit/"`, ">Récit<", "Corps du <em>billet</em> 3.",
		`property="og:type" content="article"`, `<title>Titre 3 · Un blog exutoire</title>`,
		`property="og:url" content="https://unblogexutoire.ninja/p3/"`,
	} {
		if !strings.Contains(b, want) {
			t.Errorf("/p3/ lacks %s", want)
		}
	}
}

func TestPrevNext(t *testing.T) {
	middle := body(t, false, "/p3/")
	if !strings.Contains(middle, `href="/p2/"`) || !strings.Contains(middle, "Article précédent") ||
		!strings.Contains(middle, `href="/p4/"`) || !strings.Contains(middle, "Article suivant") {
		t.Error("/p3/ lacks prev (p2) or next (p4) link")
	}
	if newest := body(t, false, "/p7/"); strings.Contains(newest, "Article suivant") {
		t.Error("newest post has a next link")
	}
	if oldest := body(t, false, "/p1/"); strings.Contains(oldest, "Article précédent") {
		t.Error("oldest post has a previous link")
	}
}

func TestPagePage(t *testing.T) {
	b := body(t, false, "/a-propos/")
	if !strings.Contains(b, "<h1>À propos</h1>") || !strings.Contains(b, "Nous écrivons.") {
		t.Errorf("/a-propos/ = %s", b)
	}
}

func TestAuthorPage(t *testing.T) {
	b := body(t, false, "/author/alice/")
	if !strings.Contains(b, "Articles de Alice") || !strings.Contains(b, "Écrit la nuit.") {
		t.Error("author page lacks title or bio")
	}
	for _, n := range []string{"1", "2", "3", "4", "5", "6", "7"} {
		if !strings.Contains(b, `href="/p`+n+`/"`) {
			t.Errorf("author page lacks p%s", n)
		}
	}
}

func TestTagPage(t *testing.T) {
	b := body(t, false, "/tag/politique/")
	if !strings.Contains(b, "Articles étiquetés « Politique »") {
		t.Error("tag page lacks its title")
	}
	for n, want := range map[string]bool{"1": false, "2": true, "3": false, "4": true, "6": true} {
		if got := strings.Contains(b, `href="/p`+n+`/"`); got != want {
			t.Errorf("tag page lists p%s = %t, want %t", n, got, want)
		}
	}
}

func TestDraftsNotServed(t *testing.T) {
	h := newHandler(t, false)
	for _, path := range []string{"/brouillon/", "/tag/secret/", "/author/bob/", "/cachee/"} {
		if rec := do(h, "GET", path); rec.Code != 404 {
			t.Errorf("GET %s = %d, want 404", path, rec.Code)
		}
	}
	// p2 (06-02) and p3 (06-10) surround the draft (06-05): prev/next must skip it.
	if b := body(t, false, "/p2/"); strings.Contains(b, "/brouillon/") || !strings.Contains(b, `href="/p3/"`) {
		t.Error("/p2/ links the draft or misses p3")
	}
	for _, path := range []string{"/", "/p3/", "/author/alice/", "/tag/recit/"} {
		if strings.Contains(body(t, false, path), "brouillon") {
			t.Errorf("%s mentions the draft", path)
		}
	}
}

func TestDraftsServed(t *testing.T) {
	h := newHandler(t, true)
	for _, path := range []string{"/brouillon/", "/tag/secret/", "/author/bob/", "/cachee/"} {
		if rec := do(h, "GET", path); rec.Code != 200 {
			t.Errorf("drafts on: GET %s = %d, want 200", path, rec.Code)
		}
	}
}
