// Package docscheck fails the build when example decks or docs drift from the code.
package docscheck

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"tuipresent/internal/deck"
	"tuipresent/internal/render"
)

const root = "../.."

var docs = []string{"README.md", "docs/AGENT_GUIDE.md", "docs/agent-skill/SKILL.md"}

func read(t *testing.T, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(b)
}

func decks(t *testing.T) []string {
	t.Helper()
	paths, _ := filepath.Glob(filepath.Join(root, "examples", "*.md"))
	var out []string
	for _, p := range paths {
		if filepath.Base(p) != "README.md" {
			out = append(out, p)
		}
	}
	root, _ := filepath.Glob(filepath.Join(root, "*-demo.md"))
	return append(out, root...)
}

func TestDecksFitStandardScreens(t *testing.T) {
	for _, p := range decks(t) {
		d, err := deck.Load(p)
		if err != nil {
			t.Errorf("%s: %v", p, err)
			continue
		}
		r := render.New(d)
		for _, sz := range [][2]int{{100, 30}, {80, 24}} {
			for i := range d.Slides {
				if rows, cols := r.Overflow(i, sz[0], sz[1]); rows > 0 || cols > 0 {
					t.Errorf("%s slide %d overflows %dx%d by %d rows, %d cols", filepath.Base(p), i+1, sz[0], sz[1], rows, cols)
				}
			}
		}
	}
}

func TestDocsCoverDirectives(t *testing.T) {
	src := read(t, "internal/deck/parse.go")
	fn := src[strings.Index(src, "func (p *slideParser) directive"):]
	fn = fn[:strings.Index(fn, "\n}\n")]
	names := []string{"end_slide"}
	for _, m := range regexp.MustCompile(`case "([a-z_]+)"`).FindAllStringSubmatch(fn, -1) {
		names = append(names, m[1])
	}
	requireMentions(t, names, func(n string) string { return n })
}

func TestDocsCoverTransitions(t *testing.T) {
	src := read(t, "internal/transition/transition.go")
	var kinds []string
	for _, m := range regexp.MustCompile(`Kind = "([a-z]+)"`).FindAllStringSubmatch(src, -1) {
		kinds = append(kinds, m[1])
	}
	if len(kinds) == 0 {
		t.Fatal("no transition kinds found")
	}
	requireMentions(t, kinds, func(k string) string { return k })
}

func TestDocsCoverFlags(t *testing.T) {
	src := read(t, "cmd/tuipresent/main.go")
	var flags []string
	for _, m := range regexp.MustCompile(`flag\.(?:Bool|Int|String)\("([a-z]+)"`).FindAllStringSubmatch(src, -1) {
		flags = append(flags, m[1])
	}
	requireMentions(t, flags, func(f string) string { return "-" + f })
}

func requireMentions(t *testing.T, items []string, token func(string) string) {
	t.Helper()
	for _, doc := range docs {
		text := read(t, doc)
		for _, it := range items {
			if !strings.Contains(text, token(it)) {
				t.Errorf("%s does not mention %q", doc, token(it))
			}
		}
	}
}
