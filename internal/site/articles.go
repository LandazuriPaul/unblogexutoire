package site

import (
	"net/http"
	"strconv"

	"github.com/LandazuriPaul/unblogexutoire/internal/content"
	"github.com/LandazuriPaul/unblogexutoire/internal/search"
)

const (
	articlesPath    = "/articles/"
	articlesPerPage = 10
)

type monthOption struct {
	Value int
	Name  string
}

// articlesData feeds templates/articles.html.
type articlesData struct {
	Query            search.Query
	Result           search.Result
	Authors          []*content.Author
	Tags             []content.Tag
	Years            []int
	Months           []monthOption
	PrevURL, NextURL string
}

// serveArticles renders the search page for this request's query string.
func (s *Site) serveArticles(w http.ResponseWriter, r *http.Request) {
	c := s.cfg.Content
	q := search.ParseQuery(r.URL.Query())
	data := articlesData{
		Query:   q,
		Result:  search.Run(c.Posts, q, articlesPerPage),
		Authors: c.Authors,
		Tags:    c.Tags,
		Years:   c.Years,
	}
	for i, name := range months {
		data.Months = append(data.Months, monthOption{Value: i + 1, Name: name})
	}
	if data.Result.PrevPage != 0 {
		data.PrevURL = pageURL(q, data.Result.PrevPage)
	}
	if data.Result.NextPage != 0 {
		data.NextURL = pageURL(q, data.Result.NextPage)
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

// pageURL links to page n of the results, keeping the filters.
func pageURL(q search.Query, n int) string {
	v := q.Values()
	v.Set("page", strconv.Itoa(n))
	return articlesPath + "?" + v.Encode()
}
