// Package content loads and validates the blog's Markdown posts, pages and authors.
package content

import (
	"cmp"
	"errors"
	"html/template"
	"io/fs"
	"slices"
	"time"

	_ "time/tzdata" // Paris must resolve even on images without a zoneinfo database.
)

// Paris is the time zone every date is shown and filtered in.
var Paris = mustLoadLocation("Europe/Paris")

// ReservedSlugs are top-level routes no post or page may take.
var ReservedSlugs = []string{"articles", "author", "tag", "rss", "static", "healthz", "sitemap.xml", "robots.txt"}

// Author is an entry of authors.yaml.
type Author struct{ Slug, Name, Bio string }

// Tag is a post tag; Slug is derived from Name.
type Tag struct{ Slug, Name string }

// Post is a dated article by one or more authors.
type Post struct {
	Slug, Title, Description string // Description: frontmatter value, else an excerpt of the body
	Date                     time.Time
	Authors                  []*Author
	Tags                     []Tag
	Draft                    bool
	HTML                     template.HTML
	SearchText               string // folded title, description and plain body
}

// Page is a standalone, undated page such as "à propos".
type Page struct {
	Slug, Title, Description string
	Draft                    bool
	HTML                     template.HTML
}

// Site is everything loaded from the content directory.
type Site struct {
	Posts   []*Post          // newest first
	Pages   map[string]*Page // by slug
	Authors []*Author        // authors with at least one loaded post, by name
	Tags    []Tag            // tags with at least one loaded post, by slug
	Years   []int            // years with at least one loaded post, descending
}

// LoadOptions tunes what Load keeps.
type LoadOptions struct {
	IncludeDrafts bool
}

// Load parses and validates every file under fsys (rooted at the content directory). All
// files are validated, drafts included; drafts are only kept when opts.IncludeDrafts is set.
// The error, if any, joins one message per problem, each prefixed with its file path.
func Load(fsys fs.FS, opts LoadOptions) (*Site, error) {
	l := &loader{fsys: fsys, slugFiles: map[string]string{}, tagNames: map[string]tagSeen{}}
	authors := l.loadAuthors()
	posts := l.loadPosts(authors)
	pages := l.loadPages()
	if len(l.errs) > 0 {
		return nil, errors.Join(l.errs...)
	}

	site := &Site{Pages: map[string]*Page{}}
	for _, p := range posts {
		if p.Draft && !opts.IncludeDrafts {
			continue
		}
		site.Posts = append(site.Posts, p)
	}
	for _, p := range pages {
		if p.Draft && !opts.IncludeDrafts {
			continue
		}
		site.Pages[p.Slug] = p
	}
	slices.SortStableFunc(site.Posts, func(a, b *Post) int { return b.Date.Compare(a.Date) })
	site.index()
	return site, nil
}

// index derives Authors, Tags and Years from the loaded posts.
func (s *Site) index() {
	authors := map[string]*Author{}
	tags := map[string]Tag{}
	years := map[int]bool{}
	for _, p := range s.Posts {
		for _, a := range p.Authors {
			authors[a.Slug] = a
		}
		for _, t := range p.Tags {
			tags[t.Slug] = t
		}
		years[p.Date.Year()] = true
	}
	for _, a := range authors {
		s.Authors = append(s.Authors, a)
	}
	slices.SortFunc(s.Authors, func(a, b *Author) int { return cmp.Compare(a.Name, b.Name) })
	for _, t := range tags {
		s.Tags = append(s.Tags, t)
	}
	slices.SortFunc(s.Tags, func(a, b Tag) int { return cmp.Compare(a.Slug, b.Slug) })
	for y := range years {
		s.Years = append(s.Years, y)
	}
	slices.SortFunc(s.Years, func(a, b int) int { return cmp.Compare(b, a) })
}

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}
