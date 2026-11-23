// Package search filters posts for the /articles/ page.
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
}

// ParseQuery reads q, auteur, tag, annee and mois. Invalid numbers are ignored, and so is
// any other parameter (such as page, from when results were paginated).
func ParseQuery(v url.Values) Query {
	q := Query{
		Text:   strings.TrimSpace(v.Get("q")),
		Author: v.Get("auteur"),
		Tag:    v.Get("tag"),
		Year:   intIn(v.Get("annee"), 1, 9999),
		Month:  intIn(v.Get("mois"), 1, 12),
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

// IsZero reports whether no filter is set.
func (q Query) IsZero() bool {
	return q == Query{}
}

// Run returns every post matching q, in the order given (newest first).
func Run(posts []*content.Post, q Query) []*content.Post {
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
	return matched
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
