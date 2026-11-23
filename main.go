// Command unblogexutoire serves the "Un blog exutoire" blog from the content embedded in
// the binary.
package main

import (
	"context"
	"errors"
	"flag"
	"io/fs"
	"log"
	"net"
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

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	log.Printf("serving %d posts on %s (drafts: %t)", len(c.Posts), addr, drafts)
	return serve(ctx, ln, s.Handler())
}

// serve handles requests on ln until ctx is cancelled, then shuts down gracefully.
func serve(ctx context.Context, ln net.Listener, h http.Handler) error {
	srv := &http.Server{Handler: h, ReadHeaderTimeout: 5 * time.Second}
	served := make(chan error, 1)
	go func() { served <- srv.Serve(ln) }()

	select {
	case err := <-served: // failed before any shutdown was asked for
		return err
	case <-ctx.Done():
	}
	// Serve returns as soon as Shutdown starts; Shutdown itself returns once in-flight
	// requests are answered (or the deadline passes), so wait on it, not on Serve.
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdown); err != nil {
		return err
	}
	if err := <-served; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
