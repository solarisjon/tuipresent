package render

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/solarisjon/tuipresent/internal/deck"
	"github.com/solarisjon/tuipresent/internal/transition"
)

func withHyperlinks(t *testing.T, on bool) {
	t.Helper()
	old := Hyperlinks
	Hyperlinks = on
	t.Cleanup(func() { Hyperlinks = old })
}

func linkDeck(t *testing.T, body string) *Renderer {
	t.Helper()
	d, err := deck.Parse("Links\n===\n\n" + body + "\n")
	if err != nil {
		t.Fatal(err)
	}
	return New(d)
}

func TestLinkMarkerRoundTrip(t *testing.T) {
	for _, n := range []int{0, 1, 7, 5000} {
		if got, ok := markerLink(linkMarker(n)); !ok || got != n {
			t.Errorf("markerLink(marker %d) = %d, %v", n, got, ok)
		}
	}
	if _, ok := markerLink("https://e.com"); ok {
		t.Error("real URL taken for a marker")
	}
}

func TestHideLinkURLs(t *testing.T) {
	src := "a [x](https://e.com) b [y [z]](https://f.org/a_(b) \"t\") `[c](https://no)` ![i](p.png) [r](rel.md) [h](#top)\n```\n[k](https://no)\n```"
	got, urls := hideLinkURLs(src)
	want := []string{"https://e.com", "https://f.org/a_(b)", ""}
	if len(urls) != len(want) {
		t.Fatalf("urls = %q, want %q", urls, want)
	}
	for i := range want {
		if urls[i] != want[i] {
			t.Errorf("urls[%d] = %q, want %q", i, urls[i], want[i])
		}
	}
	for _, keep := range []string{"`[c](https://no)`", "![i](p.png)", "[h](#top)", "[k](https://no)", "[y [z]]("} {
		if !strings.Contains(got, keep) {
			t.Errorf("rewritten source lost %q:\n%s", keep, got)
		}
	}
	if strings.Contains(got, "https://e.com") || strings.Contains(got, "rel.md") {
		t.Errorf("destinations not hidden:\n%s", got)
	}
}

// balanced reports whether every OSC 8 open in s is closed before the end.
func balanced(s string) bool {
	open := false
	for _, seg := range strings.Split(s, "\x1b]8;")[1:] {
		semi := strings.Index(seg, ";")
		end := strings.IndexAny(seg, "\x07\x1b")
		if semi < 0 || end < semi {
			return false
		}
		open = end > semi+1
	}
	return !open
}

func TestLinkHyperlinkMode(t *testing.T) {
	withHyperlinks(t, true)
	r := linkDeck(t, "See [the docs](https://example.com/docs), then [rel](other.md).")
	f := r.Frame(0, 0, 100, 30)
	checkFrame(t, f, 100, 30)
	plain := ansi.Strip(f)
	if !strings.Contains(plain, "See the docs, then rel.") {
		t.Errorf("link text not shown alone:\n%s", plain)
	}
	if strings.Contains(plain, "example.com") || strings.Contains(plain, "other.md") {
		t.Errorf("URL still visible:\n%s", plain)
	}
	if strings.Count(f, "\x1b]8;id=") != 1 || !strings.Contains(f, ";https://example.com/docs\x07") {
		t.Errorf("want one OSC 8 link to the docs, got %q", f)
	}
	if !balanced(f) {
		t.Error("unbalanced OSC 8")
	}
	if strings.Contains(f, "\u2060") || strings.Contains(f, linkScheme) {
		t.Error("marker leaked into the frame")
	}
}

func TestLinkFallbackMode(t *testing.T) {
	withHyperlinks(t, false)
	r := linkDeck(t, "See [the docs](https://example.com/docs) now.")
	f := r.Frame(0, 0, 100, 30)
	checkFrame(t, f, 100, 30)
	if strings.Contains(f, "\x1b]8;") {
		t.Errorf("OSC 8 emitted without hyperlink support: %q", f)
	}
	if !strings.Contains(ansi.Strip(f), "the docs https://example.com/docs now.") {
		t.Errorf("fallback should show text then URL:\n%s", ansi.Strip(f))
	}
}

func TestLinkWrapAndWidth(t *testing.T) {
	withHyperlinks(t, true)
	body := "Some leading words, then [a link whose text is long enough to wrap onto several lines](https://example.com/a/very/long/url/that/would/not/fit/at/all/on/one/line), then a tail.\n\n- item with [link one](https://one.example) and [link two](https://two.example) in a list that is long enough to wrap around\n\n> quoted [q](https://q.example) text"
	for _, w := range [][2]int{{100, 30}, {80, 24}, {40, 24}} {
		r := linkDeck(t, body)
		f := r.Frame(0, 0, w[0], w[1])
		checkFrame(t, f, w[0], w[1])
		for i, l := range strings.Split(f, "\n") {
			if !balanced(l) {
				t.Errorf("%v line %d unbalanced: %q", w, i, l)
			}
		}
		if rows, cols := r.Overflow(0, w[0], w[1]); rows > 0 || cols > 0 {
			t.Errorf("%v: overflow %d rows %d cols", w, rows, cols)
		}
		// Wrapped link text is the same text as without hyperlinks.
		Hyperlinks = false
		g := New(r.deck).Frame(0, 0, w[0], w[1])
		Hyperlinks = true
		if strings.Contains(ansi.Strip(f), "example") {
			t.Errorf("%v: URL visible with hyperlinks on", w)
		}
		if !strings.Contains(ansi.Strip(g), "https://example.com") && w[0] >= 100 {
			t.Errorf("%v: fallback lost URL", w)
		}
	}
}

func TestLinkTransitionsStayBalanced(t *testing.T) {
	withHyperlinks(t, true)
	r := linkDeck(t, "Go to [the docs](https://example.com/docs) or [here](https://example.org) now.\n\n# Next\n\nMore [text](https://x.example) here")
	a := r.Frame(0, 0, 80, 24)
	b := strings.Replace(a, "\x1b]8;", "\x1b]8;", 1)
	r2 := linkDeck(t, "Other [slide](https://other.example) content")
	b = r2.Frame(0, 0, 80, 24)
	for _, k := range transition.Kinds {
		for _, rev := range []bool{false, true} {
			for _, tt := range []float64{0.1, 0.3, 0.5, 0.77, 0.95} {
				out := transition.Composite(k, a, b, 80, tt, rev)
				for i, l := range strings.Split(out, "\n") {
					if !balanced(l) {
						t.Fatalf("%s rev=%v t=%v line %d unbalanced: %q", k, rev, tt, i, l)
					}
				}
			}
		}
	}
	prev := linkDeck(t, "Go to").Frame(0, 0, 80, 24)
	for _, tt := range []float64{0.2, 0.6} {
		out := transition.FadeIn(prev, a, tt, r.Background(), r.Foreground())
		for i, l := range strings.Split(out, "\n") {
			if !balanced(l) {
				t.Fatalf("fade t=%v line %d unbalanced: %q", tt, i, l)
			}
		}
		if ansi.StringWidth(strings.Split(out, "\n")[5]) != 80 {
			t.Errorf("fade changed geometry")
		}
	}
}

func TestStripLinks(t *testing.T) {
	s := "a\x1b]8;id=1;https://e.com\x07b\x1b]8;;\x1b\\c\x1b[1md\x1b[m"
	if got := StripLinks(s); got != "ab"+"c\x1b[1md\x1b[m" {
		t.Errorf("StripLinks = %q", got)
	}
}
