package site

import (
	"net/http"

	"github.com/LandazuriPaul/unblogexutoire/internal/content"
	"github.com/LandazuriPaul/unblogexutoire/internal/search"
)

const articlesPath = "/articles/"

type monthOption struct {
	Value int
	Name  string
}

// articlesData feeds templates/articles.html.
type articlesData struct {
	Query   search.Query
	Posts   []*content.Post // every match, newest first
	Authors []*content.Author
	Tags    []content.Tag
	Years   []int
	Months  []monthOption
}

// serveArticles renders the search page for this request's query string.
func (s *Site) serveArticles(w http.ResponseWriter, r *http.Request) {
	c := s.cfg.Content
	q := search.ParseQuery(r.URL.Query())
	data := articlesData{
		Query:   q,
		Posts:   search.Run(c.Posts, q),
		Authors: c.Authors,
		Tags:    c.Tags,
		Years:   c.Years,
	}
	for i, name := range months {
		data.Months = append(data.Months, monthOption{Value: i + 1, Name: name})
	}

	body, err := s.render("articles.html", view{
		Title:       "Articles",
		Description: "Tous les articles de " + SiteName + ", par auteur, étiquette, date ou mot-clé.",
		Path:        articlesPath,
		OGType:      "website",
		Data:        data,
	})
	if err != nil {
		http.Error(w, "Erreur interne", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=60")
	w.Header().Set("Vary", "Accept-Encoding")
	var gz []byte
	if acceptsGzip(r) {
		gz = gzipBytes(body)
	}
	writeBody(w, r, http.StatusOK, body, gz)
}
