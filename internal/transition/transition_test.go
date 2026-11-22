package transition

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// frame builds an h-line frame, each line exactly w cells, mixing ANSI color
// codes, box-drawing runes, and plain text.
func frame(w, h int, fill string, color int) string {
	lines := make([]string, h)
	for i := range lines {
		var sb strings.Builder
		cells := 0
		for cells < w {
			switch (i + cells) % 3 {
			case 0:
				fmt.Fprintf(&sb, "\x1b[38;5;%dm━\x1b[0m", color)
			case 1:
				fmt.Fprintf(&sb, "\x1b[1;38;2;122;162;247m%s\x1b[0m", fill)
			default:
				sb.WriteString("─")
			}
			cells++
		}
		lines[i] = sb.String()
	}
	return strings.Join(lines, "\n")
}

var kinds = []Kind{None, Slide, Wipe, Dissolve, Cover}

func TestCompositeDimensions(t *testing.T) {
	for _, size := range [][2]int{{1, 1}, {7, 3}, {40, 15}, {101, 31}} {
		w, h := size[0], size[1]
		from, to := frame(w, h, "a", 1), frame(w, h, "b", 2)
		for _, line := range strings.Split(from, "\n") {
			if got := ansi.StringWidth(line); got != w {
				t.Fatalf("fixture width %d, want %d", got, w)
			}
		}
		for _, k := range kinds {
			for _, rev := range []bool{false, true} {
				for _, tt := range []float64{0, 0.25, 0.5, 0.99, 1} {
					name := fmt.Sprintf("%s/%dx%d/rev=%v/t=%v", k, w, h, rev, tt)
					t.Run(name, func(t *testing.T) {
						out := Composite(k, from, to, w, tt, rev)
						lines := strings.Split(out, "\n")
						if len(lines) != h {
							t.Fatalf("got %d lines, want %d", len(lines), h)
						}
						for i, l := range lines {
							if got := ansi.StringWidth(l); got != w {
								t.Errorf("line %d width %d, want %d: %q", i, got, w, l)
							}
						}
					})
				}
			}
		}
	}
}

func TestCompositeEndpoints(t *testing.T) {
	w, h := 30, 10
	from, to := frame(w, h, "a", 1), frame(w, h, "b", 2)
	for _, k := range kinds {
		for _, rev := range []bool{false, true} {
			if got := Composite(k, from, to, w, 1, rev); got != to {
				t.Errorf("%s rev=%v t=1 did not return to", k, rev)
			}
			if got := Composite(k, from, to, w, 1.5, rev); got != to {
				t.Errorf("%s rev=%v t=1.5 did not return to", k, rev)
			}
		}
	}
	for _, tt := range []float64{0, 0.3, 0.99} {
		if got := Composite(None, from, to, w, tt, false); got != to {
			t.Errorf("None t=%v did not return to", tt)
		}
	}
	// At t=0 every animated kind should still show the source frame's content.
	for _, k := range []Kind{Slide, Wipe, Dissolve, Cover} {
		for _, rev := range []bool{false, true} {
			got := ansi.Strip(Composite(k, from, to, w, 0, rev))
			if got != ansi.Strip(from) {
				t.Errorf("%s rev=%v t=0 does not show from", k, rev)
			}
		}
	}
}

func TestCompositeProgress(t *testing.T) {
	// Plain frames so we can count which cells come from "to".
	w, h := 20, 10
	from := strings.TrimSuffix(strings.Repeat(strings.Repeat("a", w)+"\n", h), "\n")
	to := strings.TrimSuffix(strings.Repeat(strings.Repeat("b", w)+"\n", h), "\n")
	for _, k := range []Kind{Slide, Wipe, Dissolve, Cover} {
		for _, rev := range []bool{false, true} {
			got := strings.Count(Composite(k, from, to, w, 0.5, rev), "b")
			if want := w * h / 2; got != want {
				t.Errorf("%s rev=%v t=0.5: %d cells from to, want %d", k, rev, got, want)
			}
		}
	}
}

func TestParse(t *testing.T) {
	tests := map[string]Kind{
		"slide":    Slide,
		"wipe":     Wipe,
		"cover":    Cover,
		"dissolve": Dissolve,
		"none":     None,
		" WIPE ":   Wipe,
		"Dissolve": Dissolve,
		"":         Slide,
		"fade":     Slide,
		"bogus":    Slide,
	}
	for in, want := range tests {
		if got := Parse(in); got != want {
			t.Errorf("Parse(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestEaseOutCubic(t *testing.T) {
	if EaseOutCubic(0) != 0 || EaseOutCubic(1) != 1 {
		t.Errorf("endpoints: %v, %v", EaseOutCubic(0), EaseOutCubic(1))
	}
	prev := 0.0
	for i := 1; i <= 100; i++ {
		v := EaseOutCubic(float64(i) / 100)
		if v < prev {
			t.Fatalf("not monotonic at %d", i)
		}
		prev = v
	}
	if EaseOutCubic(0.5) <= 0.5 {
		t.Error("ease-out should be ahead of linear at midpoint")
	}
}
