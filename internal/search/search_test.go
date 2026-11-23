package search_test

import (
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/LandazuriPaul/unblogexutoire/internal/content"
	"github.com/LandazuriPaul/unblogexutoire/internal/search"
	"github.com/LandazuriPaul/unblogexutoire/internal/text"
)

var (
	alice = &content.Author{Slug: "alice", Name: "Alice"}
	bob   = &content.Author{Slug: "bob", Name: "Bob"}
)

func mk(slug, date, body string, a *content.Author, tags ...string) *content.Post {
	d, err := time.Parse(time.RFC3339, date)
	if err != nil {
		panic(err)
	}
	p := &content.Post{Slug: slug, Title: slug, Date: d.In(content.Paris), Authors: []*content.Author{a}}
	for _, t := range tags {
		p.Tags = append(p.Tags, content.Tag{Slug: text.Slugify(t), Name: t})
	}
	p.SearchText = text.Fold(p.Title + " " + body)
	return p
}

func slugs(posts []*content.Post) string {
	var s []string
	for _, p := range posts {
		s = append(s, p.Slug)
	}
	return strings.Join(s, ",")
}

func TestParseQuery(t *testing.T) {
	cases := map[string]search.Query{
		"page=2":                 {}, // old paginated links still work: page is ignored
		"annee=99999":            {},
		"annee=2019&mois=13":     {Year: 2019},
		"mois=0":                 {},
		"q=++été++&auteur=alice": {Text: "été", Author: "alice"},
		"tag=recit&mois=6":       {Tag: "recit", Month: 6},
	}
	for raw, want := range cases {
		v, _ := url.ParseQuery(raw)
		if got := search.ParseQuery(v); got != want {
			t.Errorf("ParseQuery(%q) = %+v, want %+v", raw, got, want)
		}
	}
	q := search.ParseQuery(url.Values{"q": {strings.Repeat("a", 10_000)}})
	if n := len([]rune(q.Text)); n != 200 {
		t.Errorf("Text length = %d, want 200", n)
	}
}

func TestTextMatchAND(t *testing.T) {
	posts := []*content.Post{
		mk("both", "2020-01-02T00:00:00Z", "Un cœur en été", alice),
		mk("one", "2020-01-01T00:00:00Z", "Un cœur en hiver", alice),
	}
	if got := slugs(search.Run(posts, search.Query{Text: "coeur ete"})); got != "both" {
		t.Errorf("got %q, want both", got)
	}
	if got := slugs(search.Run(posts, search.Query{Text: "COEUR"})); got != "both,one" {
		t.Errorf("got %q, want both,one", got)
	}
}

func TestTextAccentsAndApostrophes(t *testing.T) {
	posts := []*content.Post{mk("p", "2020-01-01T00:00:00Z", "Vivement L’Été !", alice)}
	if got := slugs(search.Run(posts, search.Query{Text: "l'ete"})); got != "p" {
		t.Errorf("got %q, want p", got)
	}
}

func TestFilters(t *testing.T) {
	posts := []*content.Post{
		mk("a20", "2020-06-15T10:00:00Z", "x", alice, "Récit"),
		mk("b19", "2019-06-15T10:00:00Z", "x", bob, "Politique"),
		mk("a19", "2019-03-15T10:00:00Z", "x", alice, "Politique"),
	}
	for _, tc := range []struct {
		q    search.Query
		want string
	}{
		{search.Query{Author: "alice"}, "a20,a19"},
		{search.Query{Tag: "politique"}, "b19,a19"},
		{search.Query{Year: 2019}, "b19,a19"},
		{search.Query{Year: 2019, Month: 6}, "b19"},
		{search.Query{Month: 6}, "a20,b19"},
		{search.Query{Author: "alice", Tag: "politique"}, "a19"},
		{search.Query{Author: "nope"}, ""},
	} {
		if got := slugs(search.Run(posts, tc.q)); got != tc.want {
			t.Errorf("%+v: got %q, want %q", tc.q, got, tc.want)
		}
	}
}

func TestYearBoundaryParis(t *testing.T) {
	posts := []*content.Post{mk("nye", "2019-12-31T23:30:00Z", "x", alice)}
	if got := slugs(search.Run(posts, search.Query{Year: 2020, Month: 1})); got != "nye" {
		t.Errorf("got %q, want nye in January 2020 (Paris)", got)
	}
	if got := slugs(search.Run(posts, search.Query{Year: 2019})); got != "" {
		t.Errorf("got %q, want nothing in 2019", got)
	}
}

func TestReturnsAllMatches(t *testing.T) {
	var posts []*content.Post
	for i := range 23 {
		posts = append(posts, mk(fmt.Sprintf("p%02d", i), "2020-01-01T00:00:00Z", "x", alice))
	}
	got := search.Run(posts, search.Query{})
	if len(got) != 23 || got[0].Slug != "p00" || got[22].Slug != "p22" {
		t.Errorf("got %d posts, want all 23 in their original order", len(got))
	}
	if got := search.Run(nil, search.Query{}); len(got) != 0 {
		t.Errorf("no posts: got %d", len(got))
	}
}

func TestIsZero(t *testing.T) {
	if !(search.Query{}).IsZero() || (search.Query{Month: 6}).IsZero() {
		t.Error("IsZero wrong")
	}
}
