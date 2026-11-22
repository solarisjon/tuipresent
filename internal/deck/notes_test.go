package deck

import (
	"reflect"
	"strings"
	"testing"
)

func allText(blocks []Block) string {
	var sb strings.Builder
	for _, b := range blocks {
		sb.WriteString(b.Text)
		for _, c := range b.Children {
			sb.WriteString(allText(c))
		}
	}
	return sb.String()
}

func TestSpeakerNotes(t *testing.T) {
	tests := []struct {
		name      string
		src       string
		wantNotes []string
		wantSteps int
		wantTitle string
	}{
		{
			name:      "single line",
			src:       "T\n===\n\nbody\n\n<!-- speaker_note: say hello -->\n",
			wantNotes: []string{"say hello"},
			wantSteps: 1,
			wantTitle: "T",
		},
		{
			name:      "multi line keeps relative indentation",
			src:       "body\n\n<!-- speaker_note:\n  line one\n    - nested\n  line two\n  -->\n",
			wantNotes: []string{"line one\n  - nested\nline two"},
			wantSteps: 1,
		},
		{
			name:      "multi line with text on opener and closer",
			src:       "body\n<!-- speaker_note: first\nsecond -->\n",
			wantNotes: []string{"first\nsecond"},
			wantSteps: 1,
		},
		{
			name:      "multiple notes accumulate in order",
			src:       "<!-- speaker_note: one -->\nbody\n<!-- speaker_note:\ntwo\n-->\n<!-- speaker_note: three -->\n",
			wantNotes: []string{"one", "two", "three"},
			wantSteps: 1,
		},
		{
			name:      "note before setext title does not block title detection",
			src:       "<!-- speaker_note: intro -->\nTitle\n===\n\nbody\n",
			wantNotes: []string{"intro"},
			wantSteps: 1,
			wantTitle: "Title",
		},
		{
			name:      "inside code fence ignored",
			src:       "```markdown\n<!-- speaker_note: not a note -->\n<!-- speaker_note:\nnor this\n-->\n```\n",
			wantNotes: nil,
			wantSteps: 1,
		},
		{
			name:      "empty note dropped",
			src:       "body\n<!-- speaker_note: -->\n<!-- speaker_note -->\n",
			wantNotes: nil,
			wantSteps: 1,
		},
		{
			name: "mixed with pauses and columns",
			src: "<!-- column_layout: [1, 1] -->\n<!-- column: 0 -->\nleft\n<!-- speaker_note: about left -->\n" +
				"<!-- column: 1 -->\nright\n<!-- speaker_note:\nabout right\n-->\n<!-- reset_layout -->\n<!-- pause -->\nafter\n",
			wantNotes: []string{"about left", "about right"},
			wantSteps: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := mustParse(t, tt.src)
			if len(d.Slides) != 1 {
				t.Fatalf("slides = %d, want 1", len(d.Slides))
			}
			s := d.Slides[0]
			if !reflect.DeepEqual(s.Notes, tt.wantNotes) {
				t.Errorf("Notes = %q, want %q", s.Notes, tt.wantNotes)
			}
			if s.Steps != tt.wantSteps {
				t.Errorf("Steps = %d, want %d", s.Steps, tt.wantSteps)
			}
			if s.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", s.Title, tt.wantTitle)
			}
			text := allText(s.Blocks)
			if tt.wantNotes != nil && strings.Contains(text, "speaker_note") {
				t.Errorf("note leaked into slide markdown: %q", text)
			}
			for _, n := range tt.wantNotes {
				if strings.Contains(text, n) {
					t.Errorf("note %q leaked into slide markdown: %q", n, text)
				}
			}
		})
	}
}

func TestSpeakerNoteFenceStaysCode(t *testing.T) {
	d := mustParse(t, "```markdown\n<!-- speaker_note:\nshown\n-->\n```\n")
	if got := allText(d.Slides[0].Blocks); !strings.Contains(got, "shown") {
		t.Errorf("code fence content lost: %q", got)
	}
}

func TestSpeakerNoteDoesNotHideEndSlide(t *testing.T) {
	// The multi-line note is consumed whole, so an end_slide after it still splits.
	d := mustParse(t, "a\n<!-- speaker_note:\nfor a\n-->\n<!-- end_slide -->\nb\n<!-- speaker_note: for b -->\n")
	if len(d.Slides) != 2 {
		t.Fatalf("slides = %d, want 2", len(d.Slides))
	}
	if !reflect.DeepEqual(d.Slides[0].Notes, []string{"for a"}) || !reflect.DeepEqual(d.Slides[1].Notes, []string{"for b"}) {
		t.Errorf("notes = %q / %q", d.Slides[0].Notes, d.Slides[1].Notes)
	}
}
