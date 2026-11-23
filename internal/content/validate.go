package content

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/LandazuriPaul/unblogexutoire/internal/text"
)

const authorsFile = "authors.yaml"

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

type postMeta struct {
	Title       string   `yaml:"title"`
	Slug        string   `yaml:"slug"`
	Date        string   `yaml:"date"`
	Authors     []string `yaml:"authors"`
	Tags        []string `yaml:"tags"`
	Description string   `yaml:"description"`
	Status      string   `yaml:"status"`
}

type pageMeta struct {
	Title       string `yaml:"title"`
	Slug        string `yaml:"slug"`
	Description string `yaml:"description"`
	Status      string `yaml:"status"`
}

type tagSeen struct{ name, file string }

// loader accumulates validation errors across all files.
type loader struct {
	fsys      fs.FS
	errs      []error
	slugFiles map[string]string  // post/page slug -> file that claimed it
	tagNames  map[string]tagSeen // tag slug -> first display name and its file
}

func (l *loader) fail(file, format string, args ...any) {
	l.errs = append(l.errs, fmt.Errorf("%s: %s", file, fmt.Sprintf(format, args...)))
}

func (l *loader) loadAuthors() map[string]*Author {
	authors := map[string]*Author{}
	data, err := fs.ReadFile(l.fsys, authorsFile)
	if err != nil {
		l.fail(authorsFile, "%v", err)
		return authors
	}
	var list []struct {
		Slug string `yaml:"slug"`
		Name string `yaml:"name"`
		Bio  string `yaml:"bio"`
	}
	if err := decodeStrict(data, &list); err != nil {
		l.fail(authorsFile, "%v", err)
		return authors
	}
	for i, a := range list {
		switch {
		case !slugPattern.MatchString(a.Slug):
			l.fail(authorsFile, "entry %d: invalid slug %q (use lowercase letters, digits and dashes)", i+1, a.Slug)
		case a.Name == "":
			l.fail(authorsFile, "author %q: missing name", a.Slug)
		case authors[a.Slug] != nil:
			l.fail(authorsFile, "duplicate author slug %q", a.Slug)
		default:
			authors[a.Slug] = &Author{Slug: a.Slug, Name: a.Name, Bio: a.Bio}
		}
	}
	return authors
}

// files returns the *.md files directly under dir. Anything else in dir (another extension,
// a subdirectory) is an error, so a misnamed post fails loudly instead of silently vanishing.
// Hidden files are ignored: embed never includes them.
func (l *loader) files(dir string) []string {
	var names []string
	err := fs.WalkDir(l.fsys, dir, func(p string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			if errors.Is(err, fs.ErrNotExist) && p == dir {
				return nil
			}
			return err
		case p == dir:
			return nil
		case strings.HasPrefix(d.Name(), "."):
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		case d.IsDir():
			return nil // its files are reported below
		case path.Dir(p) != dir:
			l.fail(p, "unexpected subdirectory: put the file directly in %s/", dir)
			return nil
		case path.Ext(p) != ".md":
			l.fail(p, "unexpected file: %s/ only holds files ending in .md", dir)
			return nil
		}
		names = append(names, p)
		return nil
	})
	if err != nil {
		l.fail(dir, "%v", err)
	}
	return names
}

func (l *loader) loadPosts(authors map[string]*Author) []*Post {
	var posts []*Post
	for _, file := range l.files("posts") {
		var meta postMeta
		body, ok := l.readMeta(file, &meta)
		if !ok {
			continue
		}
		before := len(l.errs)
		l.checkCommon(file, meta.Title, meta.Slug, meta.Status)
		date := l.checkDate(file, meta.Date)
		if len(meta.Authors) == 0 {
			l.fail(file, "missing authors")
		}
		p := &Post{Slug: meta.Slug, Title: meta.Title, Date: date, Draft: meta.Status == "draft"}
		for _, slug := range meta.Authors {
			a := authors[slug]
			if a == nil {
				l.fail(file, "unknown author %q (not in %s)", slug, authorsFile)
				continue
			}
			p.Authors = append(p.Authors, a)
		}
		for _, name := range meta.Tags {
			if tag, ok := l.checkTag(file, name); ok && !slices.Contains(p.Tags, tag) {
				p.Tags = append(p.Tags, tag)
			}
		}
		html, plain, err := render(body)
		if err != nil {
			l.fail(file, "render: %v", err)
		}
		if len(l.errs) > before {
			continue
		}
		p.HTML = html
		p.Description = meta.Description
		if p.Description == "" {
			p.Description = text.Excerpt(plain, 30)
		}
		p.SearchText = text.Fold(p.Title + " " + p.Description + " " + plain)
		posts = append(posts, p)
	}
	return posts
}

func (l *loader) loadPages() []*Page {
	var pages []*Page
	for _, file := range l.files("pages") {
		var meta pageMeta
		body, ok := l.readMeta(file, &meta)
		if !ok {
			continue
		}
		before := len(l.errs)
		l.checkCommon(file, meta.Title, meta.Slug, meta.Status)
		html, plain, err := render(body)
		if err != nil {
			l.fail(file, "render: %v", err)
		}
		if len(l.errs) > before {
			continue
		}
		p := &Page{Slug: meta.Slug, Title: meta.Title, Description: meta.Description, Draft: meta.Status == "draft", HTML: html}
		if p.Description == "" {
			p.Description = text.Excerpt(plain, 30)
		}
		pages = append(pages, p)
	}
	return pages
}

// readMeta reads file, decodes its frontmatter into meta and returns the body.
func (l *loader) readMeta(file string, meta any) ([]byte, bool) {
	raw, err := fs.ReadFile(l.fsys, file)
	if err != nil {
		l.fail(file, "%v", err)
		return nil, false
	}
	front, body, err := splitFrontmatter(raw)
	if err != nil {
		l.fail(file, "%v", err)
		return nil, false
	}
	if err := decodeStrict(front, meta); err != nil {
		l.fail(file, "frontmatter: %v", err)
		return nil, false
	}
	return body, true
}

func (l *loader) checkCommon(file, title, slug, status string) {
	if title == "" {
		l.fail(file, "missing title")
	}
	switch {
	case slug == "":
		l.fail(file, "missing slug")
	case slices.Contains(ReservedSlugs, slug):
		l.fail(file, "slug %q is reserved for a site route", slug)
	case !slugPattern.MatchString(slug):
		l.fail(file, "invalid slug %q (use lowercase letters, digits and dashes)", slug)
	default:
		if other, dup := l.slugFiles[slug]; dup {
			l.fail(file, "duplicate slug %q (also used by %s)", slug, other)
		} else {
			l.slugFiles[slug] = file
		}
	}
	if status != "" && status != "published" && status != "draft" {
		l.fail(file, "invalid status %q (use published or draft)", status)
	}
}

func (l *loader) checkDate(file, value string) time.Time {
	if value == "" {
		l.fail(file, "missing date")
		return time.Time{}
	}
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t.In(Paris)
	}
	if t, err := time.ParseInLocation(time.DateOnly, value, Paris); err == nil {
		return t
	}
	l.fail(file, "invalid date %q (use YYYY-MM-DD or RFC 3339)", value)
	return time.Time{}
}

func (l *loader) checkTag(file, name string) (Tag, bool) {
	slug := text.Slugify(name)
	if slug == "" {
		l.fail(file, "tag %q has no letters or digits", name)
		return Tag{}, false
	}
	if seen, ok := l.tagNames[slug]; ok && seen.name != name {
		l.fail(file, "tag %q and tag %q (in %s) share the slug %q: use one spelling", name, seen.name, seen.file, slug)
		return Tag{}, false
	} else if !ok {
		l.tagNames[slug] = tagSeen{name: name, file: file}
	}
	return Tag{Slug: slug, Name: name}, true
}
