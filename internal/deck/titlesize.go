package deck

import (
	"fmt"
	"strconv"
	"strings"
)

// MaxTitleSize is the largest title_size.
const MaxTitleSize = 3

// ParseTitleSize reads a title_size value: an integer from 1 to MaxTitleSize.
func ParseTitleSize(v string) (int, bool) {
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil || n < 1 || n > MaxTitleSize {
		return 0, false
	}
	return n, true
}

// ValidTitleSize reports whether v is a title size.
func ValidTitleSize(v string) bool {
	_, ok := ParseTitleSize(v)
	return ok
}

// TitleSize is the title size of slide i: its own directive, else the front
// matter, else 1 (the normal title).
func (d *Deck) TitleSize(i int) int {
	for _, v := range []string{d.Slides[i].TitleSize, d.Meta.TitleSize} {
		if n, ok := ParseTitleSize(v); ok {
			return n
		}
	}
	return 1
}

func unknownTitleSize(v string) string {
	return fmt.Sprintf("unknown title_size %q (valid: 1 to %d); using 1", v, MaxTitleSize)
}
