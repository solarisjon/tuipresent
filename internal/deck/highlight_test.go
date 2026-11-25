package deck

import (
	"reflect"
	"strings"
	"testing"
)

// lines renders a selection as the 1-based line numbers it picks.
func picked(sel []bool) []int {
	var out []int
	for i, on := range sel {
		if on {
			out = append(out, i+1)
		}
	}
	return out
}

func TestParseHighlights(t *testing.T) {
	cases := []struct {
		name, spec string
		n          int
		want       [][]int
		warns      int
	}{
		{"single", "2", 5, [][]int{{2}}, 0},
		{"range", "2-4", 5, [][]int{{2, 3, 4}}, 0},
		{"groups", "1-3|5", 5, [][]int{{1, 2, 3}, {5}}, 0},
		{"list", "1,3-4,5", 5, [][]int{{1, 3, 4, 5}}, 0},
		{"all", "1|all", 3, [][]int{{1}, {1, 2, 3}}, 0},
		{"ALL case", "All", 2, [][]int{{1, 2}}, 0},
		{"spaces", " 1 - 2 | 4 ", 5, [][]int{{1, 2}, {4}}, 0},
		{"single-line range", "3-3", 3, [][]int{{3}}, 0},
		{"overlap", "1-3,2-4", 5, [][]int{{1, 2, 3, 4}}, 0},
		{"clipped", "4-9", 5, [][]int{{4, 5}}, 1},
		{"past end", "1|9", 5, [][]int{{1}}, 1},
		{"past end item", "1,9", 5, [][]int{{1}}, 1},
		{"zero", "0|2", 5, [][]int{{2}}, 1},
		{"reversed", "4-2|1", 5, [][]int{{1}}, 1},
		{"junk", "a|1", 5, [][]int{{1}}, 1},
		{"negative", "-2", 5, nil, 1},
		{"open range", "2-", 5, nil, 1},
		{"empty spec", "", 5, nil, 1},
		{"empty group", "1||3", 5, [][]int{{1}, {3}}, 1},
		{"trailing bar", "1|", 5, [][]int{{1}}, 1},
		{"only commas", ",", 5, nil, 1},
		{"empty block", "1", 0, nil, 1},
		{"huge number", "99999999999999999999", 3, nil, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			groups, warns := ParseHighlights(c.spec, c.n)
			var got [][]int
			for _, g := range groups {
				if len(g) != c.n {
					t.Fatalf("group length %d, want %d", len(g), c.n)
				}
				got = append(got, picked(g))
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("groups = %v, want %v", got, c.want)
			}
			if len(warns) != c.warns {
				t.Errorf("warnings = %q, want %d", warns, c.warns)
			}
		})
	}
}

func TestSplitHighlight(t *testing.T) {
	cases := []struct {
		info, rest, spec string
		found            bool
	}{
		{"rust {1-3|5}", "rust", "1-3|5", true},
		{"rust {1-3|5} +exec", "rust +exec", "1-3|5", true},
		{"+exec rust{2}", "+exec rust", "2", true},
		{"{1}", "", "1", true},
		{"{}", "", "", true},
		{"rust { 1 | 2 }", "rust", " 1 | 2 ", true},
		{"rust {1-3", "rust {1-3", "", false},
		{"rust", "rust", "", false},
		{"rust {1} {2}", "rust", "1", true},
	}
	for _, c := range cases {
		rest, spec, found := splitHighlight(c.info)
		if strings.Join(strings.Fields(rest), " ") != c.rest || spec != c.spec || found != c.found {
			t.Errorf("splitHighlight(%q) = %q, %q, %v", c.info, rest, spec, found)
		}
	}
}

func TestFenceHighlightBlock(t *testing.T) {
	d, err := Parse("intro\n\n```rust {1-2|3}\na\nb\nc\n```\n\noutro\n")
	if err != nil {
		t.Fatal(err)
	}
	s := d.Slides[0]
	if s.Steps != 2 {
		t.Fatalf("Steps = %d, want 2", s.Steps)
	}
	if len(s.Blocks) != 3 || s.Blocks[0].Kind != Markdown || s.Blocks[1].Kind != Code || s.Blocks[2].Kind != Markdown {
		t.Fatalf("blocks = %+v", s.Blocks)
	}
	c := s.Blocks[1]
	if c.Lang != "rust" || c.Text != "a\nb\nc" || c.Step != 0 || len(c.Highlights) != 2 {
		t.Errorf("code block = %+v", c)
	}
	if s.Blocks[2].Step != 1 {
		t.Errorf("content after the code is at step %d, want 1", s.Blocks[2].Step)
	}
	if len(d.Warnings) != 0 {
		t.Errorf("warnings = %q", d.Warnings)
	}
}

func TestFenceHighlightStepsCombineWithPauses(t *testing.T) {
	d, _ := Parse("a\n\n<!-- pause -->\n\n```go {1|2|3}\nx\ny\nz\n```\n\n<!-- pause -->\n\nb\n")
	s := d.Slides[0]
	if s.Steps != 5 {
		t.Fatalf("Steps = %d, want 5 (1 + 2 pauses + 2 extra groups)", s.Steps)
	}
	var steps []int
	for _, b := range s.Blocks {
		steps = append(steps, b.Step)
	}
	if !reflect.DeepEqual(steps, []int{0, 1, 4}) {
		t.Errorf("block steps = %v", steps)
	}
}

func TestFenceHighlightStaticNoExtraStep(t *testing.T) {
	d, _ := Parse("```go {2-3}\na\nb\nc\n```\n")
	if d.Slides[0].Steps != 1 || d.Slides[0].Blocks[0].Kind != Code || len(d.Slides[0].Blocks[0].Highlights) != 1 {
		t.Errorf("%+v", d.Slides[0])
	}
}

func TestFenceHighlightWithExec(t *testing.T) {
	for _, info := range []string{"bash {1|2} +exec", "+exec bash {1|2}", "bash +exec {1|2}", "bash{1|2} +exec"} {
		d, _ := Parse("```" + info + "\necho a\necho b\n```\n\n```sh +exec\ntrue\n```\n")
		s := d.Slides[0]
		if s.Steps != 2 || len(s.Blocks) != 2 {
			t.Fatalf("%q: Steps=%d blocks=%d", info, s.Steps, len(s.Blocks))
		}
		b := s.Blocks[0]
		if wantLang := map[bool]string{true: "", false: "bash"}[strings.HasPrefix(info, "+")]; b.Kind != Exec || b.Lang != wantLang || b.ExecID != 1 || len(b.Highlights) != 2 || b.Text != "echo a\necho b" {
			t.Errorf("%q: %+v", info, b)
		}
		if s.Blocks[1].ExecID != 2 || s.Blocks[1].Step != 1 {
			t.Errorf("%q: second exec = %+v", info, s.Blocks[1])
		}
	}
}

func TestFenceHighlightExecWithoutLanguage(t *testing.T) {
	d, _ := Parse("``` {1} +exec\nls\n```\n")
	if b := d.Slides[0].Blocks[0]; b.Kind != Exec || b.Lang != "" {
		t.Errorf("%+v", b)
	}
}

func TestFenceWithoutHighlightsStaysMarkdown(t *testing.T) {
	d, _ := Parse("```go\nx\n```\n")
	if b := d.Slides[0].Blocks; len(b) != 1 || b[0].Kind != Markdown || b[0].Highlights != nil {
		t.Errorf("%+v", b)
	}
}

func TestFenceHighlightWarnings(t *testing.T) {
	d, err := Parse("# s\n\nintro\n\n```go {1-9|x|}\na\nb\n```\n\n```go {7}\na\n```\n\n```go {1\na\n```\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Warnings) != 5 {
		t.Fatalf("warnings = %q", d.Warnings)
	}
	for _, w := range d.Warnings[:3] {
		if !strings.HasPrefix(w, "line 5: ") {
			t.Errorf("warning %q should point at the fence", w)
		}
	}
	if !strings.HasPrefix(d.Warnings[3], "line 10: ") || !strings.Contains(d.Warnings[3], "past the end") {
		t.Errorf("got %q", d.Warnings[3])
	}
	if !strings.HasPrefix(d.Warnings[4], "line 14: ") || !strings.Contains(d.Warnings[4], "unclosed {") {
		t.Errorf("got %q", d.Warnings[4])
	}
	// The block with one usable group still highlights it; the all-bad ones fall back to plain code.
	var kinds []BlockKind
	for _, b := range d.Slides[0].Blocks {
		kinds = append(kinds, b.Kind)
	}
	if !reflect.DeepEqual(kinds, []BlockKind{Markdown, Code, Markdown}) {
		t.Errorf("kinds = %v", kinds)
	}
	if d.Slides[0].Steps != 1 {
		t.Errorf("Steps = %d", d.Slides[0].Steps)
	}
}

func TestFenceHighlightUnterminatedAndEmpty(t *testing.T) {
	d, err := Parse("```go {2}\na\nb\n")
	if err != nil {
		t.Fatal(err)
	}
	if b := d.Slides[0].Blocks[0]; b.Kind != Code || b.Text != "a\nb" {
		t.Errorf("%+v", b)
	}
	d, _ = Parse("```go {1}\n```\n")
	if len(d.Warnings) == 0 || d.Slides[0].Blocks[0].Kind != Markdown {
		t.Errorf("empty block: %q %+v", d.Warnings, d.Slides[0].Blocks)
	}
}

func TestFenceHighlightInColumnsAndTilde(t *testing.T) {
	d, _ := Parse("<!-- column_layout: [1, 1] -->\n<!-- column: 0 -->\n~~~py {1|2}\na\nb\n~~~\n<!-- column: 1 -->\nright\n<!-- reset_layout -->\n")
	s := d.Slides[0]
	if s.Steps != 2 {
		t.Fatalf("Steps = %d", s.Steps)
	}
	cols := s.Blocks[0]
	if cols.Children[0][0].Kind != Code || cols.Children[1][0].Step != 1 {
		t.Errorf("%+v", cols.Children)
	}
}

func TestFenceHighlightInsideNestedFenceIgnored(t *testing.T) {
	// A {..} line inside a longer fence is code, not a fence opener.
	d, _ := Parse("````md\n```go {1|2}\nx\n```\n````\n")
	if b := d.Slides[0].Blocks; len(b) != 1 || b[0].Kind != Markdown || d.Slides[0].Steps != 1 {
		t.Errorf("%+v", b)
	}
}
