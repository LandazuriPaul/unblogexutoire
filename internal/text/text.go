// Package text holds the French-aware text helpers shared by content loading, search and
// rendering: accent folding, slugs, word splitting and excerpts.
package text

import (
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// Characters that accent stripping alone doesn't turn into ASCII.
var replacer = strings.NewReplacer(
	"œ", "oe", "Œ", "oe",
	"æ", "ae", "Æ", "ae",
	"’", "'", "‘", "'",
)

// Fold lowercases s and removes accents, ligatures and typographic apostrophes, so that
// "L’Été" and "l'ete" compare equal.
func Fold(s string) string {
	s = replacer.Replace(s)
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	folded, _, err := transform.String(t, s)
	if err != nil {
		folded = s
	}
	return strings.ToLower(folded)
}

// Slugify folds s and joins its runs of [a-z0-9] with single dashes.
func Slugify(s string) string {
	var b strings.Builder
	dash := false
	for _, r := range Fold(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			if dash && b.Len() > 0 {
				b.WriteByte('-')
			}
			dash = false
			b.WriteRune(r)
			continue
		}
		dash = true
	}
	return b.String()
}

// Words folds s and splits it into its letter/digit runs.
func Words(s string) []string {
	return strings.FieldsFunc(Fold(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

// Excerpt returns the first n words of plain, followed by "…" when it was truncated.
// Zero-width characters count as spaces: the Ghost export used them as line breaks.
func Excerpt(plain string, n int) string {
	words := strings.FieldsFunc(plain, func(r rune) bool {
		return unicode.IsSpace(r) || r == '​' || r == '‌' || r == '‍'
	})
	if len(words) <= n {
		return strings.Join(words, " ")
	}
	return strings.Join(words[:n], " ") + "…"
}
