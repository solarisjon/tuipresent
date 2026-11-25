// Package docscheck fails the build when example decks or docs drift from the code.
package docscheck

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/solarisjon/tuipresent/internal/deck"
	"github.com/solarisjon/tuipresent/internal/render"
)

const root = "../.."

// docs must each mention every directive, transition kind, CLI flag and
// built-in theme: the human reference, the agent spec, the agent guide and
// the standalone agent skill.
var docs = []string{"docs/guide/reference.md", "docs/agent/LANGUAGE.md", "docs/AGENT_GUIDE.md", "docs/agent-skill/SKILL.md"}

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
	return out
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

// TestDocsCoverThemes requires every built-in theme name in the docs and in
// the theme gallery deck, so adding a theme to the code forces a doc update.
func TestDocsCoverThemes(t *testing.T) {
	names := render.ThemeNames()
	if len(names) == 0 {
		t.Fatal("no themes found")
	}
	for _, doc := range append(docs, "examples/themes.md") {
		text := read(t, doc)
		for _, n := range names {
			if !regexp.MustCompile(`(^|[^a-z-])` + regexp.QuoteMeta(n) + `($|[^a-z-])`).MatchString(text) {
				t.Errorf("%s does not mention theme %q", doc, n)
			}
		}
	}
}

// requireMentions requires each token as a whole word in every doc, so
// "-theme" is not satisfied by "-themes" nor "column" by "column_layout".
func requireMentions(t *testing.T, items []string, token func(string) string) {
	t.Helper()
	for _, doc := range docs {
		text := read(t, doc)
		for _, it := range items {
			re := regexp.MustCompile(`(^|[^A-Za-z0-9_-])` + regexp.QuoteMeta(token(it)) + `($|[^A-Za-z0-9_-])`)
			if !re.MatchString(text) {
				t.Errorf("%s does not mention %q", doc, token(it))
			}
		}
	}
}
