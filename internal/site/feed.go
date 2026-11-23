package site

import (
	"encoding/xml"
	"time"

	"github.com/LandazuriPaul/unblogexutoire/internal/content"
)

type atomLink struct {
	Rel  string `xml:"rel,attr"`
	Type string `xml:"type,attr,omitempty"`
	Href string `xml:"href,attr"`
}

type atomText struct {
	Type string `xml:"type,attr"`
	Body string `xml:",chardata"`
}

type atomAuthor struct {
	Name string `xml:"name"`
}

type atomEntry struct {
	ID        string       `xml:"id"`
	Title     string       `xml:"title"`
	Link      atomLink     `xml:"link"`
	Published string       `xml:"published"`
	Updated   string       `xml:"updated"`
	Authors   []atomAuthor `xml:"author"`
	Summary   string       `xml:"summary"`
	Content   atomText     `xml:"content"`
}

type atomFeed struct {
	XMLName xml.Name    `xml:"http://www.w3.org/2005/Atom feed"`
	Title   string      `xml:"title"`
	ID      string      `xml:"id"`
	Updated string      `xml:"updated"`
	Links   []atomLink  `xml:"link"`
	Entries []atomEntry `xml:"entry"`
}

// feed renders the Atom feed of every loaded post, with full content.
func (s *Site) feed() ([]byte, error) {
	base := s.cfg.BaseURL
	posts := s.cfg.Content.Posts
	updated := time.Unix(0, 0).In(content.Paris)
	if len(posts) > 0 {
		updated = posts[0].Date
	}
	f := atomFeed{
		Title:   SiteName,
		ID:      base + "/",
		Updated: updated.Format(time.RFC3339),
		Links: []atomLink{
			{Rel: "self", Type: "application/atom+xml", Href: base + "/rss/"},
			{Rel: "alternate", Type: "text/html", Href: base + "/"},
		},
	}
	for _, p := range posts {
		url := base + "/" + p.Slug + "/"
		date := p.Date.Format(time.RFC3339)
		e := atomEntry{
			ID: url, Title: p.Title, Link: atomLink{Rel: "alternate", Href: url},
			Published: date, Updated: date, Summary: p.Description,
			Content: atomText{Type: "html", Body: string(p.HTML)},
		}
		for _, a := range p.Authors {
			e.Authors = append(e.Authors, atomAuthor{Name: a.Name})
		}
		f.Entries = append(f.Entries, e)
	}
	return marshalXML(f)
}

type sitemapURL struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

type sitemap struct {
	XMLName xml.Name     `xml:"http://www.sitemaps.org/schemas/sitemap/0.9 urlset"`
	URLs    []sitemapURL `xml:"url"`
}

// sitemap lists every public HTML page; only posts carry a lastmod.
func (s *Site) sitemap() ([]byte, error) {
	c, base := s.cfg.Content, s.cfg.BaseURL
	sm := sitemap{URLs: []sitemapURL{{Loc: base + "/"}, {Loc: base + articlesPath}}}
	for _, p := range c.Posts {
		sm.URLs = append(sm.URLs, sitemapURL{Loc: base + "/" + p.Slug + "/", LastMod: p.Date.Format(time.DateOnly)})
	}
	for _, slug := range sortedKeys(c.Pages) {
		sm.URLs = append(sm.URLs, sitemapURL{Loc: base + "/" + slug + "/"})
	}
	for _, a := range c.Authors {
		sm.URLs = append(sm.URLs, sitemapURL{Loc: base + "/author/" + a.Slug + "/"})
	}
	for _, t := range c.Tags {
		sm.URLs = append(sm.URLs, sitemapURL{Loc: base + "/tag/" + t.Slug + "/"})
	}
	return marshalXML(sm)
}

func (s *Site) robots() []byte {
	return []byte("User-agent: *\nAllow: /\nSitemap: " + s.cfg.BaseURL + "/sitemap.xml\n")
}

func marshalXML(v any) ([]byte, error) {
	out, err := xml.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), append(out, '\n')...), nil
}
