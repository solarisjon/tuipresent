package render

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"tuipresent/internal/deck"
	"tuipresent/internal/img"
)

func imageDeck(t *testing.T, src string) *deck.Deck {
	t.Helper()
	d, err := deck.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	d.Dir = "../../examples"
	return d
}

func halfBlockRows(lines []string) int {
	n := 0
	for _, l := range lines {
		if strings.ContainsAny(ansi.Strip(l), "▀▄") {
			n++
		}
	}
	return n
}

func TestImageFillsFreeRows(t *testing.T) {
	d := imageDeck(t, "Logo\n===\n\n![logo](assets/terminal.png)\n\nCaption.\n")
	r := New(d)
	for _, sz := range [][2]int{{100, 30}, {80, 24}, {40, 15}, {220, 60}} {
		f := r.Frame(0, 0, sz[0], sz[1])
		checkFrame(t, f, sz[0], sz[1])
		if rows, cols := r.Overflow(0, sz[0], sz[1]); rows > 0 || cols > 0 {
			t.Errorf("%v: overflow %d rows %d cols", sz, rows, cols)
		}
		if halfBlockRows(strings.Split(f, "\n")) < minImageRows {
			t.Errorf("%v: image not drawn", sz)
		}
	}
	// At 100x30 the body has 28 rows: 1 top margin + 4 title + gap + caption leaves 21.
	f := ansi.Strip(r.Frame(0, 0, 100, 30))
	if n := halfBlockRows(strings.Split(f, "\n")); n != 21 {
		t.Errorf("image rows at 100x30 = %d, want 21", n)
	}
	if !strings.Contains(f, "Caption.") {
		t.Error("caption missing")
	}
}

func TestImageStableAcrossPauses(t *testing.T) {
	d := imageDeck(t, "Logo\n===\n\n![logo](assets/terminal.png)\n\n<!-- pause -->\n\nLater text.\n")
	r := New(d)
	a := strings.Split(r.Frame(0, 0, 80, 24), "\n")
	b := strings.Split(r.Frame(0, 1, 80, 24), "\n")
	if halfBlockRows(a) != halfBlockRows(b) || halfBlockRows(a) == 0 {
		t.Errorf("image size changed across pause: %d vs %d", halfBlockRows(a), halfBlockRows(b))
	}
}

func TestImageInColumns(t *testing.T) {
	d := imageDeck(t, "<!-- column_layout: [1, 1] -->\n<!-- column: 0 -->\n![logo](assets/terminal.png)\n<!-- column: 1 -->\n* one\n* two\n<!-- reset_layout -->\n")
	r := New(d)
	f := r.Frame(0, 0, 80, 24)
	checkFrame(t, f, 80, 24)
	for _, l := range strings.Split(ansi.Strip(f), "\n") {
		// Image is in the left half; column width is (72-4)/2 = 34.
		if i := strings.IndexAny(l, "▀▄"); i >= 0 && ansi.StringWidth(l[:i]) >= 40 {
			t.Errorf("image leaked into right column: %q", l)
		}
	}
	if halfBlockRows(strings.Split(f, "\n")) < 8 {
		t.Error("column image too small")
	}
}

func TestMissingImagePlaceholder(t *testing.T) {
	d := imageDeck(t, "![the architecture](nope/missing.png)\n\n![](nope/other.jpg)\n")
	r := New(d)
	f := r.Frame(0, 0, 80, 24)
	checkFrame(t, f, 80, 24)
	plain := ansi.Strip(f)
	if !strings.Contains(plain, "│ [image: the architecture] │") || !strings.Contains(plain, "[image: other.jpg]") {
		t.Errorf("missing-image boxes not drawn:\n%s", plain)
	}
	if rows, _ := r.Overflow(0, 80, 24); rows != 0 {
		t.Errorf("boxes overflow by %d", rows)
	}
	// Narrow widths degrade to a single truncated line, never a panic.
	checkFrame(t, r.Frame(0, 0, 12, 8), 12, 8)
}

func TestImageWidthPercent(t *testing.T) {
	d := imageDeck(t, "![image:width:50%](assets/terminal.png)\n")
	r := New(d)
	for _, l := range strings.Split(ansi.Strip(r.Frame(0, 0, 100, 30)), "\n") {
		if n := strings.Count(l, "▀") + strings.Count(l, "▄"); n > 46 {
			t.Errorf("image wider than 50%% of 92 cells: %d", n)
		}
	}
}

func TestKittyModeEmitsPlaceholders(t *testing.T) {
	old := Images
	defer func() { Images = old }()
	Images = ImageConfig{Mode: img.Kitty, Kitty: img.NewTransmitter()}
	d := imageDeck(t, "![logo](assets/terminal.png)\n")
	r := New(d)
	f := r.Frame(0, 0, 80, 24)
	checkFrame(t, f, 80, 24)
	if !strings.ContainsRune(f, '\U0010EEEE') {
		t.Error("no placeholder cells")
	}
	if up := Images.Kitty.Flush(); strings.Count(up, "a=T") != 1 {
		t.Errorf("want one upload, got %d", strings.Count(up, "a=T"))
	}
	// Placeholder geometry matches half-block geometry, so -check holds for kitty.
	Images = old
	blocks := New(d).Frame(0, 0, 80, 24)
	if halfBlockRows(strings.Split(blocks, "\n")) != countPlaceholderRows(f) {
		t.Error("kitty and half-block layouts differ")
	}
}

func countPlaceholderRows(f string) int {
	n := 0
	for _, l := range strings.Split(f, "\n") {
		if strings.ContainsRune(l, '\U0010EEEE') {
			n++
		}
	}
	return n
}
