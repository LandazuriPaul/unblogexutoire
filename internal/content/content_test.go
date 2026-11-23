package content_test

import (
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/LandazuriPaul/unblogexutoire/internal/content"
)

const authorsYAML = `- slug: alice
  name: Alice
  bio: Écrit parfois.
- slug: bob
  name: Bob
`

// fixture builds a content FS; authors.yaml is added unless overridden.
func fixture(files map[string]string) fs.FS {
	m := fstest.MapFS{"authors.yaml": {Data: []byte(authorsYAML)}}
	for name, data := range files {
		m[name] = &fstest.MapFile{Data: []byte(data)}
	}
	return m
}

func post(slug, date, extra, body string) string {
	return "---\ntitle: Titre " + slug + "\nslug: " + slug + "\ndate: " + date +
		"\nauthors:\n  - alice\n" + extra + "---\n" + body + "\n"
}

func mustLoad(t *testing.T, fsys fs.FS, opts content.LoadOptions) *content.Site {
	t.Helper()
	site, err := content.Load(fsys, opts)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return site
}

func TestLoadValidSite(t *testing.T) {
	site := mustLoad(t, fixture(map[string]string{
		"posts/a.md":     post("a", "2019-06-10T01:36:15Z", "tags:\n  - Récit\n", "Premier *texte*."),
		"posts/b.md":     post("b", "2019-12-31T23:30:00Z", "", "Réveillon."),
		"posts/c.md":     post("c", "2019-03-01", "", "Le plus ancien."),
		"pages/about.md": "---\ntitle: À propos\nslug: a-propos\n---\nNous.\n",
	}), content.LoadOptions{})

	var slugs []string
	for _, p := range site.Posts {
		slugs = append(slugs, p.Slug)
		if p.Date.Location() != content.Paris {
			t.Errorf("post %s: location %v, want Paris", p.Slug, p.Date.Location())
		}
	}
	if got := strings.Join(slugs, ","); got != "b,a,c" {
		t.Errorf("post order = %s, want b,a,c", got)
	}
	if len(site.Years) != 2 || site.Years[0] != 2020 || site.Years[1] != 2019 {
		t.Errorf("Years = %v, want [2020 2019]", site.Years)
	}
	if site.Pages["a-propos"] == nil || !strings.Contains(string(site.Pages["a-propos"].HTML), "Nous.") {
		t.Errorf("page a-propos missing or empty: %+v", site.Pages["a-propos"])
	}
	a := site.Posts[1]
	if !strings.Contains(string(a.HTML), "<em>texte</em>") {
		t.Errorf("HTML = %q", a.HTML)
	}
	if len(a.Authors) != 1 || a.Authors[0].Name != "Alice" || a.Authors[0].Bio != "Écrit parfois." {
		t.Errorf("Authors = %+v", a.Authors)
	}
	if len(a.Tags) != 1 || a.Tags[0] != (content.Tag{Slug: "recit", Name: "Récit"}) {
		t.Errorf("Tags = %+v", a.Tags)
	}
	if !strings.Contains(a.SearchText, "premier texte") {
		t.Errorf("SearchText = %q", a.SearchText)
	}
	if len(site.Authors) != 1 || site.Authors[0].Slug != "alice" {
		t.Errorf("site Authors = %+v, want only alice (bob has no post)", site.Authors)
	}
	if c := site.Posts[2]; c.Date.Hour() != 0 || c.Date.Day() != 1 {
		t.Errorf("date-only post = %v, want midnight Paris", c.Date)
	}
}

func TestDescriptionFallback(t *testing.T) {
	long := strings.Repeat("mot ", 40)
	site := mustLoad(t, fixture(map[string]string{
		"posts/a.md": post("a", "2020-01-01", "", long),
		"posts/b.md": post("b", "2020-01-02", "description: Résumé à moi.\n", long),
	}), content.LoadOptions{})
	b, a := site.Posts[0], site.Posts[1]
	if want := strings.TrimSpace(strings.Repeat("mot ", 30)) + "…"; a.Description != want {
		t.Errorf("fallback = %q, want %q", a.Description, want)
	}
	if b.Description != "Résumé à moi." {
		t.Errorf("explicit = %q", b.Description)
	}
	if !strings.Contains(b.SearchText, "resume a moi") {
		t.Errorf("SearchText lacks description: %q", b.SearchText)
	}
}

func TestRawHTMLStripped(t *testing.T) {
	site := mustLoad(t, fixture(map[string]string{
		"posts/a.md": post("a", "2020-01-01", "", "<script>x</script>\n\nok"),
	}), content.LoadOptions{})
	if h := string(site.Posts[0].HTML); strings.Contains(h, "<script") || !strings.Contains(h, "ok") {
		t.Errorf("HTML = %q", h)
	}
}

func draftFixture() fs.FS {
	return fixture(map[string]string{
		"posts/a.md":     post("a", "2020-01-01", "tags:\n  - public\n", "Publié."),
		"posts/d.md":     "---\ntitle: Brouillon\nslug: d\ndate: 2021-05-01\nauthors:\n  - bob\ntags:\n  - secret\nstatus: draft\n---\nPas fini.\n",
		"pages/p.md":     "---\ntitle: Page brouillon\nslug: p\nstatus: draft\n---\nx\n",
		"pages/about.md": "---\ntitle: À propos\nslug: a-propos\n---\nNous.\n",
	})
}

func TestDraftsExcluded(t *testing.T) {
	site := mustLoad(t, draftFixture(), content.LoadOptions{})
	if len(site.Posts) != 1 || site.Posts[0].Slug != "a" {
		t.Errorf("Posts = %d, want only a", len(site.Posts))
	}
	if site.Pages["p"] != nil {
		t.Error("draft page loaded")
	}
	for _, tag := range site.Tags {
		if tag.Slug == "secret" {
			t.Error("draft-only tag listed")
		}
	}
	for _, a := range site.Authors {
		if a.Slug == "bob" {
			t.Error("draft-only author listed")
		}
	}
	if len(site.Years) != 1 || site.Years[0] != 2020 {
		t.Errorf("Years = %v, want [2020]", site.Years)
	}
}

func TestDraftsIncluded(t *testing.T) {
	site := mustLoad(t, draftFixture(), content.LoadOptions{IncludeDrafts: true})
	if len(site.Posts) != 2 || site.Posts[0].Slug != "d" || !site.Posts[0].Draft {
		t.Errorf("Posts[0] = %+v, want draft d", site.Posts[0])
	}
	if p := site.Pages["p"]; p == nil || !p.Draft {
		t.Error("draft page missing")
	}
	if len(site.Tags) != 2 || len(site.Authors) != 2 {
		t.Errorf("Tags=%v Authors=%d", site.Tags, len(site.Authors))
	}
}

func TestCRLFAndBOM(t *testing.T) {
	src := "\ufeff---\r\ntitle: X\r\nslug: x\r\ndate: 2020-01-01\r\nauthors:\r\n  - alice\r\n---\r\nCorps\r\n"
	site := mustLoad(t, fixture(map[string]string{"posts/x.md": src}), content.LoadOptions{})
	if site.Posts[0].Title != "X" || !strings.Contains(string(site.Posts[0].HTML), "Corps") {
		t.Errorf("post = %+v", site.Posts[0])
	}
}

func TestValidationErrors(t *testing.T) {
	ok := post("ok", "2020-01-01", "", "x")
	cases := []struct {
		name    string
		files   map[string]string
		file    string
		keyword string
	}{
		{"missing title", map[string]string{"posts/m.md": "---\nslug: m\ndate: 2020-01-01\nauthors: [alice]\n---\nx"}, "posts/m.md", "title"},
		{"bad slug", map[string]string{"posts/m.md": post("Foo Bar", "2020-01-01", "", "x")}, "posts/m.md", "slug"},
		{"duplicate slug", map[string]string{"posts/ok.md": ok, "pages/ok.md": "---\ntitle: P\nslug: ok\n---\nx"}, "ok.md", "duplicate"},
		{"reserved slug", map[string]string{"posts/m.md": post("rss", "2020-01-01", "", "x")}, "posts/m.md", "reserved"},
		{"bad date", map[string]string{"posts/m.md": post("m", "10/06/2019", "", "x")}, "posts/m.md", "date"},
		{"missing date", map[string]string{"posts/m.md": "---\ntitle: T\nslug: m\nauthors: [alice]\n---\nx"}, "posts/m.md", "date"},
		{"missing authors", map[string]string{"posts/m.md": "---\ntitle: T\nslug: m\ndate: 2020-01-01\n---\nx"}, "posts/m.md", "authors"},
		{"unknown author", map[string]string{"posts/m.md": strings.Replace(post("m", "2020-01-01", "", "x"), "alice", "carol", 1)}, "posts/m.md", "carol"},
		{"unknown key", map[string]string{"posts/m.md": post("m", "2020-01-01", "auteurs: [alice]\n", "x")}, "posts/m.md", "auteurs"},
		{"no closing fence", map[string]string{"posts/m.md": "---\ntitle: T\nslug: m\n"}, "posts/m.md", "frontmatter"},
		{"no frontmatter", map[string]string{"posts/m.md": "Juste du texte."}, "posts/m.md", "frontmatter"},
		{"bad status", map[string]string{"posts/m.md": post("m", "2020-01-01", "status: publié\n", "x")}, "posts/m.md", "status"},
		{"tag name clash", map[string]string{
			"posts/m.md": post("m", "2020-01-01", "tags: [Récit]\n", "x"),
			"posts/n.md": post("n", "2020-01-02", "tags: [Recit]\n", "x"),
		}, ".md", "recit"},
		{"empty tag", map[string]string{"posts/m.md": post("m", "2020-01-01", "tags: ['!!']\n", "x")}, "posts/m.md", "tag"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := content.Load(fixture(tc.files), content.LoadOptions{IncludeDrafts: true})
			if err == nil {
				t.Fatal("Load succeeded, want error")
			}
			msg := err.Error()
			if !strings.Contains(msg, tc.file) || !strings.Contains(msg, tc.keyword) {
				t.Errorf("error %q lacks %q or %q", msg, tc.file, tc.keyword)
			}
		})
	}
}

func TestMisplacedFilesRejected(t *testing.T) {
	for _, name := range []string{"posts/m.markdown", "posts/m.MD", "posts/2026/m.md", "pages/sub/p.md", "posts/m.txt"} {
		t.Run(name, func(t *testing.T) {
			_, err := content.Load(fixture(map[string]string{name: post("m", "2020-01-01", "", "x")}), content.LoadOptions{})
			if err == nil || !strings.Contains(err.Error(), name) {
				t.Errorf("err = %v, want an error naming %s", err, name)
			}
		})
	}
	// Hidden files (.DS_Store, editor swap files) are never embedded, so they're ignored.
	if _, err := content.Load(fixture(map[string]string{"posts/.DS_Store": "x"}), content.LoadOptions{}); err != nil {
		t.Errorf("hidden file rejected: %v", err)
	}
}

func TestAuthorsFileErrors(t *testing.T) {
	for name, data := range map[string]string{
		"duplicate": "- slug: a\n  name: A\n- slug: a\n  name: B\n",
		"bad slug":  "- slug: A B\n  name: A\n",
		"no name":   "- slug: a\n",
	} {
		t.Run(name, func(t *testing.T) {
			fsys := fstest.MapFS{"authors.yaml": {Data: []byte(data)}}
			if _, err := content.Load(fsys, content.LoadOptions{}); err == nil || !strings.Contains(err.Error(), "authors.yaml") {
				t.Errorf("err = %v, want authors.yaml error", err)
			}
		})
	}
}

func TestTagSameSlugSameName(t *testing.T) {
	site := mustLoad(t, fixture(map[string]string{
		"posts/m.md": post("m", "2020-01-01", "tags: [récit]\n", "x"),
		"posts/n.md": post("n", "2020-01-02", "tags: [récit, Absurde]\n", "x"),
	}), content.LoadOptions{})
	want := []content.Tag{{Slug: "absurde", Name: "Absurde"}, {Slug: "recit", Name: "récit"}}
	if len(site.Tags) != 2 || site.Tags[0] != want[0] || site.Tags[1] != want[1] {
		t.Errorf("Tags = %+v, want %+v", site.Tags, want)
	}
}

func TestValidationCoversDrafts(t *testing.T) {
	_, err := content.Load(fixture(map[string]string{
		"posts/d.md": post("d", "bientôt", "status: draft\n", "x"),
	}), content.LoadOptions{})
	if err == nil || !strings.Contains(err.Error(), "posts/d.md") {
		t.Errorf("err = %v, want error on draft", err)
	}
}

func TestParisIsParis(t *testing.T) {
	if content.Paris.String() != "Europe/Paris" {
		t.Errorf("Paris = %v", content.Paris)
	}
	_ = time.Now().In(content.Paris)
}
