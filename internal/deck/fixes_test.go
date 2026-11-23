package deck

import (
	"reflect"
	"strings"
	"testing"
)

func TestEmptyFrontMatter(t *testing.T) {
	for _, src := range []string{"---\n---\nBody\n===\n", "---\n---\n\nBody\n===\n\ntext\n"} {
		d := mustParse(t, src)
		if len(d.Slides) != 1 || d.Slides[0].Title != "Body" {
			t.Errorf("%q: slides %d, title %q; want 1 slide titled Body", src, len(d.Slides), d.Slides[0].Title)
		}
	}
	if _, err := Parse("---\n---"); err == nil {
		t.Error("front matter alone should leave no slides")
	}
}

func TestUnclosedNoteStopsBeforeEndSlide(t *testing.T) {
	src := "One\n===\n\n<!-- speaker_note: forgot to close\nmore note\n<!-- end_slide -->\nTwo\n===\n"
	d := mustParse(t, src)
	if len(d.Slides) != 2 {
		t.Fatalf("slides = %d, want 2 (end_slide must not be swallowed)", len(d.Slides))
	}
	if d.Slides[1].Title != "Two" {
		t.Errorf("slide 2 title = %q", d.Slides[1].Title)
	}
	if want := []string{"forgot to close\nmore note"}; !reflect.DeepEqual(d.Slides[0].Notes, want) {
		t.Errorf("notes = %q, want %q", d.Slides[0].Notes, want)
	}
	if len(d.Warnings) != 1 || !strings.HasPrefix(d.Warnings[0], "line 4: speaker_note has no closing -->") {
		t.Errorf("warnings = %q", d.Warnings)
	}
}

func TestUnclosedNoteAtEndWarns(t *testing.T) {
	d := mustParse(t, "---\ntitle: T\n---\nA\n===\n<!-- speaker_note:\nrest\n")
	if len(d.Warnings) != 1 || !strings.HasPrefix(d.Warnings[0], "line 6: ") {
		t.Errorf("warnings = %q", d.Warnings)
	}
	if !reflect.DeepEqual(d.Slides[1].Notes, []string{"rest"}) {
		t.Errorf("notes = %q", d.Slides[1].Notes)
	}
}

func TestUnclosedNoteStopsBeforeDirective(t *testing.T) {
	src := "One\n===\n\nBefore\n\n<!-- speaker_note: forgot to close\nmore note\n<!-- pause -->\nAfter\n"
	d := mustParse(t, src)
	s := d.Slides[0]
	if s.Steps != 2 {
		t.Errorf("steps = %d, want 2 (pause must not be swallowed)", s.Steps)
	}
	if want := []string{"forgot to close\nmore note"}; !reflect.DeepEqual(s.Notes, want) {
		t.Errorf("notes = %q, want %q", s.Notes, want)
	}
	if len(d.Warnings) != 1 || !strings.HasPrefix(d.Warnings[0], "line 6: speaker_note has no closing -->") {
		t.Errorf("warnings = %q", d.Warnings)
	}
}

func TestClosedNoteWithArrowMidLineIsNotCut(t *testing.T) {
	d := mustParse(t, "A\n<!-- speaker_note:\nsay this --> then\n<!-- pause -->\nB\n")
	if d.Slides[0].Steps != 2 || len(d.Warnings) != 0 {
		t.Errorf("steps = %d, warnings = %q", d.Slides[0].Steps, d.Warnings)
	}
}

func TestClosedNoteHasNoWarning(t *testing.T) {
	d := mustParse(t, "A\n<!-- speaker_note:\nx\n-->\n")
	if len(d.Warnings) != 0 {
		t.Errorf("warnings = %q", d.Warnings)
	}
}

func TestEmptySlideConsistency(t *testing.T) {
	src := "a\n<!-- end_slide -->\n" +
		"<!-- speaker_note: pause and talk -->\n<!-- end_slide -->\n" + // notes only: kept
		"<!-- column_layout: [1, 1] -->\n<!-- end_slide -->\n" + // empty layout: dropped
		"<!-- column_layout: [1, 1] -->\n<!-- column: 0 -->\n<!-- speaker_note: talk over nothing -->\n<!-- end_slide -->\n" + // layout + note: kept
		"<!-- jump_to_middle -->\n<!-- transition: none -->\n<!-- end_slide -->\n" + // directives only: dropped
		"<!-- column_layout: [1, 1] -->\n<!-- column: 1 -->\nb\n" // layout with content: kept
	d := mustParse(t, src)
	var got []string
	for _, s := range d.Slides {
		got = append(got, strings.Join(s.Notes, "|")+":"+strings.TrimSpace(allText(s.Blocks)))
	}
	want := []string{":a", "pause and talk:", "talk over nothing:", ":b"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("slides = %q, want %q", got, want)
	}
	if _, err := Parse("<!-- column_layout: [1] -->\n"); err == nil {
		t.Error("a deck of only an empty layout should have no slides")
	}
}

func TestUnknownTransitionWarns(t *testing.T) {
	src := "---\ntransition: wpie\n---\nA\n<!-- end_slide -->\n<!-- transition: Dissolve -->\nB\n<!-- end_slide -->\nC\n<!-- transition: fade -->\n"
	d := mustParse(t, src)
	want := []string{
		`front matter: unknown transition "wpie", using slide (valid: slide, wipe, cover, dissolve, none)`,
		`line 10: unknown transition "fade", using slide (valid: slide, wipe, cover, dissolve, none)`,
	}
	if !reflect.DeepEqual(d.Warnings, want) {
		t.Errorf("warnings = %q, want %q", d.Warnings, want)
	}
	if d.Slides[2].Transition != "fade" {
		t.Errorf("the directive value is kept as written: %q", d.Slides[2].Transition)
	}
}

func TestFenceRules(t *testing.T) {
	if m, info, ok := FenceOpen("  ````markdown x"); !ok || m != "````" || info != "markdown x" {
		t.Errorf("FenceOpen = %q %q %v", m, info, ok)
	}
	for _, tc := range []struct {
		marker, line string
		want         bool
	}{
		{"```", "```", true},
		{"```", "  `````  ", true},
		{"````", "```", false},
		{"```", "~~~", false},
		{"```", "```go", false},
		{"~~~", "~~~~", true},
		{"```", "text", false},
	} {
		if got := ClosesFence(tc.marker, tc.line); got != tc.want {
			t.Errorf("ClosesFence(%q, %q) = %v", tc.marker, tc.line, got)
		}
	}
}
