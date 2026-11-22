package notes

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"tuipresent/internal/deck"
	"tuipresent/internal/render"
)

func TestFrame(t *testing.T) {
	d, err := deck.Parse("---\ntitle: Talk\n---\n\nFirst\n===\n\nbody\n<!-- speaker_note: remember the **demo** -->\n<!-- pause -->\nmore\n" +
		"<!-- end_slide -->\n\nSecond\n===\n\nbody two\n")
	if err != nil {
		t.Fatal(err)
	}
	r := render.New(d)
	start := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	now := start.Add(time.Hour + 2*time.Minute + 3*time.Second)
	const w, h = 60, 12

	check := func(name, frame string, want ...string) {
		t.Helper()
		lines := strings.Split(frame, "\n")
		if len(lines) != h {
			t.Errorf("%s: %d lines, want %d", name, len(lines), h)
		}
		for i, l := range lines {
			if n := ansi.StringWidth(l); n != w {
				t.Errorf("%s: line %d is %d cells, want %d", name, i, n, w)
			}
		}
		plain := ansi.Strip(frame)
		for _, s := range want {
			if !strings.Contains(plain, s) {
				t.Errorf("%s: missing %q in:\n%s", name, s, plain)
			}
		}
	}

	check("notes", Frame(d, r, "deck.md", State{Slide: 1, Step: 1, Started: start}, true, now, w, h),
		"First", "slide 2/3", "step 2/2", "1:02:03", "remember the demo", "Next: Second")
	check("no notes", Frame(d, r, "deck.md", State{Slide: 2, Started: start}, true, now, w, h),
		"(no notes for this slide)", "Next: end of deck")
	check("cover", Frame(d, r, "deck.md", State{Slide: 0, Started: start}, true, now, w, h),
		"Talk", "Next: First")
	check("waiting", Frame(d, r, "deck.md", State{}, false, now, w, h),
		"Waiting for the presenter", "tuipresent deck.md")
	check("stale index clamps", Frame(d, r, "deck.md", State{Slide: 99, Step: 9, Started: start}, true, now, w, h),
		"slide 3/3", "step 1/1")
}

func TestFrameTruncatesLongNotes(t *testing.T) {
	d, err := deck.Parse("x\n<!-- speaker_note:\n" + strings.Repeat("- point\n", 40) + "-->\n")
	if err != nil {
		t.Fatal(err)
	}
	f := Frame(d, render.New(d), "d.md", State{}, true, time.Now(), 40, 10)
	if n := len(strings.Split(f, "\n")); n != 10 {
		t.Errorf("%d lines, want 10", n)
	}
	if !strings.Contains(ansi.Strip(f), "…") || !strings.Contains(ansi.Strip(f), "Next:") {
		t.Errorf("long notes not truncated above footer:\n%s", ansi.Strip(f))
	}
}
