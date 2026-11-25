package render

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/solarisjon/tuipresent/internal/deck"
	"github.com/solarisjon/tuipresent/internal/transition"
)

// Cross-feature checks: every combination of directives and terminal
// capabilities must render frames of exactly w x h cells without panicking,
// and must never leave an OSC 8 hyperlink open at the end of a line (an open
// link would turn the rest of the screen into one).

var integrationSizes = [][2]int{{100, 30}, {80, 24}, {60, 20}, {40, 12}, {20, 5}, {10, 3}, {1, 1}}

const linkBody = "See [the docs](https://example.com/docs) and [more](https://example.org/a-very-long-path/that/keeps/going/and/going).\n"

var integrationDecks = map[string]string{
	"links-everywhere": "Links [t](https://t.example)\n===\n<!-- subtitle: sub [s](https://s.example) -->\n" + linkBody +
		"\n* item [one](https://one.example)\n* item two\n\n> quoted [q](https://q.example)\n\n| a | b |\n|---|---|\n| [x](https://x.example) | y |\n",
	"links-in-columns": "T\n===\n<!-- column_layout: [1, 1] -->\n<!-- column: 0 -->\n" + linkBody +
		"<!-- column: 1 -->\n> [!NOTE]\n> callout [c](https://c.example) with a link\n<!-- reset_layout -->\nafter\n",
	"callouts-all":            "T\n===\n> [!NOTE]\n> note\n\n> [!TIP]\n> tip ==hi== ++u++ ~~s~~\n\n> [!IMPORTANT]\n> imp\n\n> [!WARNING]\n> warn\n\n> [!CAUTION]\n> caution [x]{.curly} [y]{.dotted}\n",
	"title-combo":             "Big\n===\n<!-- title_style: frame -->\n<!-- title_size: 3 -->\n<!-- subtitle: with a subtitle -->\nbody\n",
	"title-combo-curly":       "Big\n===\n<!-- title_style: curly -->\n<!-- title_size: 2 -->\n<!-- subtitle: sub -->\n<!-- jump_to_middle -->\nbody [l](https://l.example)\n",
	"title-weird":             "Ünïcödé 日本語 title\n===\n<!-- title_size: 2 -->\n<!-- title_style: nonsense -->\n<!-- subtitle: -->\nbody\n",
	"title-size-bad":          "T\n===\n<!-- title_size: 9 -->\n<!-- title_size: -->\nbody\n",
	"highlights-pauses":       "T\n===\nbefore\n<!-- pause -->\n```go {1-2|3|all}\nline1\nline2\nline3\n```\n<!-- pause -->\nafter\n",
	"highlights-columns":      "T\n===\n<!-- column_layout: [1, 1] -->\n<!-- column: 0 -->\n```go {1|2}\na := 1\nb := 2\n```\n<!-- column: 1 -->\n```sh +exec {1|2}\necho 1\necho 2\n```\n",
	"highlights-bad":          "T\n===\n```go {0|99|x-y|3-1|}\nonly\n```\n```go {1\nunclosed highlight spec\n```\n",
	"unterminated-fence":      "T\n===\n```go {1|2}\nnever closed\nlines\n",
	"unterminated-fence-exec": "T\n===\n```sh +exec\nnever closed",
	"unterminated-note":       "T\n===\n<!--\nspeaker_note: never closed\n<!-- pause -->\nbody\n<!-- end_slide -->\nnext\n",
	"column-weird":            "T\n===\n<!-- column: 3 -->\nstray\n<!-- column_layout: [0, 0, 0] -->\n<!-- column: -4 -->\nx\n<!-- column: 99 -->\ny\n<!-- column_layout: [] -->\nz\n",
	"column-huge":             "T\n===\n<!-- column_layout: [1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1] -->\n<!-- column: 7 -->\n" + linkBody,
	"styles-nested":           "**bold *italic ==hi== ~~gone~~* [x]{.curly}** `code ++u++`\n===\n++under++ ~~strike~~ [faint]{.faint} [d]{.double} [z]{.dashed}\n",
	"empty":                   "",
	"only-title":              "T\n===\n",
	"only-end":                "<!-- end_slide -->\n<!-- end_slide -->\n",
	"fade-off":                "T\n===\n<!-- fade: off -->\na\n<!-- pause -->\nb\n<!-- pause -->\nc\n",
	"centered":                "T\n===\n<!-- jump_to_middle -->\nstatement [l](https://l.example)\n<!-- pause -->\nmore\n",
	"code-wide":               "T\n===\n```\n" + strings.Repeat("x", 300) + "\n```\n",
	"wide-runes":              "T\n===\n日本語日本語日本語日本語日本語日本語日本語日本語日本語日本語日本語日本語日本語日本語日本語\n\n| 日本 | 😀😀😀 |\n|--|--|\n| 😀 | 日 |\n",
	"missing-image":           "T\n===\n![alt](does/not/exist.png)\n<!-- column_layout: [1, 1] -->\n<!-- column: 0 -->\n![a](nope.png)\n",
	"front-matter-weird":      "---\ntitle: [unclosed\n---\nbody\n",
	"front-matter-types":      "---\ntitle: 5\ntheme: 7\ntransition: [a, b]\nfade: yes\nprogress: 0\ntitle_size: big\ntitle_style: 3\n---\nT\n===\nbody\n",
	"front-matter-open":       "---\ntitle: never closed\nT\n===\nbody\n",
	"front-matter-title-size": "---\ntitle_size: 3\ntitle_style: frame\n---\nT\n===\nbody\n<!-- end_slide -->\nU\n===\n<!-- title_size: 1 -->\nbody\n",
}

type caps struct {
	links, curly, sizing bool
	paint                bool
}

func setCaps(t *testing.T, c caps) {
	t.Helper()
	oh, oc, ot := Hyperlinks, Curly, TextSizing
	t.Cleanup(func() { Hyperlinks, Curly, TextSizing = oh, oc, ot })
	Hyperlinks, Curly, TextSizing = c.links, c.curly, c.sizing
}

// openLinkAtEnd reports whether line leaves an OSC 8 hyperlink open, and
// whether it closes one it never opened.
func openLinkAtEnd(line string) (open bool) {
	for rest := line; ; {
		i := strings.Index(rest, "\x1b]8;")
		if i < 0 {
			return open
		}
		rest = rest[i+4:]
		end := strings.IndexAny(rest, "\x07\x1b")
		if end < 0 {
			return open
		}
		params := rest[:end]
		// params is "id=..;url" ; an empty url closes.
		_, url, _ := strings.Cut(params, ";")
		open = url != ""
		rest = rest[end:]
	}
}

func checkGeometry(t *testing.T, name string, f string, w, h int) {
	t.Helper()
	lines := strings.Split(f, "\n")
	if len(lines) != h {
		t.Errorf("%s: %d lines, want %d", name, len(lines), h)
		return
	}
	for i, l := range lines {
		if got := ansi.StringWidth(l); got != w {
			t.Errorf("%s: line %d is %d cells wide, want %d: %q", name, i, got, w, l)
			return
		}
		if openLinkAtEnd(l) {
			t.Errorf("%s: line %d ends inside an OSC 8 link: %q", name, i, l)
			return
		}
	}
}

func renderOrSkip(src string, paint bool) *Renderer {
	d, err := deck.Parse(src)
	if err != nil {
		return nil
	}
	if paint {
		d.Meta.Theme.Background = "paint"
	}
	return New(d)
}

func TestFrameGeometryAcrossFeatures(t *testing.T) {
	for name, src := range integrationDecks {
		for _, c := range []caps{{}, {links: true, curly: true, sizing: true}, {links: true, paint: true}, {sizing: true, paint: true, curly: true}} {
			setCaps(t, c)
			r := renderOrSkip(src, c.paint)
			if r == nil {
				continue
			}
			for si, s := range r.deck.Slides {
				for step := range s.Steps {
					for _, sz := range integrationSizes {
						label := fmt.Sprintf("%s %+v slide %d step %d %dx%d", name, c, si+1, step, sz[0], sz[1])
						func() {
							defer func() {
								if p := recover(); p != nil {
									t.Errorf("%s: panic: %v", label, p)
								}
							}()
							checkGeometry(t, label, r.Frame(si, step, sz[0], sz[1]), sz[0], sz[1])
							r.Overflow(si, sz[0], sz[1])
							r.BigTitleAt(si, step, sz[0], sz[1])
							r.Overview(si, sz[0], sz[1])
						}()
					}
				}
			}
		}
	}
}

// Transitions and fades slice frames by cell; links and styles in the middle
// of a slice must not leak out of it.
func TestTransitionsKeepLinksClosed(t *testing.T) {
	setCaps(t, caps{links: true, curly: true})
	r := renderDeck(t, "A\n===\n"+linkBody+"\n> [!NOTE]\n> a [n](https://n.example) link\n<!-- end_slide -->\nB\n===\n<!-- column_layout: [1, 1] -->\n<!-- column: 0 -->\n"+linkBody+"<!-- column: 1 -->\n[x]{.curly} ==hi==\n")
	const w, h = 80, 24
	a, b := r.Frame(0, 0, w, h), r.Frame(1, 0, w, h)
	if !strings.Contains(a, "\x1b]8;") {
		t.Fatalf("test frame has no OSC 8 link:\n%q", a)
	}
	for _, k := range transition.Kinds {
		for _, rev := range []bool{false, true} {
			for _, p := range []float64{0, 0.1, 0.25, 0.5, 0.77, 0.99} {
				out := transition.Composite(k, a, b, w, p, rev)
				for i, l := range strings.Split(out, "\n") {
					if openLinkAtEnd(l) {
						t.Fatalf("%s reverse=%v t=%v: line %d leaves a link open: %q", k, rev, p, i, l)
					}
					if got := ansi.StringWidth(l); got != w {
						t.Fatalf("%s reverse=%v t=%v: line %d is %d wide, want %d", k, rev, p, i, got, w)
					}
				}
			}
		}
	}
}

func TestFadeKeepsLinksClosed(t *testing.T) {
	setCaps(t, caps{links: true, curly: true})
	r := renderDeck(t, "A\n===\nfirst [a](https://a.example)\n<!-- pause -->\nsecond [b](https://b.example) ==hi== [x]{.curly}\n<!-- pause -->\n```go {1|2}\nx\ny\n```\n")
	const w, h = 80, 24
	for step := 1; step < r.deck.Slides[0].Steps; step++ {
		from, to := r.Frame(0, step-1, w, h), r.Frame(0, step, w, h)
		for _, p := range []float64{0, 0.3, 0.7, 0.99} {
			out := transition.FadeIn(from, to, p, r.Background(), r.Foreground())
			for i, l := range strings.Split(out, "\n") {
				if openLinkAtEnd(l) {
					t.Fatalf("step %d t=%v: line %d leaves a link open: %q", step, p, i, l)
				}
				if got := ansi.StringWidth(l); got != w {
					t.Fatalf("step %d t=%v: line %d is %d wide, want %d", step, p, i, got, w)
				}
			}
		}
	}
}

func FuzzFrame(f *testing.F) {
	for _, src := range integrationDecks {
		f.Add(src, uint8(80), uint8(24))
	}
	f.Add("T\n===\n```go {1|2}\nx\n", uint8(1), uint8(1))
	f.Fuzz(func(t *testing.T, src string, w, h uint8) {
		if len(src) > 4000 {
			return
		}
		d, err := deck.Parse(src)
		if err != nil {
			return
		}
		r := New(d)
		ww, hh := int(w)%130+1, int(h)%45+1
		for si, s := range d.Slides {
			for step := range min(s.Steps, 6) {
				got := r.Frame(si, step, ww, hh)
				lines := strings.Split(got, "\n")
				if len(lines) != hh {
					t.Fatalf("%dx%d slide %d step %d: %d lines", ww, hh, si+1, step, len(lines))
				}
				for i, l := range lines {
					if ansi.StringWidth(l) != ww {
						t.Fatalf("%dx%d slide %d step %d line %d: width %d: %q", ww, hh, si+1, step, i, ansi.StringWidth(l), l)
					}
				}
			}
			r.Overflow(si, ww, hh)
		}
	})
}
