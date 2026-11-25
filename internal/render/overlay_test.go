package render

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"tuipresent/internal/img"
)

func TestOverlayModesKeepGeometry(t *testing.T) {
	old := Images
	defer func() { Images = old }()
	src := "Logo\n===\n\n![logo](assets/terminal.png)\n\nCaption.\n"
	for _, sz := range [][2]int{{100, 30}, {80, 24}} {
		Images = old
		blocks := New(imageDeck(t, src)).Frame(0, 0, sz[0], sz[1])
		for _, mode := range []img.Mode{img.Sixel, img.ITerm2} {
			ov := img.NewOverlay(mode)
			Images = ImageConfig{Mode: mode, Overlay: ov}
			f := New(imageDeck(t, src)).Frame(0, 0, sz[0], sz[1])
			checkFrame(t, f, sz[0], sz[1])
			if countPlaceholderRows(f) != halfBlockRows(strings.Split(blocks, "\n")) {
				t.Errorf("%s %v: layout differs from half-blocks", mode, sz)
			}
			clean, ps := ov.Extract(f)
			if len(ps) != 1 || strings.ContainsRune(clean, '\U0010EEEE') {
				t.Fatalf("%s %v: placements %v", mode, sz, ps)
			}
			checkFrame(t, clean, sz[0], sz[1])
			// The image's cell box in the clean frame is blank, and sits where
			// the half-block image does.
			p := ps[0]
			rows := strings.Split(ansi.Strip(clean), "\n")
			for y := p.Row; y < p.Row+p.Rows; y++ {
				if seg := []rune(rows[y]); strings.TrimSpace(string(seg[p.Col:p.Col+p.Cols])) != "" {
					t.Errorf("%s %v: row %d under the image is not blank", mode, sz, y)
				}
			}
			bl := strings.Split(ansi.Strip(blocks), "\n")
			if !strings.ContainsAny(bl[p.Row], "▀▄") || !strings.ContainsAny(bl[p.Row+p.Rows-1], "▀▄") ||
				strings.ContainsAny(bl[p.Row-1], "▀▄") || strings.ContainsAny(bl[p.Row+p.Rows], "▀▄") {
				t.Errorf("%s %v: placement rows %d+%d do not match the half-block image", mode, sz, p.Row, p.Rows)
			}
			if ov.Draw(ps) == "" {
				t.Error("nothing to draw")
			}
		}
	}
}

func TestOverlayStandInsSurvivePaint(t *testing.T) {
	old := Images
	defer func() { Images = old }()
	ov := img.NewOverlay(img.Sixel)
	Images = ImageConfig{Mode: img.Sixel, Overlay: ov}
	d := imageDeck(t, "---\ntheme:\n  name: dark\n  background: paint\n---\n\n![logo](assets/terminal.png)\n")
	f := New(d).Frame(0, 0, 80, 24)
	if _, ps := ov.Extract(f); len(ps) != 1 {
		t.Errorf("painted frame: placements %v", ps)
	}
}
