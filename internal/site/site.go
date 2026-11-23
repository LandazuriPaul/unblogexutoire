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
	"time"

	"github.com/LandazuriPaul/unblogexutoire/internal/content"
)

// SiteName is the blog's only branding.
const SiteName = "Un blog exutoire"

const homePosts = 5

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
	return s, nil
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
			Data: c.Posts[:min(homePosts, len(c.Posts))]}},
	}
	for _, p := range c.Pages {
		path := "/" + p.Slug + "/"
		routes = append(routes, pageRoute{path, "page.html",
			view{Title: p.Title, Description: p.Description, Path: path, OGType: "website", Data: p}})
	}
	return routes
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
