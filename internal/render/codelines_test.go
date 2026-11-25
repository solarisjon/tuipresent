package render

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/solarisjon/tuipresent/internal/deck"
)

const hlDeck = "# T\n\n```go {1-2|4|all}\nalpha := 1\nbeta := 2\n\ngamma := 3\n```\n\nafter\n"

// codeLine returns the frame line containing text.
func codeLine(t *testing.T, frame, text string) string {
	t.Helper()
	for _, l := range strings.Split(frame, "\n") {
		if strings.Contains(ansi.Strip(l), text) {
			return l
		}
	}
	t.Fatalf("no line with %q in:\n%s", text, ansi.Strip(frame))
	return ""
}

func TestHighlightStepsDimOtherLines(t *testing.T) {
	d, err := deck.Parse(hlDeck)
	if err != nil {
		t.Fatal(err)
	}
	if d.Slides[0].Steps != 3 {
		t.Fatalf("Steps = %d, want 3", d.Slides[0].Steps)
	}
	r := New(d)
	f0, f1, f2 := r.Frame(0, 0, 80, 24), r.Frame(0, 1, 80, 24), r.Frame(0, 2, 80, 24)
	for _, f := range []string{f0, f1, f2} {
		checkFrame(t, f, 80, 24)
	}
	// Content after the block waits for the last group, like after a pause.
	if strings.Contains(ansi.Strip(f1), "after") || !strings.Contains(ansi.Strip(f2), "after") {
		t.Error("content after the code block should appear at the last step only")
	}
	// Dimmed lines differ in escape sequences only.
	a0, b0, c0 := codeLine(t, f0, "alpha"), codeLine(t, f0, "beta"), codeLine(t, f0, "gamma")
	a1, c1 := codeLine(t, f1, "alpha"), codeLine(t, f1, "gamma")
	a2, c2 := codeLine(t, f2, "alpha"), codeLine(t, f2, "gamma")
	if a0 != codeLine(t, f2, "alpha") || a0 != a2 {
		t.Error("selected line should render like the fully lit one")
	}
	if a0 == a1 || b0 == codeLine(t, f1, "beta") {
		t.Error("lines outside group 2 should be dimmed at step 2")
	}
	if c0 == c2 || c1 != c2 {
		t.Error("gamma: dimmed at step 1, lit at steps 2 and 3")
	}
}

func TestHighlightStaticSingleGroup(t *testing.T) {
	d, _ := deck.Parse("```go {2}\none\ntwo\nthree\n```\n")
	if d.Slides[0].Steps != 1 {
		t.Fatalf("Steps = %d, want 1", d.Slides[0].Steps)
	}
	f := New(d).Frame(0, 0, 80, 24)
	if codeLine(t, f, "two") == codeLine(t, f, "one") {
		t.Error("expected one and two to be styled differently")
	}
	if strings.Count(ansi.Strip(f), "two") != 1 {
		t.Error("code should appear once")
	}
}

func TestHighlightWithExecAndFade(t *testing.T) {
	d, _ := deck.Parse("```bash {1|2} +exec\necho a\necho b\n```\n\n<!-- pause -->\n\nend\n")
	s := d.Slides[0]
	if s.Steps != 3 || s.Blocks[0].Kind != deck.Exec || s.Blocks[0].ExecID != 1 {
		t.Fatalf("steps=%d blocks=%+v", s.Steps, s.Blocks)
	}
	r := New(d)
	r.SetExecOutput(1, "a\nb\n")
	for st := range s.Steps {
		f := r.Frame(0, st, 80, 24)
		checkFrame(t, f, 80, 24)
		if !strings.Contains(ansi.Strip(f), "echo b") {
			t.Errorf("step %d: code missing", st)
		}
	}
	if strings.Contains(ansi.Strip(r.Frame(0, 1, 80, 24)), "end") {
		t.Error("pause content shown too early")
	}
}

func TestHighlightOnlyAffectsItsBlock(t *testing.T) {
	d, _ := deck.Parse("```go\nplain\n```\n\n```go {1}\nx\ny\n```\n")
	r := New(d)
	f := r.Frame(0, 0, 80, 24)
	if !strings.Contains(ansi.Strip(f), "plain") || !strings.Contains(ansi.Strip(f), "y") {
		t.Fatal(ansi.Strip(f))
	}
}

func TestHighlightBlankEdgeLinesAndTabs(t *testing.T) {
	d, _ := deck.Parse("```go {2|3}\n\n\tfoo\nbar\n\n```\n")
	r := New(d)
	for st := range 2 {
		checkFrame(t, r.Frame(0, st, 60, 20), 60, 20)
	}
	a, b := r.Frame(0, 0, 60, 20), r.Frame(0, 1, 60, 20)
	if codeLine(t, a, "foo") == codeLine(t, b, "foo") {
		t.Error("foo should change between steps")
	}
}

func TestFenceAround(t *testing.T) {
	if got := fenceAround("go", "x"); got != "```go\nx\n```" {
		t.Errorf("got %q", got)
	}
	if got := fenceAround("", "a ``` b"); !strings.HasPrefix(got, "````\n") {
		t.Errorf("got %q", got)
	}
}
