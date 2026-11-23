// Command unblogexutoire serves the "Un blog exutoire" blog from the content embedded in
// the binary.
package main

import (
	"context"
	"errors"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "time/tzdata"

	"github.com/LandazuriPaul/unblogexutoire/internal/content"
	"github.com/LandazuriPaul/unblogexutoire/internal/site"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	drafts := flag.Bool("drafts", false, "serve drafts too (local preview only)")
	baseURL := flag.String("base-url", "https://unblogexutoire.ninja", "public URL, used in the feed, sitemap and Open Graph tags")
	flag.Parse()

	if err := run(*addr, *drafts, strings.TrimSuffix(*baseURL, "/")); err != nil {
		log.Fatal(err)
	}
}

func run(addr string, drafts bool, baseURL string) error {
	contentFS, err := fs.Sub(files, "content")
	if err != nil {
		return err
	}
	c, err := content.Load(contentFS, content.LoadOptions{IncludeDrafts: drafts})
	if err != nil {
		return err
	}
	s, err := site.New(site.Config{Content: c, Assets: files, BaseURL: baseURL})
	if err != nil {
		return err
	}

	srv := &http.Server{Addr: addr, Handler: s.Handler(), ReadHeaderTimeout: 5 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdown)
	}()

	log.Printf("serving %d posts on %s (drafts: %t)", len(c.Posts), addr, drafts)
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
