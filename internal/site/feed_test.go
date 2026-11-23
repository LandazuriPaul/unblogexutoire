package site_test

import (
	"encoding/xml"
	"strings"
	"testing"
)

type atomFeed struct {
	Title   string `xml:"title"`
	ID      string `xml:"id"`
	Updated string `xml:"updated"`
	Links   []struct {
		Rel  string `xml:"rel,attr"`
		Href string `xml:"href,attr"`
	} `xml:"link"`
	Entries []struct {
		ID        string   `xml:"id"`
		Title     string   `xml:"title"`
		Published string   `xml:"published"`
		Authors   []string `xml:"author>name"`
		Summary   string   `xml:"summary"`
		Content   struct {
			Type string `xml:"type,attr"`
			Body string `xml:",chardata"`
		} `xml:"content"`
	} `xml:"entry"`
}

func TestFeed(t *testing.T) {
	h := newHandler(t, false)
	rec := do(h, "GET", "/rss/")
	if rec.Code != 200 || rec.Header().Get("Content-Type") != "application/atom+xml; charset=utf-8" {
		t.Fatalf("GET /rss/ = %d %q", rec.Code, rec.Header().Get("Content-Type"))
	}
	var f atomFeed
	if err := xml.Unmarshal(rec.Body.Bytes(), &f); err != nil {
		t.Fatal(err)
	}
	if f.Title != "Un blog exutoire" || f.ID != baseURL+"/" || len(f.Entries) != 7 {
		t.Fatalf("feed = %q %q, %d entries", f.Title, f.ID, len(f.Entries))
	}
	e := f.Entries[0]
	if e.ID != baseURL+"/p7/" || e.Title != "Titre 7" || len(e.Authors) != 1 || e.Authors[0] != "Alice" ||
		e.Published != "2019-06-17T14:00:00+02:00" || f.Updated != e.Published {
		t.Errorf("first entry = %+v (feed updated %s)", e, f.Updated)
	}
	if e.Content.Type != "html" || !strings.Contains(e.Content.Body, "<em>billet</em>") || e.Summary == "" {
		t.Errorf("content = %+v, summary %q", e.Content, e.Summary)
	}
	if !strings.Contains(rec.Body.String(), `href="`+baseURL+`/rss/"`) {
		t.Error("no self link")
	}
	if strings.Contains(rec.Body.String(), "brouillon") {
		t.Error("draft in feed")
	}
	if r := do(h, "GET", "/rss"); r.Code != 301 || r.Header().Get("Location") != "/rss/" {
		t.Errorf("/rss -> %d %q", r.Code, r.Header().Get("Location"))
	}
}

func TestFeedOneAuthorElementPerAuthor(t *testing.T) {
	files := baseFiles()
	files["posts/duo.md"] = "---\ntitle: Duo\nslug: duo\ndate: 2021-01-01\nauthors: [alice, bob]\n---\nÀ deux.\n"
	b := do(handler(t, load(t, files, false)), "GET", "/rss/").Body.String()
	if !strings.Contains(b, "<author>\n      <name>Alice</name>\n    </author>\n    <author>\n      <name>Bob</name>") {
		t.Errorf("authors not in separate <author> elements:\n%s", b[:min(len(b), 900)])
	}
}

func TestSitemap(t *testing.T) {
	rec := do(newHandler(t, false), "GET", "/sitemap.xml")
	var sm struct {
		URLs []struct {
			Loc     string `xml:"loc"`
			LastMod string `xml:"lastmod"`
		} `xml:"url"`
	}
	if err := xml.Unmarshal(rec.Body.Bytes(), &sm); err != nil {
		t.Fatal(err)
	}
	locs := map[string]string{}
	for _, u := range sm.URLs {
		locs[u.Loc] = u.LastMod
	}
	for _, path := range []string{"/", "/articles/", "/a-propos/", "/p3/", "/author/alice/", "/tag/recit/"} {
		if _, ok := locs[baseURL+path]; !ok {
			t.Errorf("sitemap lacks %s", path)
		}
	}
	if locs[baseURL+"/p3/"] != "2019-06-10" {
		t.Errorf("p3 lastmod = %q", locs[baseURL+"/p3/"])
	}
	if _, ok := locs[baseURL+"/brouillon/"]; ok {
		t.Error("draft in sitemap")
	}
	if !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/xml") {
		t.Errorf("Content-Type = %q", rec.Header().Get("Content-Type"))
	}
}

func TestRobots(t *testing.T) {
	rec := do(newHandler(t, false), "GET", "/robots.txt")
	want := "User-agent: *\nAllow: /\nSitemap: https://unblogexutoire.ninja/sitemap.xml\n"
	if rec.Code != 200 || rec.Body.String() != want {
		t.Errorf("robots = %d %q", rec.Code, rec.Body.String())
	}
}
