package site

import (
	"fmt"
	"time"

	"github.com/LandazuriPaul/unblogexutoire/internal/content"
)

var months = [...]string{"janvier", "février", "mars", "avril", "mai", "juin", "juillet",
	"août", "septembre", "octobre", "novembre", "décembre"}

// FormatDate renders t in Paris time as a long French date: "10 juin 2019", "1er janvier 2020".
func FormatDate(t time.Time) string {
	t = t.In(content.Paris)
	day := fmt.Sprint(t.Day())
	if t.Day() == 1 {
		day = "1er"
	}
	return fmt.Sprintf("%s %s %d", day, months[t.Month()-1], t.Year())
}
