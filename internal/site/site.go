// Package site renders the blog and serves it over HTTP. Every page that doesn't depend on
// the request is rendered once, at startup, and kept in memory with its gzip version.
package site

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html/template"
	"io/fs"
	"slices"
	"time"

	"github.com/LandazuriPaul/unblogexutoire/internal/content"
)

// SiteName is the blog's only branding.
const SiteName = "Un blog exutoire"

// Config is what New needs to build the site.
type Config struct {
	Content *content.Site
	Assets  fs.FS  // contains templates/*.html and static/style.css
	BaseURL string // without a trailing slash
}

// Site holds the pre-rendered pages and the templates for dynamic ones.
type Site struct {
	cfg       Config
	templates map[string]*template.Template // by file name, each joined with base.html
	routes    map[string]*entry
	cssPath   string
	css       *entry
	notFound  *entry
}

// view is the data every template receives.
type view struct {
	Title       string // page title; empty on the home page
	Description string
	Path        string // canonical path, for og:url
	OGType      string // "website" or "article"
	Data        any
}

// New parses the templates and pre-renders every static route.
func New(cfg Config) (*Site, error) {
	s := &Site{cfg: cfg, templates: map[string]*template.Template{}, routes: map[string]*entry{}}

	css, err := fs.ReadFile(cfg.Assets, "static/style.css")
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(css)
	s.cssPath = "/static/style." + hex.EncodeToString(sum[:])[:10] + ".css"
	s.css = newEntry("text/css; charset=utf-8", "public, max-age=31536000, immutable", css)

	if err := s.parseTemplates(); err != nil {
		return nil, err
	}
	notFound, err := s.render("404.html", view{Title: "Page introuvable", Path: "/404/", OGType: "website"})
	if err != nil {
		return nil, err
	}
	s.notFound = newEntry("text/html; charset=utf-8", "public, max-age=60", notFound)

	for _, r := range s.pages() {
		if err := s.renderPage(r); err != nil {
			return nil, fmt.Errorf("render %s: %w", r.path, err)
		}
	}

	feed, err := s.feed()
	if err != nil {
		return nil, fmt.Errorf("render feed: %w", err)
	}
	s.put("/rss/", "application/atom+xml; charset=utf-8", feed)
	sm, err := s.sitemap()
	if err != nil {
		return nil, fmt.Errorf("render sitemap: %w", err)
	}
	s.put("/sitemap.xml", "application/xml; charset=utf-8", sm)
	s.put("/robots.txt", "text/plain; charset=utf-8", s.robots())
	return s, nil
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// pageRoute is a pre-rendered HTML route.
type pageRoute struct {
	path, template string
	view           view
}

// pages lists every pre-rendered HTML route.
func (s *Site) pages() []pageRoute {
	c := s.cfg.Content
	routes := []pageRoute{
		{"/", "home.html", view{Description: "Textes, récits et coups de gueule de quelques ami·es.", Path: "/", OGType: "website",
			Data: c.Posts}},
	}
	for _, p := range c.Pages {
		path := "/" + p.Slug + "/"
		routes = append(routes, pageRoute{path, "page.html",
			view{Title: p.Title, Description: p.Description, Path: path, OGType: "website", Data: p}})
	}
	for i, p := range c.Posts {
		data := postData{Post: p}
		if i+1 < len(c.Posts) {
			data.Prev = c.Posts[i+1]
		}
		if i > 0 {
			data.Next = c.Posts[i-1]
		}
		path := "/" + p.Slug + "/"
		routes = append(routes, pageRoute{path, "post.html",
			view{Title: p.Title, Description: p.Description, Path: path, OGType: "article", Data: data}})
	}
	for _, a := range c.Authors {
		heading := "Articles de " + a.Name
		routes = append(routes, s.listRoute("/author/"+a.Slug+"/", heading, a.Bio, func(p *content.Post) bool {
			return slices.Contains(p.Authors, a)
		}))
	}
	for _, t := range c.Tags {
		heading := "Articles étiquetés « " + t.Name + " »"
		routes = append(routes, s.listRoute("/tag/"+t.Slug+"/", heading, "", func(p *content.Post) bool {
			return slices.Contains(p.Tags, t)
		}))
	}
	return routes
}

// postData is a post with its older (Prev) and newer (Next) neighbours.
type postData struct {
	Post       *content.Post
	Prev, Next *content.Post
}

// listData is an author or tag page.
type listData struct {
	Heading, Intro string
	Posts          []*content.Post
}

func (s *Site) listRoute(path, heading, intro string, keep func(*content.Post) bool) pageRoute {
	var posts []*content.Post
	for _, p := range s.cfg.Content.Posts {
		if keep(p) {
			posts = append(posts, p)
		}
	}
	description := heading + " sur " + SiteName + "."
	return pageRoute{path, "list.html", view{Title: heading, Description: description, Path: path, OGType: "website",
		Data: listData{Heading: heading, Intro: intro, Posts: posts}}}
}

func (s *Site) renderPage(r pageRoute) error {
	body, err := s.render(r.template, r.view)
	if err != nil {
		return err
	}
	s.put(r.path, "text/html; charset=utf-8", body)
	return nil
}

func (s *Site) parseTemplates() error {
	funcs := template.FuncMap{
		"css":      func() string { return s.cssPath },
		"siteName": func() string { return SiteName },
		"url":      func(path string) string { return s.cfg.BaseURL + path },
		"frdate":   FormatDate,
		"isodate":  func(t time.Time) string { return t.Format(time.RFC3339) },
		"hasPage":  func(slug string) bool { return s.cfg.Content.Pages[slug] != nil },
	}
	names, err := fs.Glob(s.cfg.Assets, "templates/*.html")
	if err != nil {
		return err
	}
	for _, name := range names {
		base := name[len("templates/"):]
		if base == "base.html" {
			continue
		}
		t, err := template.New("base.html").Funcs(funcs).ParseFS(s.cfg.Assets, "templates/base.html", name)
		if err != nil {
			return err
		}
		s.templates[base] = t
	}
	return nil
}

func (s *Site) render(name string, v view) ([]byte, error) {
	t := s.templates[name]
	if t == nil {
		return nil, fmt.Errorf("no template %s", name)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// put registers a pre-rendered route.
func (s *Site) put(path, contentType string, body []byte) {
	s.routes[path] = newEntry(contentType, "public, max-age=300", body)
}
