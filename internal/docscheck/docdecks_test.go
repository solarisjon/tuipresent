package docscheck

import (
	"path/filepath"
	"strings"
	"testing"

	"tuipresent/internal/deck"
	"tuipresent/internal/render"
	"tuipresent/internal/report"
)

// TestDocDecksFit loads every complete deck shown in the user and agent
// docs (a markdown code block starting with front matter) and requires it to
// load without warnings and fit 100x30 and 80x24, as the docs promise.
func TestDocDecksFit(t *testing.T) {
	paths, _ := filepath.Glob(filepath.Join(root, "docs", "guide", "*.md"))
	docsWithDecks := []string{"README.md", "docs/AGENT_GUIDE.md", "docs/agent-skill/SKILL.md"}
	for _, p := range paths {
		rel, _ := filepath.Rel(root, p)
		docsWithDecks = append(docsWithDecks, filepath.ToSlash(rel))
	}
	n := 0
	for _, doc := range docsWithDecks {
		for _, src := range markdownDecks(read(t, doc)) {
			d, err := deck.Parse(src)
			if err != nil && err.Error() == "deck has no slides" {
				continue // a front matter snippet, not a deck
			}
			n++
			t.Logf("%s: deck starting %q", doc, firstLines(src))
			if err != nil {
				t.Errorf("%s: deck starting %q: %v", doc, firstLines(src), err)
				continue
			}
			d.Dir = filepath.Join(root, filepath.Dir(doc))
			r := render.New(d)
			if w := report.Warnings(d, r, ""); len(w) > 0 {
				t.Errorf("%s: deck starting %q warns: %q", doc, firstLines(src), w)
			}
			for _, sz := range [][2]int{{100, 30}, {80, 24}} {
				for i := range d.Slides {
					if rows, cols := r.Overflow(i, sz[0], sz[1]); rows > 0 || cols > 0 {
						t.Errorf("%s: deck starting %q: slide %d overflows %dx%d by %d rows, %d cols", doc, firstLines(src), i+1, sz[0], sz[1], rows, cols)
					}
				}
			}
		}
	}
	if n == 0 {
		t.Fatal("no decks found in the docs")
	}
}

func firstLines(src string) string {
	lines := strings.SplitN(src, "\n", 4)
	return strings.Join(lines[:min(3, len(lines))], "\\n")
}

// markdownDecks returns the bodies of top-level ```markdown / ````markdown
// blocks that start with a front matter line.
func markdownDecks(md string) []string {
	var out []string
	lines := strings.Split(md, "\n")
	for i := 0; i < len(lines); i++ {
		marker, info, ok := deck.FenceOpen(lines[i])
		if !ok {
			continue
		}
		j := i + 1
		for j < len(lines) && !deck.ClosesFence(marker, lines[j]) {
			j++
		}
		if strings.TrimSpace(info) == "markdown" && i+1 < j && lines[i+1] == "---" {
			out = append(out, strings.Join(lines[i+1:j], "\n")+"\n")
		}
		i = j
	}
	return out
}
