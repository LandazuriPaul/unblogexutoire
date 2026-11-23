package site_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"testing/fstest"

	"github.com/LandazuriPaul/unblogexutoire/internal/content"
	"github.com/LandazuriPaul/unblogexutoire/internal/site"
)

const baseURL = "https://unblogexutoire.ninja"

// baseFiles: 7 published posts p1 (oldest) .. p7 (newest) by alice, tagged "Récit" (odd)
// or "Politique" (even); p3 is dated 2019-06-10T01:36:15Z. Plus a draft by bob with the
// draft-only tag "Secret", the a-propos page and a draft page.
func baseFiles() map[string]string {
	files := map[string]string{
		"authors.yaml": "- slug: alice\n  name: Alice\n  bio: Écrit la nuit.\n- slug: bob\n  name: Bob\n",
		"posts/draft.md": "---\ntitle: Brouillon secret\nslug: brouillon\ndate: 2019-06-05T12:00:00Z\nauthors: [bob]\n" +
			"tags: [Secret]\nstatus: draft\n---\nPas fini.\n",
		"pages/a-propos.md": "---\ntitle: À propos\nslug: a-propos\ndescription: Qui sommes-nous.\n---\nNous écrivons.\n",
		"pages/cache.md":    "---\ntitle: Page cachée\nslug: cachee\nstatus: draft\n---\nChut.\n",
	}
	for i := 1; i <= 7; i++ {
		date := fmt.Sprintf("2019-06-0%dT12:00:00Z", i)
		if i == 3 {
			date = "2019-06-10T01:36:15Z"
		}
		if i > 3 {
			date = fmt.Sprintf("2019-06-1%dT12:00:00Z", i)
		}
		tag := "Récit"
		if i%2 == 0 {
			tag = "Politique"
		}
		files[fmt.Sprintf("posts/p%d.md", i)] = fmt.Sprintf(
			"---\ntitle: Titre %d\nslug: p%d\ndate: %s\nauthors: [alice]\ntags: [%s]\n---\nCorps du *billet* %d.\n",
			i, i, date, tag, i)
	}
	return files
}

func load(t *testing.T, files map[string]string, drafts bool) *content.Site {
	t.Helper()
	m := fstest.MapFS{}
	for name, data := range files {
		m[name] = &fstest.MapFile{Data: []byte(data)}
	}
	c, err := content.Load(m, content.LoadOptions{IncludeDrafts: drafts})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func handler(t *testing.T, c *content.Site) http.Handler {
	t.Helper()
	s, err := site.New(site.Config{Content: c, Assets: os.DirFS("../.."), BaseURL: baseURL})
	if err != nil {
		t.Fatal(err)
	}
	return s.Handler()
}

func newHandler(t *testing.T, drafts bool) http.Handler {
	return handler(t, load(t, baseFiles(), drafts))
}

func do(h http.Handler, method, target string, headers ...string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, nil)
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}
