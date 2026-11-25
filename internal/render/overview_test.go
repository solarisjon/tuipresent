package render

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"tuipresent/internal/deck"
)

func overviewDeck(t *testing.T, slides int) *deck.Deck {
	t.Helper()
	src := "---\ntitle: Big Talk\n---\n"
	for i := 1; i <= slides; i++ {
		src += fmt.Sprintf("Slide number %d with a rather long title that must wrap\n===\n\nbody\n<!-- end_slide -->\n", i)
	}
	d, err := deck.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestOverviewDimensions(t *testing.T) {
	r := New(overviewDeck(t, 30))
	for _, sz := range [][2]int{{100, 30}, {80, 24}, {40, 15}, {30, 8}, {20, 4}, {220, 60}} {
		for _, sel := range []int{0, 7, 30} {
			checkFrame(t, r.Overview(sel, sz[0], sz[1]), sz[0], sz[1])
		}
	}
}

func TestOverviewContent(t *testing.T) {
	r := New(overviewDeck(t, 30))
	plain := ansi.Strip(r.Overview(0, 100, 30))
	for _, want := range []string{"Overview  1 / 31", "Big Talk", "Slide number 1 with", "enter open"} {
		if !strings.Contains(plain, want) {
			t.Errorf("overview lacks %q:\n%s", want, plain)
		}
	}
	// 4 columns x 5 rows per page: slide 31 is on the last page, slide 1 is not.
	last := ansi.Strip(r.Overview(30, 100, 30))
	if !strings.Contains(last, "Overview  31 / 31") || strings.Contains(last, "Big Talk") {
		t.Errorf("last page wrong:\n%s", last)
	}
}
