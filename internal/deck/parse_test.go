package deck

import (
	"strings"
	"testing"
)

func mustParse(t *testing.T, src string) *Deck {
	t.Helper()
	d, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return d
}

func TestFrontMatter(t *testing.T) {
	tests := []struct {
		name       string
		src        string
		wantMeta   Meta
		wantCover  bool
		wantSlides int
	}{
		{
			name: "full front matter makes cover",
			src:  "---\ntitle: T\nsub_title: S\nauthor: A\ndate: D\ntheme:\n  name: dracula\ntransition: wipe\n---\n\nhello\n",
			wantMeta: func() Meta {
				m := Meta{Title: "T", SubTitle: "S", Author: "A", Date: "D", Transition: "wipe"}
				m.Theme.Name = "dracula"
				return m
			}(),
			wantCover:  true,
			wantSlides: 2,
		},
		{
			name:       "no title means no cover",
			src:        "---\nauthor: A\n---\nhello\n",
			wantMeta:   Meta{Author: "A"},
			wantCover:  false,
			wantSlides: 1,
		},
		{
			name:       "no front matter",
			src:        "hello\n",
			wantSlides: 1,
		},
		{
			name:       "front matter only, closing at EOF",
			src:        "---\ntitle: Only\n---",
			wantMeta:   Meta{Title: "Only"},
			wantCover:  true,
			wantSlides: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := mustParse(t, tt.src)
			if d.Meta != tt.wantMeta {
				t.Errorf("Meta = %+v, want %+v", d.Meta, tt.wantMeta)
			}
			if len(d.Slides) != tt.wantSlides {
				t.Fatalf("got %d slides, want %d", len(d.Slides), tt.wantSlides)
			}
			if got := d.Slides[0].Cover; got != tt.wantCover {
				t.Errorf("Slides[0].Cover = %v, want %v", got, tt.wantCover)
			}
			if tt.wantCover {
				c := d.Slides[0]
				if c.Steps != 1 || !c.Centered || len(c.Blocks) != 0 {
					t.Errorf("cover slide = %+v", c)
				}
			}
		})
	}
}

func TestParseErrors(t *testing.T) {
	for _, src := range []string{"", "\n\n", "<!-- end_slide -->\n<!-- end_slide -->\n", "---\ntitle: [unclosed\n---\nx\n"} {
		if _, err := Parse(src); err == nil {
			t.Errorf("Parse(%q) succeeded, want error", src)
		}
	}
}

func TestSlideSplitting(t *testing.T) {
	tests := []struct {
		name  string
		src   string
		texts []string // first block text of each slide
	}{
		{"single", "a\n", []string{"a"}},
		{"two", "a\n<!-- end_slide -->\nb\n", []string{"a", "b"}},
		{"crlf", "a\r\n<!-- end_slide -->\r\nb\r\n", []string{"a", "b"}},
		{"empty slides dropped", "a\n<!-- end_slide -->\n\n<!-- end_slide -->\nb\n", []string{"a", "b"}},
		{"whitespace in directive", "a\n  <!--end_slide-->  \nb\n", []string{"a", "b"}},
		{
			"end_slide inside backtick fence does not split",
			"```html\n<!-- end_slide -->\n```\n<!-- end_slide -->\nb\n",
			[]string{"```html\n<!-- end_slide -->\n```", "b"},
		},
		{
			"end_slide inside tilde fence does not split",
			"~~~\n<!-- end_slide -->\n~~~\n",
			[]string{"~~~\n<!-- end_slide -->\n~~~"},
		},
		{
			"longer fence not closed by shorter",
			"````\n```\n<!-- end_slide -->\n```\n````\n",
			[]string{"````\n```\n<!-- end_slide -->\n```\n````"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := mustParse(t, tt.src)
			if len(d.Slides) != len(tt.texts) {
				t.Fatalf("got %d slides, want %d", len(d.Slides), len(tt.texts))
			}
			for i, want := range tt.texts {
				bs := d.Slides[i].Blocks
				if len(bs) == 0 {
					t.Fatalf("slide %d has no blocks", i)
				}
				if got := strings.TrimSpace(bs[0].Text); got != want {
					t.Errorf("slide %d text = %q, want %q", i, got, want)
				}
			}
		})
	}
}

func TestTitle(t *testing.T) {
	tests := []struct {
		name, src, title string
	}{
		{"setext", "Hello\n===\nbody\n", "Hello"},
		{"leading blank lines", "\n\n  Hello  \n=\nbody\n", "Hello"},
		{"after directive", "<!-- jump_to_middle -->\nHello\n===\n", "Hello"},
		{"not after content", "body\n\nHello\n===\n", ""},
		{"atx heading is not title", "# Hello\nbody\n", ""},
		{"setext h2 is not title", "Hello\n---\nbody\n", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := mustParse(t, tt.src).Slides[0]
			if s.Title != tt.title {
				t.Errorf("Title = %q, want %q", s.Title, tt.title)
			}
			if tt.title != "" {
				for _, b := range s.Blocks {
					if strings.Contains(b.Text, "===") || strings.Contains(b.Text, tt.title) {
						t.Errorf("title leaked into block %q", b.Text)
					}
				}
			}
		})
	}
}

func TestTitleOnlySlideKept(t *testing.T) {
	d := mustParse(t, "Just a title\n===\n")
	if len(d.Slides) != 1 || d.Slides[0].Title != "Just a title" || len(d.Slides[0].Blocks) != 0 {
		t.Fatalf("got %+v", d.Slides)
	}
}

func TestPauses(t *testing.T) {
	tests := []struct {
		name      string
		src       string
		steps     int
		wantSteps []int // Step of each top-level block
	}{
		{"none", "a\n\nb\n", 1, []int{0}},
		{"one", "a\n<!-- pause -->\nb\n", 2, []int{0, 1}},
		{"two", "a\n<!-- pause -->\nb\n<!-- pause -->\nc\n", 3, []int{0, 1, 2}},
		{"leading pause", "<!-- pause -->\na\n", 2, []int{1}},
		{"trailing pause counts", "a\n<!-- pause -->\n", 2, []int{0}},
		{"pause inside fence ignored", "```\n<!-- pause -->\n```\n", 1, []int{0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := mustParse(t, tt.src).Slides[0]
			if s.Steps != tt.steps {
				t.Errorf("Steps = %d, want %d", s.Steps, tt.steps)
			}
			if len(s.Blocks) != len(tt.wantSteps) {
				t.Fatalf("got %d blocks, want %d: %+v", len(s.Blocks), len(tt.wantSteps), s.Blocks)
			}
			for i, w := range tt.wantSteps {
				if s.Blocks[i].Step != w {
					t.Errorf("block %d Step = %d, want %d", i, s.Blocks[i].Step, w)
				}
			}
		})
	}
}

func TestPauseKeepsRawSource(t *testing.T) {
	src := "* one\n<!-- pause -->\n* two\n"
	s := mustParse(t, src).Slides[0]
	var parts []string
	for _, b := range s.Blocks {
		parts = append(parts, b.Text)
	}
	if got := strings.Join(parts, "\n"); got != "* one\n* two\n" {
		t.Errorf("rejoined = %q", got)
	}
}

func TestColumns(t *testing.T) {
	src := `before
<!-- column_layout: [2, 1] -->
<!-- column: 0 -->
left a
<!-- pause -->
left b
<!-- column: 1 -->
right
<!-- reset_layout -->
after
`
	s := mustParse(t, src).Slides[0]
	if s.Steps != 2 {
		t.Errorf("Steps = %d, want 2", s.Steps)
	}
	if len(s.Blocks) != 3 {
		t.Fatalf("got %d top-level blocks, want 3: %+v", len(s.Blocks), s.Blocks)
	}
	if s.Blocks[0].Kind != Markdown || strings.TrimSpace(s.Blocks[0].Text) != "before" {
		t.Errorf("block 0 = %+v", s.Blocks[0])
	}
	cols := s.Blocks[1]
	if cols.Kind != Columns || cols.Step != 0 {
		t.Fatalf("block 1 = %+v", cols)
	}
	if len(cols.Widths) != 2 || cols.Widths[0] != 2 || cols.Widths[1] != 1 {
		t.Errorf("Widths = %v", cols.Widths)
	}
	if len(cols.Children) != 2 {
		t.Fatalf("Children len = %d", len(cols.Children))
	}
	left, right := cols.Children[0], cols.Children[1]
	if len(left) != 2 || strings.TrimSpace(left[0].Text) != "left a" || left[0].Step != 0 ||
		strings.TrimSpace(left[1].Text) != "left b" || left[1].Step != 1 {
		t.Errorf("left column = %+v", left)
	}
	if len(right) != 1 || strings.TrimSpace(right[0].Text) != "right" || right[0].Step != 1 {
		t.Errorf("right column = %+v", right)
	}
	after := s.Blocks[2]
	if after.Kind != Markdown || strings.TrimSpace(after.Text) != "after" || after.Step != 1 {
		t.Errorf("block 2 = %+v", after)
	}
}

func TestColumnEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		widths   []int
		topLevel int // number of top-level blocks
	}{
		{"bad widths default to [1]", "<!-- column_layout: [a, 0] -->\n<!-- column: 0 -->\nx\n", []int{1}, 1},
		{"column without layout ignored", "<!-- column: 0 -->\nx\n", nil, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := mustParse(t, tt.src).Slides[0]
			if len(s.Blocks) != tt.topLevel {
				t.Fatalf("got %d top-level blocks, want %d: %+v", len(s.Blocks), tt.topLevel, s.Blocks)
			}
			if tt.widths == nil {
				return
			}
			c := s.Blocks[0]
			if c.Kind != Columns || len(c.Widths) != len(tt.widths) {
				t.Fatalf("block 0 = %+v", c)
			}
			for i := range tt.widths {
				if c.Widths[i] != tt.widths[i] {
					t.Errorf("Widths = %v, want %v", c.Widths, tt.widths)
				}
			}
		})
	}
}

func TestSlideDirectives(t *testing.T) {
	tests := []struct {
		name       string
		src        string
		centered   bool
		transition string
	}{
		{"default", "x\n", false, ""},
		{"jump_to_middle", "<!-- jump_to_middle -->\nx\n", true, ""},
		{"transition", "<!-- transition: dissolve -->\nx\n", false, "dissolve"},
		{"transition with spaces", "<!--   transition:   cover   -->\nx\n", false, "cover"},
		{"both", "x\n<!-- jump_to_middle -->\n<!-- transition: none -->\n", true, "none"},
		{"unknown directive ignored", "<!-- speaker_note: hi -->\nx\n", false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := mustParse(t, tt.src).Slides[0]
			if s.Centered != tt.centered {
				t.Errorf("Centered = %v, want %v", s.Centered, tt.centered)
			}
			if s.Transition != tt.transition {
				t.Errorf("Transition = %q, want %q", s.Transition, tt.transition)
			}
			for _, b := range s.Blocks {
				if strings.Contains(b.Text, "<!--") {
					t.Errorf("directive leaked into block text %q", b.Text)
				}
			}
		})
	}
}

func TestTransitionPerSlide(t *testing.T) {
	d := mustParse(t, "<!-- transition: wipe -->\na\n<!-- end_slide -->\nb\n")
	if d.Slides[0].Transition != "wipe" || d.Slides[1].Transition != "" {
		t.Errorf("transitions = %q, %q", d.Slides[0].Transition, d.Slides[1].Transition)
	}
}

func TestExec(t *testing.T) {
	src := "intro\n```bash +exec\necho hi\necho there\n```\n<!-- end_slide -->\n" +
		"<!-- pause -->\n```python3 +exec\nprint(1)\n```\n" +
		"```sh\nnot exec\n```\n" +
		"```+exec\nls\n```\n"
	d := mustParse(t, src)
	if len(d.Slides) != 2 {
		t.Fatalf("got %d slides", len(d.Slides))
	}

	var execs []Block
	for _, s := range d.Slides {
		for _, b := range s.Blocks {
			if b.Kind == Exec {
				execs = append(execs, b)
			}
		}
	}
	want := []struct {
		lang, text string
		step       int
	}{
		{"bash", "echo hi\necho there", 0},
		{"python3", "print(1)", 1},
		{"", "ls", 1},
	}
	if len(execs) != len(want) {
		t.Fatalf("got %d exec blocks, want %d", len(execs), len(want))
	}
	seen := map[int]bool{}
	for i, w := range want {
		e := execs[i]
		if e.Lang != w.lang || e.Text != w.text || e.Step != w.step {
			t.Errorf("exec %d = {Lang:%q Text:%q Step:%d}, want %+v", i, e.Lang, e.Text, e.Step, w)
		}
		if e.ExecID == 0 || seen[e.ExecID] {
			t.Errorf("exec %d has non-unique or zero ExecID %d", i, e.ExecID)
		}
		seen[e.ExecID] = true
	}

	// The non-exec fence stays in markdown, and the intro precedes the exec block.
	s0 := d.Slides[0].Blocks
	if len(s0) != 2 || s0[0].Kind != Markdown || strings.TrimSpace(s0[0].Text) != "intro" {
		t.Errorf("slide 0 blocks = %+v", s0)
	}
	found := false
	for _, b := range d.Slides[1].Blocks {
		if b.Kind == Markdown && strings.Contains(b.Text, "not exec") {
			found = true
		}
	}
	if !found {
		t.Error("plain sh fence not kept as markdown")
	}
}

func TestExecInColumn(t *testing.T) {
	src := "<!-- column_layout: [1, 1] -->\n<!-- column: 1 -->\n```bash +exec\necho x\n```\n"
	s := mustParse(t, src).Slides[0]
	if len(s.Blocks) != 1 || s.Blocks[0].Kind != Columns {
		t.Fatalf("blocks = %+v", s.Blocks)
	}
	right := s.Blocks[0].Children[1]
	if len(right) != 1 || right[0].Kind != Exec || right[0].Text != "echo x" {
		t.Errorf("right column = %+v", right)
	}
}

func TestUnclosedFence(t *testing.T) {
	s := mustParse(t, "```bash +exec\necho x\n").Slides[0]
	if len(s.Blocks) != 1 || s.Blocks[0].Kind != Exec || s.Blocks[0].Text != "echo x\n" {
		t.Errorf("blocks = %+v", s.Blocks)
	}
}

func TestDemoDeck(t *testing.T) {
	d, err := Load("../../cpe-staff-meeting-demo.md")
	if err != nil {
		t.Fatal(err)
	}
	if d.Meta.Title != "CPE Staff Meeting" || d.Meta.Theme.Name != "tokyonight-storm" {
		t.Errorf("Meta = %+v", d.Meta)
	}
	want := []struct {
		title string
		steps int
	}{
		{"", 1}, // cover
		{"Agenda", 1},
		{"We are an escalation-sustaining", 2},
		{"Offsite: how we create capacity", 4},
		{"Thrive: engagement 71, four-year slide", 2},
		{"Thrive: strong teams, strained above", 2},
		{"CSE culture ambassadors", 1},
		{"Sev1: one owner, one source of truth", 2},
		{"Before the year closes", 2},
	}
	if len(d.Slides) != len(want) {
		t.Fatalf("got %d slides, want %d", len(d.Slides), len(want))
	}
	for i, w := range want {
		s := d.Slides[i]
		if s.Title != w.title || s.Steps != w.steps {
			t.Errorf("slide %d = {Title:%q Steps:%d}, want %+v", i, s.Title, s.Steps, w)
		}
	}
	if !d.Slides[0].Cover || !d.Slides[2].Centered {
		t.Error("expected cover slide 0 and centered slide 2")
	}
	sev := d.Slides[7].Blocks
	if last := sev[len(sev)-1]; last.Kind != Exec || last.Lang != "bash" || last.Step != 1 || last.ExecID != 1 {
		t.Errorf("sev1 exec block = %+v", last)
	}
}

func TestNestedFenceOpenerDoesNotSplit(t *testing.T) {
	src := "a\n```\n```bash\n<!-- end_slide -->\n```\nb\n"
	d, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Slides) != 1 {
		t.Fatalf("got %d slides, want 1", len(d.Slides))
	}
}

func TestThemeScalarAndMapping(t *testing.T) {
	for _, src := range []string{"---\ntitle: x\ntheme: dracula\n---\na\n", "---\ntitle: x\ntheme:\n  name: dracula\n---\na\n"} {
		d, err := Parse(src)
		if err != nil {
			t.Fatal(err)
		}
		if d.Meta.Theme.Name != "dracula" {
			t.Fatalf("theme = %q", d.Meta.Theme.Name)
		}
	}
}

func TestColumnIntroAndClamp(t *testing.T) {
	src := "<!-- column_layout: [1, 1] -->\nintro\n<!-- column: 0 -->\nleft\n<!-- column: 9 -->\nright\n<!-- reset_layout -->\nafter\n"
	d, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	b := d.Slides[0].Blocks
	if len(b) != 3 || b[0].Kind != Markdown || b[1].Kind != Columns || b[2].Kind != Markdown {
		t.Fatalf("want [md, columns, md], got %+v", b)
	}
	if !strings.Contains(b[0].Text, "intro") || !strings.Contains(b[1].Children[1][0].Text, "right") {
		t.Fatalf("intro should precede layout and column 9 should clamp to 1: %+v", b)
	}
}
