package deck

import (
	"strings"
	"testing"
)

// checkDeck asserts the invariants the renderer and presenter rely on.
func checkDeck(t *testing.T, d *Deck) {
	t.Helper()
	if len(d.Slides) == 0 {
		t.Fatal("deck with no slides and no error")
	}
	for i, s := range d.Slides {
		if s.Steps < 1 {
			t.Fatalf("slide %d: Steps = %d", i+1, s.Steps)
		}
		var walk func(bs []Block)
		walk = func(bs []Block) {
			for _, b := range bs {
				if b.Step < 0 || b.Step+max(len(b.Highlights)-1, 0) > s.Steps-1 {
					t.Fatalf("slide %d: block step %d (+%d highlight groups) outside 0..%d", i+1, b.Step, len(b.Highlights), s.Steps-1)
				}
				if b.Kind == Columns {
					if len(b.Children) != len(b.Widths) {
						t.Fatalf("slide %d: %d columns but %d widths", i+1, len(b.Children), len(b.Widths))
					}
					for _, c := range b.Children {
						walk(c)
					}
				}
			}
		}
		walk(s.Blocks)
	}
}

// Directives that interact must parse without panic, with a sane step count
// and a warning for whatever was wrong.
func TestParseDirectiveCombinations(t *testing.T) {
	for _, tc := range []struct {
		name, src string
		steps     int // of the first slide; 0 = don't care
		warn      string
	}{
		{"highlight groups plus pauses", "T\n===\na\n<!-- pause -->\n```go {1|2|3}\nx\ny\nz\n```\n<!-- pause -->\nb\n", 5, ""},
		{"highlights in columns", "T\n===\n<!-- column_layout: [1,1] -->\n<!-- column: 0 -->\n```go {1|2}\na\nb\n```\n<!-- column: 1 -->\nx\n", 2, ""},
		{"exec with highlights", "T\n===\n```sh +exec {1|2}\necho a\necho b\n```\n", 2, ""},
		{"unclosed highlight brace", "T\n===\n```go {1|2\nx\n```\n", 1, "unclosed {"},
		{"unterminated fence with highlights", "T\n===\n```go {1|2}\nx\ny", 2, ""},
		{"unterminated note", "T\n===\n<!-- speaker_note: x\nmore\n<!-- pause -->\nb\n", 2, "no closing -->"},
		{"title directives together", "T\n===\n<!-- title_style: frame -->\n<!-- title_size: 2 -->\n<!-- subtitle: s -->\nb\n", 1, ""},
		{"bad title size and style", "T\n===\n<!-- title_style: zzz -->\n<!-- title_size: 7 -->\nb\n", 1, "unknown title_size"},
		{"column outside layout", "T\n===\n<!-- column: 2 -->\nb\n", 1, ""},
		{"empty column layout", "T\n===\n<!-- column_layout: [] -->\n<!-- column: 0 -->\nb\n", 1, ""},
		{"front matter all wrong types", "---\ntitle: 5\ntheme: 7\ntransition: [a]\n---\nT\n===\nb\n", 0, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, err := Parse(tc.src)
			if err != nil {
				return // a front matter error is fine; it must not panic
			}
			checkDeck(t, d)
			first := d.Slides[0]
			if first.Cover && len(d.Slides) > 1 {
				first = d.Slides[1]
			}
			if tc.steps != 0 && first.Steps != tc.steps {
				t.Errorf("steps = %d, want %d", first.Steps, tc.steps)
			}
			if tc.warn != "" && !strings.Contains(strings.Join(d.Warnings, "\n"), tc.warn) {
				t.Errorf("warnings %q lack %q", d.Warnings, tc.warn)
			}
		})
	}
}

func FuzzParse(f *testing.F) {
	for _, s := range []string{
		"T\n===\nbody\n",
		"---\ntitle: x\ntheme: nord\n---\nT\n===\n",
		"T\n===\n<!-- column_layout: [1,2] -->\n<!-- column: 1 -->\n```go {1|2}\nx\n```\n",
		"```sh +exec {1|2}\n",
		"<!--\nspeaker_note: x\n",
		"<!-- title_size: 3 -->\n<!-- subtitle: x -->\n<!-- title_style: frame -->\n",
		"<!-- end_slide -->\n<!-- pause -->\n<!-- column: -1 -->\n<!-- column_layout: [0] -->\n",
		"---\n---\n---\n",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, src string) {
		d, err := Parse(src)
		if err != nil {
			return
		}
		checkDeck(t, d)
	})
}
