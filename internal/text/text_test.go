package text_test

import (
	"slices"
	"testing"

	"github.com/LandazuriPaul/unblogexutoire/internal/text"
)

func TestFold(t *testing.T) {
	for in, want := range map[string]string{
		"ÉTÉ":     "ete",
		"Cœur":    "coeur",
		"l’été":   "l'ete",
		"Æther":   "aether",
		"Ça":      "ca",
		"déjà-vu": "deja-vu",
	} {
		if got := text.Fold(in); got != want {
			t.Errorf("Fold(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSlugify(t *testing.T) {
	for in, want := range map[string]string{
		"Récit":                     "recit",
		"  La Choucroute Masquée! ": "la-choucroute-masquee",
		"l’été":                     "l-ete",
		"---":                       "",
		"174 boulevard":             "174-boulevard",
	} {
		if got := text.Slugify(in); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestWords(t *testing.T) {
	if got, want := text.Words("L’été, c'est  COOL!"), []string{"l", "ete", "c", "est", "cool"}; !slices.Equal(got, want) {
		t.Errorf("Words = %q, want %q", got, want)
	}
	if got := text.Words(""); len(got) != 0 {
		t.Errorf("Words(\"\") = %q, want empty", got)
	}
}

func TestExcerpt(t *testing.T) {
	if got, want := text.Excerpt("un deux  trois\nquatre", 3), "un deux trois…"; got != want {
		t.Errorf("Excerpt = %q, want %q", got, want)
	}
	if got, want := text.Excerpt("un deux", 3), "un deux"; got != want {
		t.Errorf("Excerpt = %q, want %q", got, want)
	}
}
