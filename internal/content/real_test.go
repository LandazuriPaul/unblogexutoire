package content_test

import (
	"os"
	"testing"

	"github.com/LandazuriPaul/unblogexutoire/internal/content"
)

// TestRealContent validates the committed content/ directory: a post with a typo fails CI.
func TestRealContent(t *testing.T) {
	all, err := content.Load(os.DirFS("../../content"), content.LoadOptions{IncludeDrafts: true})
	if err != nil {
		t.Fatalf("content/ is invalid:\n%v", err)
	}
	if len(all.Posts) != 34 || all.Pages["a-propos"] == nil || len(all.Authors) != 4 {
		t.Errorf("with drafts: %d posts, a-propos=%v, %d authors; want 34, page, 4 (robert only wrote the page)",
			len(all.Posts), all.Pages["a-propos"] != nil, len(all.Authors))
	}

	pub, err := content.Load(os.DirFS("../../content"), content.LoadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(pub.Posts) != 26 {
		t.Errorf("published posts = %d, want 26", len(pub.Posts))
	}
	if d := pub.Posts[0].Date; d.Year() != 2020 || d.Month() != 8 || d.Day() != 27 {
		t.Errorf("newest post dated %v, want 2020-08-27", d)
	}
}
