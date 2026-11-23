// Package search filters and paginates posts for the /articles/ page.
package search

import (
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/LandazuriPaul/unblogexutoire/internal/content"
	"github.com/LandazuriPaul/unblogexutoire/internal/text"
)

const (
	maxTextRunes = 200
	maxWords     = 10
)

// Query is a parsed /articles/ request. Zero values mean "any".
type Query struct {
	Text        string // trimmed, at most 200 runes
	Author, Tag string // slugs
	Year, Month int    // Year in 1..9999, Month in 1..12
	Page        int    // >= 1
}

// ParseQuery reads q, auteur, tag, annee, mois and page. Invalid numbers are ignored.
func ParseQuery(v url.Values) Query {
	q := Query{
		Text:   strings.TrimSpace(v.Get("q")),
		Author: v.Get("auteur"),
		Tag:    v.Get("tag"),
		Year:   intIn(v.Get("annee"), 1, 9999),
		Month:  intIn(v.Get("mois"), 1, 12),
		Page:   max(intIn(v.Get("page"), 1, 1<<30), 1),
	}
	if r := []rune(q.Text); len(r) > maxTextRunes {
		q.Text = strings.TrimSpace(string(r[:maxTextRunes]))
	}
	return q
}

func intIn(s string, lo, hi int) int {
	n, err := strconv.Atoi(s)
	if err != nil || n < lo || n > hi {
		return 0
	}
	return n
}

// Values encodes the filters (never the page), for building links that keep them.
func (q Query) Values() url.Values {
	v := url.Values{}
	set := func(k, s string) {
		if s != "" {
			v.Set(k, s)
		}
	}
	set("q", q.Text)
	set("auteur", q.Author)
	set("tag", q.Tag)
	if q.Year != 0 {
		v.Set("annee", strconv.Itoa(q.Year))
	}
	if q.Month != 0 {
		v.Set("mois", strconv.Itoa(q.Month))
	}
	return v
}

// IsZero reports whether no filter is set; the page is ignored.
func (q Query) IsZero() bool {
	q.Page = 0
	return q == Query{}
}

// Result is one page of matching posts.
type Result struct {
	Posts              []*content.Post
	Total, Page, Pages int // Pages >= 1
	PrevPage, NextPage int // 0 = no link
}

// Run filters posts (kept in their order) and returns page q.Page of perPage results.
func Run(posts []*content.Post, q Query, perPage int) Result {
	words := text.Words(q.Text)
	if len(words) > maxWords {
		words = words[:maxWords]
	}
	var matched []*content.Post
	for _, p := range posts {
		if matches(p, q, words) {
			matched = append(matched, p)
		}
	}

	page := max(q.Page, 1)
	r := Result{Total: len(matched), Page: page, Pages: max((len(matched)+perPage-1)/perPage, 1)}
	if start := (page - 1) * perPage; start < len(matched) {
		r.Posts = matched[start:min(start+perPage, len(matched))]
	}
	if page > 1 && page <= r.Pages {
		r.PrevPage = page - 1
	}
	if page < r.Pages {
		r.NextPage = page + 1
	}
	return r
}

func matches(p *content.Post, q Query, words []string) bool {
	if q.Author != "" && !slices.ContainsFunc(p.Authors, func(a *content.Author) bool { return a.Slug == q.Author }) {
		return false
	}
	if q.Tag != "" && !slices.ContainsFunc(p.Tags, func(t content.Tag) bool { return t.Slug == q.Tag }) {
		return false
	}
	if q.Year != 0 && p.Date.Year() != q.Year {
		return false
	}
	if q.Month != 0 && int(p.Date.Month()) != q.Month {
		return false
	}
	for _, w := range words {
		if !strings.Contains(p.SearchText, w) {
			return false
		}
	}
	return true
}
