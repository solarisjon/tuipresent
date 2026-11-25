package render

import (
	"image/color"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"tuipresent/internal/transition"
)

func headLen(t *testing.T, src string) int {
	t.Helper()
	r := renderDeck(t, src)
	head, _ := r.head(0, r.deck.Slides[0], 72)
	return len(head)
}

func TestBigTitleRowCost(t *testing.T) {
	for _, tc := range []struct {
		directives string
		want       int
	}{
		{"", 4},
		{"<!-- title_size: 1 -->\n", 4},
		{"<!-- title_size: 2 -->\n", 6},
		{"<!-- title_size: 3 -->\n", 8},
		{"<!-- title_size: 2 -->\n<!-- subtitle: s -->\n", 7},
		{"<!-- title_size: 2 -->\n<!-- title_style: frame -->\n", 7},
		{"<!-- title_size: 3 -->\n<!-- title_style: frame -->\n<!-- subtitle: s -->\n", 10},
		{"<!-- title_size: 3 -->\n<!-- title_style: curly -->\n", 8},
	} {
		if got := headLen(t, "Hi\n===\n"+tc.directives+"body\n"); got != tc.want {
			t.Errorf("%q: head rows = %d, want %d", tc.directives, got, tc.want)
		}
	}
	for size, rows := range map[int]int{1: 1, 2: 3, 3: 5} {
		if TitleRows(size) != rows {
			t.Errorf("TitleRows(%d) = %d, want %d", size, TitleRows(size), rows)
		}
	}
}

func TestBigTitleCurlyIsRuleEvenWhereSupported(t *testing.T) {
	defer func(c bool) { Curly = c }(Curly)
	Curly = true
	r := renderDeck(t, "Hi\n===\n<!-- title_size: 2 -->\n<!-- title_style: curly -->\nbody\n")
	f := r.Frame(0, 0, 80, 24)
	if !strings.Contains(ansi.Strip(f), "━━━") || strings.Contains(f, "4:3") {
		t.Errorf("curly big title should draw the rule, no curly underline:\n%s", f)
	}
}

func TestBigTitleOverflowCountsRows(t *testing.T) {
	body := strings.Repeat("line\n\n", 14)
	rows := func(size string) int {
		r := renderDeck(t, "Hi\n===\n<!-- title_size: "+size+" -->\n"+body)
		n, _ := r.Overflow(0, 80, 24)
		return n
	}
	r1, r2, r3 := rows("1"), rows("2"), rows("3")
	if r2-r1 != 2 || r3-r1 != 4 {
		t.Errorf("overflow rows by size = %d, %d, %d; want +2 and +4 over size 1", r1, r2, r3)
	}
}

func TestBigTitleFramesAreExactWidth(t *testing.T) {
	for _, src := range []string{
		"Hi there\n===\n<!-- title_size: 2 -->\n<!-- subtitle: 日本語の字幕 -->\nbody **x**\n",
		"Hi\n===\n<!-- title_size: 3 -->\n<!-- title_style: frame -->\n<!-- subtitle: wide 日本 -->\nbody\n",
		"Hi\n===\n<!-- title_size: 2 -->\n<!-- jump_to_middle -->\nbody\n",
		"`code` ==hi==\n===\n<!-- title_size: 2 -->\nbody\n",
	} {
		r := renderDeck(t, src)
		f := r.Frame(0, 0, 80, 24)
		lines := strings.Split(f, "\n")
		if len(lines) != 24 {
			t.Errorf("%d lines", len(lines))
		}
		for i, l := range lines {
			if w := ansi.StringWidth(l); w != 80 {
				t.Errorf("%q line %d width %d: %q", src, i, w, l)
			}
		}
		if strings.Contains(f, "\x1b]") {
			t.Errorf("Frame must not contain OSC sequences")
		}
	}
}

func TestBigTitleDegradesWhenTooWide(t *testing.T) {
	r := renderDeck(t, "A title much too wide for size three\n===\n<!-- title_size: 3 -->\nbody\n\n<!-- end_slide -->\nOK\n===\n<!-- title_size: 3 -->\nbody\n")
	w := r.TitleSizeWarnings(80)
	if len(w) != 1 || !strings.HasPrefix(w[0], "slide 1: title_size 3 does not fit") || !strings.Contains(w[0], "using the normal title") {
		t.Errorf("warnings = %q", w)
	}
	if got := len(strings.Split(ansi.Strip(r.Frame(0, 0, 80, 24)), "\n")); got != 24 {
		t.Errorf("lines = %d", got)
	}
	head, _ := r.head(0, r.deck.Slides[0], 72)
	if len(head) != 4 {
		t.Errorf("degraded head has %d rows, want 4", len(head))
	}
	if _, ok := r.BigTitleAt(0, 0, 80, 24); ok {
		t.Error("degraded title must have no overlay")
	}
}

func TestBigTitleFrameBoxCountsTowardWidth(t *testing.T) {
	// "ABCDEF" at size 2 is 35 cells; a frame adds 4.
	r := renderDeck(t, "ABCDEF\n===\n<!-- title_size: 2 -->\n<!-- title_style: frame -->\nbody\n")
	_, w := r.head(0, r.deck.Slides[0], 72)
	if w != 39 {
		t.Errorf("frame width = %d, want 39", w)
	}
	// contentWidth(46) = 38 < 39
	if got := r.TitleSizeWarnings(46); len(got) != 1 {
		t.Errorf("box wider than content should warn: %q", got)
	}
	if got := r.TitleSizeWarnings(50); len(got) != 0 {
		t.Errorf("box fits 42 columns: %q", got)
	}
}

func TestBigTitleWideCharactersDegrade(t *testing.T) {
	r := renderDeck(t, "日本語\n===\n<!-- title_size: 2 -->\nbody\n\n<!-- end_slide -->\nCafé\n===\n<!-- title_size: 2 -->\nbody\n")
	w := r.TitleSizeWarnings(100)
	if len(w) != 2 || !strings.Contains(w[0], "no block letter") {
		t.Errorf("warnings = %q", w)
	}
	for i := range 2 {
		for _, l := range strings.Split(r.Frame(i, 0, 100, 30), "\n") {
			if ansi.StringWidth(l) != 100 {
				t.Errorf("slide %d line width %d", i, ansi.StringWidth(l))
			}
		}
		if rows, cols := r.Overflow(i, 100, 30); rows != 0 || cols != 0 {
			t.Errorf("overflow %d,%d", rows, cols)
		}
	}
}

func TestBigTitleGradientAcrossBlock(t *testing.T) {
	r := renderDeck(t, "HOME\n===\n<!-- title_size: 3 -->\nbody\n")
	lines := strings.Split(r.Frame(0, 0, 100, 30), "\n")
	color := func(row string, last bool) string {
		i := strings.Index(row, "\x1b[38;")
		if last {
			i = strings.LastIndex(row, "\x1b[38;")
		}
		c := row[i:]
		return c[:strings.Index(c, "m")]
	}
	first, last := color(lines[3], false), color(lines[3], true)
	if first == last {
		t.Errorf("no gradient across the block: %q", first)
	}
	if !strings.Contains(lines[5], first+"m") {
		t.Errorf("rows do not share colors")
	}
}

func TestBigTitleCenteredBlockStaysAligned(t *testing.T) {
	r := renderDeck(t, "H\n===\n<!-- title_size: 3 -->\n<!-- jump_to_middle -->\nbody\n")
	var starts []int
	for _, l := range strings.Split(ansi.Strip(r.Frame(0, 0, 80, 24)), "\n") {
		if strings.Contains(l, "██") {
			starts = append(starts, len(l)-len(strings.TrimLeft(l, " ")))
		}
	}
	if len(starts) != 5 {
		t.Fatalf("block rows = %d", len(starts))
	}
	for _, s := range starts {
		if s != starts[0] {
			t.Errorf("block rows start at different columns: %v", starts)
		}
	}
}

func TestBigTitleOverlay(t *testing.T) {
	r := renderDeck(t, "Hi 9\n===\n<!-- title_size: 2 -->\nbody\n\n<!-- end_slide -->\nPlain\n===\nbody\n")
	bt, ok := r.BigTitleAt(0, 0, 80, 24)
	if !ok {
		t.Fatal("no overlay")
	}
	// Title starts on row 3 (blank row, blank row above the head), column 5.
	if !strings.Contains(bt.Draw, "\x1b[3;5H") {
		t.Errorf("overlay not positioned at row 3, col 5: %q", bt.Draw)
	}
	for _, ch := range "HI9" {
		if !strings.Contains(bt.Draw, "\x1b]66;s=3;"+string(ch)+"\x1b\\") {
			t.Errorf("missing scaled %c in %q", ch, bt.Draw)
		}
	}
	if !strings.HasPrefix(bt.Draw, bt.Erase) || strings.Contains(bt.Erase, "\x1b]66") {
		t.Errorf("erase must be a blank-only prefix of draw")
	}
	if _, ok := r.BigTitleAt(1, 0, 80, 24); ok {
		t.Error("plain slide has no overlay")
	}
}

// The scaled title takes scale cells per rune and the block letters at least
// as many, so the overlay never reaches past the cells -check measured.
func TestBigTitleOverlayNeverWiderThanBlock(t *testing.T) {
	for size := 2; size <= 3; size++ {
		text, colors := []rune("HELLO, WORLD!"), make([]string, 13)
		_, bw, _, ok := blockRows(text, colors, size)
		if !ok {
			t.Fatal("no glyphs")
		}
		if scaled := len(text) * TitleRows(size); scaled > bw {
			t.Errorf("size %d: scaled %d wider than block %d", size, scaled, bw)
		}
	}
}

func TestBigTitleOverlayFollowsCenteringAndFrame(t *testing.T) {
	r := renderDeck(t, "H\n===\n<!-- title_size: 2 -->\n<!-- title_style: frame -->\n<!-- jump_to_middle -->\nbody\n")
	f := strings.Split(ansi.Strip(r.Frame(0, 0, 80, 24)), "\n")
	bt, _ := r.BigTitleAt(0, 0, 80, 24)
	for y, l := range f {
		if strings.ContainsAny(l, "█▀▄") {
			col := strings.IndexAny(l, "█▀▄")
			want := ansi.CursorPosition(ansi.StringWidth(l[:col])+1, y+1)
			if !strings.Contains(bt.Draw, want) {
				t.Errorf("overlay should start at %q (row %d col %d)", want, y, col)
			}
			return
		}
	}
	t.Fatal("no block rows")
}

func TestBigTitleTransitionsKeepFramesIntact(t *testing.T) {
	r := renderDeck(t, "Hi\n===\n<!-- title_size: 2 -->\nbody\n<!-- pause -->\nmore\n\n<!-- end_slide -->\nBig\n===\n<!-- title_size: 3 -->\nbody\n")
	a0, a1, b := r.Frame(0, 0, 60, 20), r.Frame(0, 1, 60, 20), r.Frame(1, 0, 60, 20)
	for _, k := range []transition.Kind{transition.Slide, transition.Wipe, transition.Dissolve, transition.Cover} {
		for _, p := range []float64{0, 0.25, 0.5, 0.9, 1} {
			out := transition.Composite(k, a0, b, 60, p, false)
			for i, l := range strings.Split(out, "\n") {
				if ansi.StringWidth(l) != 60 {
					t.Fatalf("%v p=%v line %d width %d", k, p, i, ansi.StringWidth(l))
				}
			}
			if strings.Contains(out, "\x1b]66") {
				t.Fatal("OSC 66 in a composited frame")
			}
		}
	}
	// The pause reveal fades only the body; the title rows pass through byte for byte.
	out := transition.FadeIn(a0, a1, 0.5, color.Black, color.White)
	la, lo := strings.Split(a1, "\n"), strings.Split(out, "\n")
	for i := 1; i < 7; i++ {
		if la[i] != lo[i] {
			t.Errorf("title row %d altered by the fade", i)
		}
	}
}

func TestDetectTextSizing(t *testing.T) {
	env := func(kv ...string) func(string) string {
		return func(k string) string {
			for i := 0; i+1 < len(kv); i += 2 {
				if kv[i] == k {
					return kv[i+1]
				}
			}
			return ""
		}
	}
	for name, tc := range map[string]struct {
		env  func(string) string
		want bool
	}{
		"kitty":         {env("TERM", "xterm-kitty"), true},
		"kitty program": {env("TERM_PROGRAM", "kitty", "TERM", "xterm-256color"), true},
		"kitty window":  {env("KITTY_WINDOW_ID", "3", "TERM", "xterm-256color"), true},
		"kitty in tmux": {env("TERM", "xterm-kitty", "TMUX", "/tmp/x"), false},
		"screen":        {env("TERM", "screen-256color", "KITTY_WINDOW_ID", "3"), false},
		"ghostty":       {env("TERM_PROGRAM", "ghostty", "TERM", "xterm-ghostty"), false},
		"plain xterm":   {env("TERM", "xterm-256color"), false},
		"forced on":     {env("TUIPRESENT_TEXT_SIZING", "on", "TERM", "xterm-ghostty"), true},
		"forced off":    {env("TUIPRESENT_TEXT_SIZING", "off", "TERM", "xterm-kitty"), false},
		"empty":         {env(), false},
	} {
		if got := DetectTextSizing(tc.env); got != tc.want {
			t.Errorf("%s: got %v, want %v", name, got, tc.want)
		}
	}
}
