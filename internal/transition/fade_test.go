package transition

import (
	"fmt"
	"image/color"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

var (
	fadeBg = color.RGBA{0x1a, 0x1b, 0x26, 0xff}
	fadeFg = color.RGBA{0xd0, 0xd0, 0xd0, 0xff}
)

// pad right-pads a styled line with plain spaces to w cells.
func pad(s string, w int) string { return s + strings.Repeat(" ", w-ansi.StringWidth(s)) }

// fadeFrames returns a step n-1 frame and a step n frame of width w: a
// two-column line where only the right column is new, a new 24-bit line, a
// new 256-color line with a background, and an untouched line.
func fadeFrames(w int) (from, to string) {
	title := pad("\x1b[1;38;2;122;162;247mTitle ━━\x1b[m", w)
	left := "\x1b[38;5;252m1. old\x1b[m      "
	same := pad("\x1b[38;5;244m─── footer ───\x1b[m", w)
	from = strings.Join([]string{
		title,
		pad(left, w),
		pad("", w),
		pad("", w),
		same,
	}, "\n")
	to = strings.Join([]string{
		title,
		pad(left+"\x1b[38;5;252m2. new\x1b[m", w),
		pad("\x1b[38;2;255;121;198mtruecolor 漢字\x1b[0m plain", w),
		pad("\x1b[38;5;39;48;5;236mindexed\x1b[m \x1b[4:3;31mcurly\x1b[m", w),
		same,
	}, "\n")
	return from, to
}

func TestFadeInGeometry(t *testing.T) {
	for _, w := range []int{30, 41, 80} {
		from, to := fadeFrames(w)
		for _, tt := range []float64{0, 0.5, 1} {
			t.Run(fmt.Sprintf("w=%d/t=%v", w, tt), func(t *testing.T) {
				lines := strings.Split(FadeIn(from, to, tt, fadeBg, fadeFg), "\n")
				if len(lines) != 5 {
					t.Fatalf("got %d lines, want 5", len(lines))
				}
				for i, l := range lines {
					if got := ansi.StringWidth(l); got != w {
						t.Errorf("line %d width %d, want %d: %q", i, got, w, l)
					}
				}
				if got, want := ansi.Strip(strings.Join(lines, "\n")), ansi.Strip(to); got != want {
					t.Errorf("text changed:\n%s\nwant\n%s", got, want)
				}
			})
		}
	}
}

func TestFadeInEndpoints(t *testing.T) {
	from, to := fadeFrames(40)
	for _, tt := range []float64{1, 1.5} {
		if got := FadeIn(from, to, tt, fadeBg, fadeFg); got != to {
			t.Errorf("t=%v did not return to", tt)
		}
	}
	if got := FadeIn(to, to, 0, fadeBg, fadeFg); got != to {
		t.Error("identical frames should be returned unchanged")
	}
}

func TestFadeInLeavesUnchangedLines(t *testing.T) {
	from, to := fadeFrames(40)
	want := strings.Split(to, "\n")
	for _, tt := range []float64{0, 0.5} {
		got := strings.Split(FadeIn(from, to, tt, fadeBg, fadeFg), "\n")
		for _, i := range []int{0, 4} {
			if got[i] != want[i] {
				t.Errorf("t=%v line %d modified: %q", tt, i, got[i])
			}
		}
	}
}

// styles returns each cell's style on line i of frame.
func styles(frame string, i int) []cell {
	c, _ := parseCells(ansi.NewParser(), strings.Split(frame, "\n")[i])
	return c
}

func near(t *testing.T, what string, got, want color.Color) {
	t.Helper()
	if got == nil || want == nil {
		t.Fatalf("%s: got %v, want %v", what, got, want)
	}
	gr, gg, gb, _ := got.RGBA()
	wr, wg, wb, _ := want.RGBA()
	d := func(a, b uint32) int { return max(int(a>>8)-int(b>>8), int(b>>8)-int(a>>8)) }
	if d(gr, wr) > 3 || d(gg, wg) > 3 || d(gb, wb) > 3 {
		t.Errorf("%s: got %v, want ~%v", what, got, want)
	}
}

func TestFadeInColors(t *testing.T) {
	from, to := fadeFrames(40)
	start := FadeIn(from, to, 0, fadeBg, fadeFg)
	mid := FadeIn(from, to, 0.5, fadeBg, fadeFg)
	late := FadeIn(from, to, 0.999, fadeBg, fadeFg)

	// Two-column line: the old left column keeps its 256-color style, the new
	// right column starts at the background.
	row := styles(start, 1)
	near(t, "old column", row[0].style.fg, ansi.IndexedColor(252))
	near(t, "new column t=0", row[12].style.fg, fadeBg)
	near(t, "new column t=.999", styles(late, 1)[12].style.fg, ansi.IndexedColor(252))

	// 24-bit foreground, including wide graphemes.
	pink := color.RGBA{255, 121, 198, 0xff}
	near(t, "truecolor t=0", styles(start, 2)[0].style.fg, fadeBg)
	near(t, "wide t=0", styles(start, 2)[10].style.fg, fadeBg)
	near(t, "truecolor t=.999", styles(late, 2)[0].style.fg, pink)
	m := styles(mid, 2)[0].style.fg
	r, _, _, _ := m.RGBA()
	if r>>8 <= 0x1a || r>>8 >= 255 {
		t.Errorf("truecolor t=.5 red %d not between bg and target", r>>8)
	}
	// Default-foreground text fades toward the fallback.
	plain := styles(late, 2)
	near(t, "default fg", plain[13].style.fg, fadeFg)

	// 256-color foreground and background, and 16-color with sub-parameter attrs.
	ix := styles(start, 3)
	near(t, "indexed fg t=0", ix[0].style.fg, fadeBg)
	near(t, "indexed bg t=0", ix[0].style.bg, fadeBg)
	lx := styles(late, 3)
	near(t, "indexed fg t=.999", lx[0].style.fg, ansi.IndexedColor(39))
	near(t, "indexed bg t=.999", lx[0].style.bg, ansi.IndexedColor(236))
	near(t, "basic fg t=.999", lx[8].style.fg, ansi.BasicColor(1))
	if lx[8].style.attrs != "4:3" {
		t.Errorf("curly underline attrs = %q, want 4:3", lx[8].style.attrs)
	}
}
