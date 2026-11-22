package render

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"tuipresent/internal/deck"
)

func checkFrame(t *testing.T, f string, w, h int) {
	t.Helper()
	lines := strings.Split(f, "\n")
	if len(lines) != h {
		t.Fatalf("got %d lines, want %d", len(lines), h)
	}
	for i, l := range lines {
		if got := ansi.StringWidth(l); got != w {
			t.Errorf("line %d width %d, want %d: %q", i, got, w, ansi.Strip(l))
		}
	}
}

func TestFrameDimensionsDemoDeck(t *testing.T) {
	d, err := deck.Load("../../cpe-staff-meeting-demo.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, size := range [][2]int{{40, 15}, {100, 30}, {220, 60}} {
		w, h := size[0], size[1]
		r := New(d)
		for i, s := range d.Slides {
			for st := range s.Steps {
				t.Run(fmt.Sprintf("%dx%d/slide%d/step%d", w, h, i+1, st+1), func(t *testing.T) {
					checkFrame(t, r.Frame(i, st, w, h), w, h)
				})
			}
		}
	}
}

func TestFrameWithExecOutput(t *testing.T) {
	d, err := deck.Parse("```bash +exec\necho hi\n```\n")
	if err != nil {
		t.Fatal(err)
	}
	r := New(d)
	before := r.Frame(0, 0, 80, 24)
	r.SetExecOutput(1, "hello from exec ━━━ with a long line that should be wrapped or clipped to fit the frame width\n")
	after := r.Frame(0, 0, 80, 24)
	checkFrame(t, after, 80, 24)
	if before == after || !strings.Contains(ansi.Strip(after), "hello from exec") {
		t.Error("exec output not rendered")
	}
}

func TestFit(t *testing.T) {
	tests := []struct {
		in string
		w  int
	}{
		{"", 5},
		{"abc", 5},
		{"abcdefgh", 5},
		{"\x1b[31m━━━━━━━━\x1b[0m", 4},
		{"\x1b[1mhi\x1b[0m", 10},
	}
	for _, tt := range tests {
		if got := ansi.StringWidth(Fit(tt.in, tt.w)); got != tt.w {
			t.Errorf("Fit(%q, %d) width = %d", tt.in, tt.w, got)
		}
	}
}

func TestTinyTerminalDoesNotPanic(t *testing.T) {
	d, err := deck.Parse("# hi\n")
	if err != nil {
		t.Fatal(err)
	}
	r := New(d)
	for h := 0; h <= 6; h++ {
		f := r.Frame(0, 0, 40, h)
		if h > 0 && len(strings.Split(f, "\n")) != h {
			t.Fatalf("h=%d: got %d lines", h, len(strings.Split(f, "\n")))
		}
	}
}

func TestTabsExpanded(t *testing.T) {
	d, err := deck.Parse("```go\nfunc f() {\n\treturn\n}\n```\n")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(New(d).Frame(0, 0, 80, 20), "\t") {
		t.Fatal("frame contains a raw tab")
	}
}

func plainFrame(t *testing.T, src string, w, h, step int) []string {
	t.Helper()
	d, err := deck.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(ansi.Strip(New(d).Frame(0, step, w, h)), "\n")
}

func TestWrappedListItemsHangIndent(t *testing.T) {
	lines := plainFrame(t, "* alpha beta gamma delta epsilon zeta eta theta iota kappa lambda mu\n", 40, 20, 0)
	for i, l := range lines {
		if strings.Contains(l, "• alpha") {
			bullet := strings.Index(l, "•")
			next := lines[i+1]
			if strings.TrimSpace(next) == "" || len(next)-len(strings.TrimLeft(next, " ")) != bullet+2 {
				t.Fatalf("continuation not aligned under item text:\n%q\n%q", l, next)
			}
			return
		}
	}
	t.Fatal("list item not found")
}

func TestListLikeCodeIsNotRewrapped(t *testing.T) {
	lines := plainFrame(t, "```markdown\n1. one\n<!-- pause -->\n```\n", 60, 20, 0)
	joined := strings.Join(lines, "\n")
	if strings.Contains(joined, "1. one <!--") {
		t.Fatalf("code lines were merged:\n%s", joined)
	}
}

func TestJumpToMiddleCentersStatement(t *testing.T) {
	lines := plainFrame(t, "<!-- jump_to_middle -->\n\nBig idea.\n", 80, 20, 0)
	for _, l := range lines {
		if i := strings.Index(l, "Big idea."); i >= 0 {
			if right := len(l) - i - len("Big idea."); abs(i-right) > 1 {
				t.Fatalf("not centered: left %d right %d", i, right)
			}
			return
		}
	}
	t.Fatal("statement not found")
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func TestOverflowCatchesWideColumnCode(t *testing.T) {
	src := "<!-- column_layout: [1, 1] -->\n<!-- column: 0 -->\n```text\n" + strings.Repeat("x", 50) + "\n```\n"
	d, err := deck.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	if _, cols := New(d).Overflow(0, 80, 24); cols == 0 {
		t.Fatal("expected column code overflow to be reported")
	}
}
