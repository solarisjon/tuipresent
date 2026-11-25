package render

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"tuipresent/internal/deck"
)

func TestParseInlineTitle(t *testing.T) {
	for in, want := range map[string]string{
		"**Bold** and `code`":      "Bold and code",
		"a *b* ~~c~~ ==d==":        "a b c d",
		"unclosed ** and `":        "unclosed ** and `",
		`escaped \*star\*`:         "escaped *star*",
		"**bold *nested* inside**": "bold nested inside",
	} {
		if got := PlainTitle(in); got != want {
			t.Errorf("PlainTitle(%q) = %q, want %q", in, got, want)
		}
	}
	runs := parseInline("**a** *b*")
	if len(runs) != 3 || !runs[0].bold || runs[1].bold || !runs[2].italic {
		t.Errorf("runs = %+v", runs)
	}
}

func renderDeck(t *testing.T, src string) *Renderer {
	t.Helper()
	d, err := deck.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	return New(d)
}

func TestTitleMarkdownRendersWithoutMarkers(t *testing.T) {
	r := renderDeck(t, "**Bold** `code`\n===\nbody\n")
	f := ansi.Strip(r.Frame(0, 0, 80, 24))
	if !strings.Contains(f, "Bold code") || strings.Contains(f, "**") || strings.Contains(f, "`") {
		t.Errorf("title not rendered as markdown:\n%s", f)
	}
	if rows, cols := r.Overflow(0, 80, 24); rows != 0 || cols != 0 {
		t.Errorf("overflow %d, %d", rows, cols)
	}
}

func TestHeadRows(t *testing.T) {
	defer func(c bool) { Curly = c }(Curly)
	for _, tc := range []struct {
		src   string
		curly bool
		want  int
	}{
		{"T\n===\nbody\n", false, 4},
		{"T\n===\n<!-- subtitle: s -->\nbody\n", false, 5},
		{"T\n===\n<!-- title_style: frame -->\nbody\n", false, 5},
		{"T\n===\n<!-- title_style: frame -->\n<!-- subtitle: s -->\nbody\n", false, 6},
		{"T\n===\n<!-- title_style: curly -->\nbody\n", false, 4},
		{"T\n===\n<!-- title_style: curly -->\nbody\n", true, 4},
		{"T\n===\n<!-- title_style: curly -->\n<!-- subtitle: s -->\nbody\n", true, 5},
	} {
		Curly = tc.curly
		r := renderDeck(t, tc.src)
		head, _ := r.head(0, r.deck.Slides[0], 72)
		if len(head) != tc.want {
			t.Errorf("%q curly=%v: head rows %d, want %d", tc.src, tc.curly, len(head), tc.want)
		}
		checkFrame(t, r.Frame(0, 0, 80, 24), 80, 24)
	}
}

func TestCurlyUnderlineOnlyWhereSupported(t *testing.T) {
	defer func(c bool) { Curly = c }(Curly)
	src := "T\n===\n<!-- title_style: curly -->\nbody\n"
	Curly = false
	if f := renderDeck(t, src).Frame(0, 0, 80, 24); strings.Contains(f, "4:3") || !strings.Contains(ansi.Strip(f), "━") {
		t.Error("fallback should draw the rule and no curly underline")
	}
	Curly = true
	r := renderDeck(t, src)
	head, _ := r.head(0, r.deck.Slides[0], 72)
	if h := strings.Join(head, "\n"); !strings.Contains(h, "\x1b[4:3m") || strings.Contains(ansi.Strip(h), "━") {
		t.Error("supported terminals should get a curly underline and no rule")
	}
}

func TestDetectCurly(t *testing.T) {
	env := func(m map[string]string) func(string) string { return func(k string) string { return m[k] } }
	if !DetectCurly(env(map[string]string{"TERM_PROGRAM": "ghostty"})) {
		t.Error("ghostty should support curly underlines")
	}
	if DetectCurly(env(map[string]string{"TERM_PROGRAM": "ghostty", "TMUX": "/tmp/x"})) {
		t.Error("tmux should not")
	}
	if DetectCurly(env(map[string]string{"TERM": "xterm-256color"})) {
		t.Error("unknown terminals should not")
	}
}

func TestFrameTitleBoxAligned(t *testing.T) {
	r := renderDeck(t, "Wide title here\n===\n<!-- title_style: frame -->\n<!-- subtitle: sub -->\nbody\n")
	lines := strings.Split(ansi.Strip(r.Frame(0, 0, 80, 24)), "\n")
	var box []string
	for _, l := range lines {
		if s := strings.TrimSpace(l); s != "" && strings.ContainsRune("╭│╰", []rune(s)[0]) {
			box = append(box, strings.TrimRight(l, " "))
		}
	}
	if len(box) != 4 {
		t.Fatalf("box lines = %q", box)
	}
	for _, l := range box {
		if ansi.StringWidth(l) != ansi.StringWidth(box[0]) {
			t.Errorf("ragged box: %q", box)
		}
	}
}

func TestBodyMarks(t *testing.T) {
	r := renderDeck(t, "T\n===\nsee ==this== and ++that++ and [faint]{.faint} and `==code==` and [x]{.nope}\n")
	f := r.Frame(0, 0, 100, 30)
	plain := ansi.Strip(f)
	if !strings.Contains(plain, "see this and that and faint and") || !strings.Contains(plain, "==code==") || !strings.Contains(plain, "[x]{.nope}") {
		t.Errorf("marks not resolved:\n%s", plain)
	}
	p := r.theme.Palette
	for _, want := range []string{sgrColor(48, p.Highlight), "\x1b[4m", "\x1b[2m"} {
		if !strings.Contains(f, want) {
			t.Errorf("frame lacks %q", want)
		}
	}
	for _, ch := range f {
		if isMark(ch) {
			t.Fatal("marker rune leaked into the frame")
		}
	}
}

func TestMarkSpanningWrappedLines(t *testing.T) {
	r := renderDeck(t, "T\n===\n==one two three four five six seven eight nine ten eleven twelve==\n")
	lines := strings.Split(r.Frame(0, 0, 40, 24), "\n")
	hi := sgrColor(48, r.theme.Palette.Highlight)
	n := 0
	for _, l := range lines {
		if strings.Contains(l, hi) {
			n++
			if !strings.Contains(l, "\x1b[49m") {
				t.Errorf("highlight left open at end of line: %q", l)
			}
		}
	}
	if n < 2 {
		t.Errorf("expected a highlight on each wrapped line, got %d", n)
	}
}

func TestVariantUnderlinesFallBack(t *testing.T) {
	defer func(c bool) { Curly = c }(Curly)
	src := "T\n===\n[a]{.curly} [b]{.dotted} [c]{.dashed} [d]{.double}\n"
	Curly = false
	if f := renderDeck(t, src).Frame(0, 0, 80, 24); strings.Contains(f, "4:") {
		t.Error("fallback must not use styled underlines")
	}
	Curly = true
	f := renderDeck(t, src).Frame(0, 0, 80, 24)
	for _, want := range []string{"\x1b[4:3m", "\x1b[4:4m", "\x1b[4:5m", "\x1b[4:2m"} {
		if !strings.Contains(f, want) {
			t.Errorf("missing %q", want)
		}
	}
}

func TestSplitCallouts(t *testing.T) {
	src := "before\n\n> [!NOTE]\n> one\n>\n> two\n\n> plain quote\n\n> [!tip]\n> t\nafter"
	got := splitCallouts(src)
	var kinds []string
	for _, p := range got {
		kinds = append(kinds, p.kind)
	}
	if strings.Join(kinds, ",") != ",note,,tip," {
		t.Fatalf("kinds = %q (%+v)", kinds, got)
	}
	if got[1].text != "one\n\ntwo" {
		t.Errorf("note body = %q", got[1].text)
	}
	if !strings.Contains(got[2].text, "> plain quote") {
		t.Errorf("plain quote lost: %q", got[2].text)
	}
	if bogus := splitCallouts("> [!BOGUS]\n> x"); len(bogus) != 1 || bogus[0].kind != "" {
		t.Error("unknown callout type must stay a normal quote")
	}
}

func TestCalloutBox(t *testing.T) {
	for _, kind := range []string{"NOTE", "TIP", "IMPORTANT", "WARNING", "CAUTION"} {
		r := renderDeck(t, "T\n===\n> [!"+kind+"]\n> Body text that is long enough to wrap onto a second line in the box, with `code`.\n")
		for _, size := range [][2]int{{80, 24}, {100, 30}, {40, 15}} {
			f := r.Frame(0, 0, size[0], size[1])
			checkFrame(t, f, size[0], size[1])
			plain := ansi.Strip(f)
			label := kind[:1] + strings.ToLower(kind[1:])
			if !strings.Contains(plain, "╭─ "+label+" ") || !strings.Contains(plain, "╰") {
				t.Errorf("%s at %v: no box:\n%s", kind, size, plain)
			}
			if rows, cols := r.Overflow(0, size[0], size[1]); rows != 0 || cols != 0 {
				t.Errorf("%s at %v overflows %d, %d", kind, size, rows, cols)
			}
		}
	}
	r := renderDeck(t, "T\n===\n> [!NOTE]\n> hi\n")
	lines := r.body(r.deck.Slides[0].Blocks, 0, 72)
	if len(lines) != 3 {
		t.Errorf("one-line callout is %d rows, want 3", len(lines))
	}
	for i, l := range lines {
		if ansi.StringWidth(l) != ansi.StringWidth(lines[0]) {
			t.Errorf("row %d width differs", i)
		}
	}
}

func TestCalloutInColumnsAndCodeFenceUntouched(t *testing.T) {
	src := "T\n===\n<!-- column_layout: [1, 1] -->\n<!-- column: 0 -->\n> [!WARNING]\n> narrow\n<!-- column: 1 -->\n```\n> [!NOTE]\n> code\n```\n"
	r := renderDeck(t, src)
	plain := ansi.Strip(r.Frame(0, 0, 100, 30))
	if !strings.Contains(plain, "Warning") || !strings.Contains(plain, "> [!NOTE]") {
		t.Errorf("columns/fence handling wrong:\n%s", plain)
	}
}

func TestStatusColorsDefaultForEveryTheme(t *testing.T) {
	for _, name := range ThemeNames() {
		th, _ := LookupTheme(name)
		p := th.Palette
		for k, v := range map[string]string{"highlight": p.Highlight, "note": p.Note, "tip": p.Tip, "important": p.Important, "warning": p.Warning, "caution": p.Caution} {
			if _, ok := normalizeHex(v); !ok {
				t.Errorf("%s: %s = %q", name, k, v)
			}
		}
	}
	d, _ := deck.Parse("---\ntheme: {name: nord, warning: \"#ff0000\", highlight: \"#00ff00\"}\n---\nx\n")
	r := New(d)
	if r.Theme().Palette.Warning != "#ff0000" || r.Theme().Palette.Highlight != "#00ff00" || len(r.ThemeWarnings()) != 0 {
		t.Errorf("overrides not applied: %+v %q", r.Theme().Palette, r.ThemeWarnings())
	}
}
