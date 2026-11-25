package render

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	chromastyles "github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/x/ansi"

	"github.com/solarisjon/tuipresent/internal/deck"
	"github.com/solarisjon/tuipresent/internal/img"
	"github.com/solarisjon/tuipresent/internal/transition"
)

func TestBuiltinThemesValid(t *testing.T) {
	names := ThemeNames()
	if len(names) < 20 {
		t.Fatalf("only %d built-in themes", len(names))
	}
	seen := map[string]bool{}
	for _, n := range names {
		if seen[n] {
			t.Errorf("duplicate theme %q", n)
		}
		seen[n] = true
		th, ok := LookupTheme(n)
		if !ok || th.Name != n {
			t.Fatalf("LookupTheme(%q) = %+v, %v", n, th, ok)
		}
		if _, ok := chromastyles.Registry[th.CodeStyle]; !ok {
			t.Errorf("%s: chroma style %q is not registered", n, th.CodeStyle)
		}
		p := th.Palette
		for _, k := range paletteKeys {
			v := *k.get(&p)
			if h, ok := normalizeHex(v); !ok || h != v {
				t.Errorf("%s: %s = %q is not a normalized #rrggbb color", n, k.key, v)
			}
		}
		if light := strings.Contains(n, "light") || strings.HasSuffix(n, "-day") || strings.HasSuffix(n, "-latte") || strings.HasSuffix(n, "-dawn"); light != th.IsLight() {
			t.Errorf("%s: IsLight = %v", n, th.IsLight())
		}
	}
	for alias, target := range ThemeAliases() {
		if !seen[target] {
			t.Errorf("alias %q points at unknown theme %q", alias, target)
		}
		if seen[alias] {
			t.Errorf("alias %q shadows a built-in", alias)
		}
	}
}

func TestThemeLookupIsExact(t *testing.T) {
	for name, want := range map[string]string{
		"tokyo-night":       "tokyonight-storm",
		"Tokyo-Night":       "tokyonight-storm",
		" dracula ":         "dracula",
		"tokyonight-day":    "tokyonight-day",
		"catppuccin":        "catppuccin-mocha",
		"gruvbox":           "gruvbox-dark",
		"terminal-light":    "light",
		"tokyo-night-day":   "tokyonight-day",
		"solarized":         "solarized-dark",
		"rose-pine-dawn":    "rose-pine-dawn",
		"everforest":        "everforest-dark",
		"tokyonight-storm":  "tokyonight-storm",
		"catppuccin-frappe": "catppuccin-frappe",
	} {
		th, ok := LookupTheme(name)
		if !ok || th.Name != want {
			t.Errorf("LookupTheme(%q) = %q, %v; want %q", name, th.Name, ok, want)
		}
	}
	// No substring matching: these used to pick light or tokyo by accident.
	for _, name := range []string{"sunday", "my-light-theme", "tokyo", "draculaish"} {
		if th, ok := LookupTheme(name); ok {
			t.Errorf("LookupTheme(%q) = %q, want no match", name, th.Name)
		}
	}
}

func TestUnknownThemeFallsBackWithWarning(t *testing.T) {
	th, warns, err := ResolveTheme(deck.Theme{Name: "tokyonight-stomr"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if th.Name != "dark" {
		t.Errorf("fallback theme = %q, want dark", th.Name)
	}
	if len(warns) != 1 || !strings.Contains(warns[0], `unknown theme "tokyonight-stomr"`) || !strings.Contains(warns[0], "tokyonight-storm") {
		t.Errorf("warnings = %q", warns)
	}
	d, err := deck.Parse("---\ntheme: nope\n---\nhi\n")
	if err != nil {
		t.Fatal(err)
	}
	r := New(d)
	if r.Theme().Name != "dark" || len(r.ThemeWarnings()) != 1 || r.ThemeErr() != nil {
		t.Errorf("renderer: theme %q, warnings %q, err %v", r.Theme().Name, r.ThemeWarnings(), r.ThemeErr())
	}
	if _, warns, _ := ResolveTheme(deck.Theme{}, ""); len(warns) != 0 {
		t.Errorf("no theme set should not warn: %q", warns)
	}
}

func writeTheme(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, filepath.Dir(name)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCustomThemeFileExtendsAndOverrides(t *testing.T) {
	dir := t.TempDir()
	writeTheme(t, dir, "themes/brand.yaml", "extends: nord\naccent: \"#FF8800\"\ncode_background: \"#123\"\ncode_style: monokai\n")
	th, warns, err := ResolveTheme(deck.Theme{Path: "themes/brand.yaml", Overrides: map[string]string{"accent2": "#00ff00", "background": "#000000"}}, dir)
	if err != nil || len(warns) != 0 {
		t.Fatalf("err %v, warnings %q", err, warns)
	}
	nord, _ := LookupTheme("nord")
	p := th.Palette
	if th.Name != "brand" || th.CodeStyle != "monokai" {
		t.Errorf("name %q, code style %q", th.Name, th.CodeStyle)
	}
	if p.Accent != "#ff8800" || p.CodeBackground != "#112233" || p.Accent2 != "#00ff00" || p.Background != "#000000" {
		t.Errorf("overrides not applied: %+v", p)
	}
	if p.Heading != "#ff8800" {
		t.Errorf("heading should follow the overridden accent, got %s", p.Heading)
	}
	if p.Foreground != nord.Palette.Foreground || p.Link != nord.Palette.Link || p.Dim != nord.Palette.Dim {
		t.Errorf("unset keys should come from nord: %+v", p)
	}

	// Inline overrides on a built-in name.
	th, _, err = ResolveTheme(deck.Theme{Name: "dracula", Background: "paint", Overrides: map[string]string{"accent": "#88c0d0"}}, "")
	if err != nil || th.Name != "dracula" || th.Palette.Accent != "#88c0d0" || th.Palette.Bullet != "#88c0d0" || !th.Paint {
		t.Errorf("inline override: %+v, %v", th, err)
	}

	// A file without extends starts from dark.
	writeTheme(t, dir, "plain.yml", "accent: \"#abcdef\"\n")
	th, _, err = ResolveTheme(deck.Theme{Path: filepath.Join(dir, "plain.yml")}, "")
	dark, _ := LookupTheme("dark")
	if err != nil || th.Palette.Accent != "#abcdef" || th.Palette.Background != dark.Palette.Background {
		t.Errorf("plain file: %+v, %v", th, err)
	}
}

func TestCustomThemeErrors(t *testing.T) {
	dir := t.TempDir()
	writeTheme(t, dir, "bad-hex.yaml", "accent: \"#ggg\"\n")
	writeTheme(t, dir, "bare-hash.yaml", "accent: #88c0d0\n")
	writeTheme(t, dir, "bad-extends.yaml", "extends: nordic-ish\n")
	writeTheme(t, dir, "not-map.yaml", "- a\n- b\n")
	for _, tc := range []struct {
		spec deck.Theme
		want string
	}{
		{deck.Theme{Path: "bad-hex.yaml"}, `accent = "#ggg" is not a hex color`},
		{deck.Theme{Path: "bare-hash.yaml"}, "quote hex colors"},
		{deck.Theme{Path: "bad-extends.yaml"}, `extends unknown theme "nordic-ish"`},
		{deck.Theme{Path: "not-map.yaml"}, "must be a mapping"},
		{deck.Theme{Path: "missing.yaml"}, "theme file"},
		{deck.Theme{Name: "nord", Overrides: map[string]string{"accent": "88c0d0"}}, "not a hex color"},
	} {
		th, _, err := ResolveTheme(tc.spec, dir)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%+v: err %v, want %q", tc.spec, err, tc.want)
		}
		if th.Name != "dark" {
			t.Errorf("%+v: error should fall back to dark, got %q", tc.spec, th.Name)
		}
	}
	_, warns, err := ResolveTheme(deck.Theme{Name: "nord", Background: "bogus", Overrides: map[string]string{"code_style": "no-such-style", "acent": "#88c0d0", "extends": "dracula"}, Nested: []string{"override"}}, "")
	if err != nil || len(warns) != 5 || !strings.Contains(strings.Join(warns, "|"), `unknown key "acent"`) || !strings.Contains(strings.Join(warns, "|"), "extends only works in a theme file") {
		t.Errorf("want 5 warnings, got %q, %v", warns, err)
	}
	// A presenterm-style theme file (nested sections) renders its base with a warning.
	writeTheme(t, dir, "presenterm.yaml", "extends: nord\ndefault:\n  colors:\n    foreground: \"e5e5e5\"\n")
	th, warns, err := ResolveTheme(deck.Theme{Path: "presenterm.yaml"}, dir)
	if err != nil || len(warns) != 1 || th.Palette.Background != "#2e3440" {
		t.Errorf("presenterm file: %q, %v", warns, err)
	}
}

func exampleDecks(t *testing.T) []string {
	t.Helper()
	paths, _ := filepath.Glob("../../examples/*.md")
	var out []string
	for _, p := range paths {
		if filepath.Base(p) != "README.md" {
			out = append(out, p)
		}
	}
	return out
}

// cellBackgrounds reports, per cell of a line, whether it has an explicit
// background color.
func cellBackgrounds(line string) []bool {
	var out []bool
	p := ansi.NewParser()
	var state byte
	bg := false
	for len(line) > 0 {
		seq, width, n, ns := ansi.DecodeSequence(line, state, p)
		state, line = ns, line[n:]
		if width > 0 {
			for range width {
				out = append(out, bg)
			}
			continue
		}
		if ansi.HasCsiPrefix(seq) && ansi.Cmd(p.Command()) == 'm' {
			if _, reset := sgrResets(p.Params()); reset {
				bg = false
			}
			params := p.Params()
			for i := range params {
				if v := params[i].Param(0); v == 48 || (v >= 40 && v <= 47) || (v >= 100 && v <= 107) {
					bg = true
				}
			}
		}
	}
	return out
}

func checkPainted(t *testing.T, what, f string, w int) {
	t.Helper()
	for i, l := range strings.Split(f, "\n") {
		cells := cellBackgrounds(l)
		if len(cells) != w {
			t.Errorf("%s line %d: %d cells, want %d", what, i, len(cells), w)
			continue
		}
		for x, ok := range cells {
			if !ok {
				t.Errorf("%s line %d: cell %d has no background: %q", what, i, x, ansi.Strip(l))
				return
			}
		}
	}
}

func TestEveryThemeKeepsGeometry(t *testing.T) {
	if testing.Short() {
		t.Skip("renders every example in every theme")
	}
	for _, path := range exampleDecks(t) {
		for _, name := range ThemeNames() {
			t.Run(filepath.Base(path)+"/"+name, func(t *testing.T) {
				t.Parallel()
				everyThemeGeometry(t, path, name)
			})
		}
	}
}

func everyThemeGeometry(t *testing.T, path, name string) {
	for _, bg := range []string{"terminal", "paint"} {
		d, err := deck.Load(path)
		if err != nil {
			t.Fatal(err)
		}
		d.Meta.Theme = deck.Theme{Name: name, Background: bg}
		r := New(d)
		for _, sz := range [][2]int{{100, 30}, {80, 24}} {
			for i, s := range d.Slides {
				for st := range s.Steps {
					f := r.Frame(i, st, sz[0], sz[1])
					what := fmt.Sprintf("%s/%s/%s/%dx%d/slide%d.%d", filepath.Base(path), name, bg, sz[0], sz[1], i+1, st+1)
					checkFrame(t, f, sz[0], sz[1])
					if bg == "paint" {
						checkPainted(t, what, f, sz[0])
					}
					if t.Failed() {
						t.Fatalf("at %s", what)
					}
				}
			}
		}
	}
}

func paintedDeck(t *testing.T, src string) *Renderer {
	t.Helper()
	d, err := deck.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	d.Meta.Theme = deck.Theme{Name: "nord", Background: "paint"}
	return New(d)
}

func TestPaintSurvivesTransitionsAndFade(t *testing.T) {
	src := "A\n===\n\n* one\n\n```go\nfunc main() {} // c\n```\n\n<!-- pause -->\n\n| a | b |\n|---|---|\n| 1 | 2 |\n<!-- end_slide -->\nB\n===\n\n> quote `code` **bold**\n"
	r := paintedDeck(t, src)
	w, h := 70, 22
	a, b := r.Frame(0, 1, w, h), r.Frame(1, 0, w, h)
	for _, k := range []transition.Kind{transition.Slide, transition.Wipe, transition.Cover, transition.Dissolve} {
		for _, rev := range []bool{false, true} {
			for _, tt := range []float64{0.13, 0.5, 0.87} {
				f := transition.Composite(k, a, b, w, tt, rev)
				checkFrame(t, f, w, h)
				checkPainted(t, fmt.Sprintf("%s rev=%v t=%.2f", k, rev, tt), f, w)
			}
		}
	}
	bg, _ := r.Background().(color.Color)
	f := transition.FadeIn(r.Frame(0, 0, w, h), r.Frame(0, 1, w, h), 0.4, bg, r.Foreground())
	checkFrame(t, f, w, h)
	checkPainted(t, "fade", f, w)
}

func TestPaintKeepsKittyPlaceholderIDs(t *testing.T) {
	line := "ab" + strings.Join(img.Placeholder(0x123456, 3, 1), "") + "\x1b[0mcd"
	out := paint(line, "#d8dee9", "#2e3440")
	if ansi.StringWidth(out) != ansi.StringWidth(line) {
		t.Fatalf("width changed: %d -> %d", ansi.StringWidth(line), ansi.StringWidth(out))
	}
	// Every placeholder cell must still be drawn in the id color.
	p := ansi.NewParser()
	var state byte
	var fg color.Color
	s := out
	cells := 0
	for len(s) > 0 {
		seq, width, n, ns := ansi.DecodeSequence(s, state, p)
		state, s = ns, s[n:]
		if width > 0 {
			if strings.ContainsRune(seq, '\U0010EEEE') {
				cells++
				r, g, b, _ := fg.RGBA()
				if r>>8 != 0x12 || g>>8 != 0x34 || b>>8 != 0x56 {
					t.Fatalf("placeholder fg = %v", fg)
				}
			}
			continue
		}
		if ansi.HasCsiPrefix(seq) && ansi.Cmd(p.Command()) == 'm' {
			params := p.Params()
			for i := 0; i < len(params); i++ {
				if params[i].Param(0) == 38 {
					ansi.ReadStyleColor(params[i:], &fg)
				}
			}
		}
	}
	if cells != 3 {
		t.Fatalf("found %d placeholder cells, want 3", cells)
	}
	checkPainted(t, "kitty", out, ansi.StringWidth(line))
}

func TestCodeBlockPanel(t *testing.T) {
	d, err := deck.Parse("```go\nx := 1\n```\n")
	if err != nil {
		t.Fatal(err)
	}
	d.Meta.Theme = deck.Theme{Name: "nord"}
	r := New(d)
	f := r.Frame(0, 0, 60, 10)
	checkFrame(t, f, 60, 10)
	nord, _ := LookupTheme("nord")
	if !strings.Contains(f, sgrColor(48, nord.Palette.CodeBackground)) {
		t.Error("code block not drawn on the code background")
	}
}

func TestSwatchShowsTheme(t *testing.T) {
	th, _ := LookupTheme("dracula")
	s := Swatch(th)
	if !strings.Contains(s, sgrColor(48, "#282a36")) || !strings.Contains(ansi.Strip(s), "code") {
		t.Errorf("swatch = %q", s)
	}
}
