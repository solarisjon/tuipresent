package render

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/alecthomas/chroma/v2"
	chromastyles "github.com/alecthomas/chroma/v2/styles"
	"github.com/lucasb-eyer/go-colorful"
	"gopkg.in/yaml.v3"

	"tuipresent/internal/deck"
)

// Palette is every color a theme sets, as #rrggbb. Empty derived fields
// (Heading, Subheading, Link, Bullet, Border, Quote) follow the core colors,
// so overriding accent also recolors the headings that use it.
type Palette struct {
	Background     string
	Foreground     string
	Dim            string // footer, author line, comments-like text
	Accent         string // gradient start, H1, progress bar
	Accent2        string // gradient end, H2-H6
	Heading        string // H1 (default accent)
	Subheading     string // H2-H6 (default accent2)
	Link           string // default accent
	Code           string // inline code text
	CodeBackground string // inline code and fenced code panels
	Quote          string // blockquote text and bar (default dim)
	Border         string // table borders and horizontal rules (default dim)
	Bullet         string // list bullets and numbers (default accent)
}

// Theme is a resolved palette plus the chroma style used for code.
type Theme struct {
	Name      string
	Palette   Palette
	CodeStyle string  // chroma style name
	Paint     bool    // paint the theme background into every cell of the frame
	raw       Palette // before withDefaults, so overrides of core colors carry to derived ones
}

// paletteKeys maps the YAML key of each palette color to its field.
var paletteKeys = []struct {
	key string
	get func(*Palette) *string
}{
	{"background", func(p *Palette) *string { return &p.Background }},
	{"foreground", func(p *Palette) *string { return &p.Foreground }},
	{"dim", func(p *Palette) *string { return &p.Dim }},
	{"accent", func(p *Palette) *string { return &p.Accent }},
	{"accent2", func(p *Palette) *string { return &p.Accent2 }},
	{"heading", func(p *Palette) *string { return &p.Heading }},
	{"subheading", func(p *Palette) *string { return &p.Subheading }},
	{"link", func(p *Palette) *string { return &p.Link }},
	{"code", func(p *Palette) *string { return &p.Code }},
	{"code_background", func(p *Palette) *string { return &p.CodeBackground }},
	{"quote", func(p *Palette) *string { return &p.Quote }},
	{"border", func(p *Palette) *string { return &p.Border }},
	{"bullet", func(p *Palette) *string { return &p.Bullet }},
}

// PaletteKeys lists the color keys a custom theme may set.
func PaletteKeys() []string {
	out := make([]string, len(paletteKeys))
	for i, k := range paletteKeys {
		out[i] = k.key
	}
	return out
}

// ThemeKeys lists every key a theme file, or the front-matter theme mapping
// besides name, path and background, may set: the palette keys and code_style.
func ThemeKeys() []string {
	return append(PaletteKeys(), "code_style")
}

type builtin struct {
	name  string
	style string // chroma style
	p     Palette
}

// builtins are the official palettes of each scheme. Order is the -themes order.
var builtins = []builtin{
	{"dark", "onedark", Palette{Background: "#1a1b26", Foreground: "#d0d0d0", Dim: "#5c6370", Accent: "#7aa2f7", Accent2: "#bb9af7", Code: "#9ece6a", CodeBackground: "#24283b"}},
	{"light", "github", Palette{Background: "#e1e2e7", Foreground: "#3760bf", Dim: "#8990b3", Accent: "#2e7de9", Accent2: "#9854f1", Code: "#587539", CodeBackground: "#d0d5e3"}},
	{"tokyonight-storm", "tokyonight-storm", Palette{Background: "#24283b", Foreground: "#c0caf5", Dim: "#565f89", Accent: "#7aa2f7", Accent2: "#bb9af7", Code: "#9ece6a", CodeBackground: "#1f2335", Quote: "#a9b1d6", Border: "#3b4261"}},
	{"tokyonight-night", "tokyonight-night", Palette{Background: "#1a1b26", Foreground: "#c0caf5", Dim: "#565f89", Accent: "#7aa2f7", Accent2: "#bb9af7", Code: "#9ece6a", CodeBackground: "#16161e", Quote: "#a9b1d6", Border: "#3b4261"}},
	{"tokyonight-moon", "tokyonight-moon", Palette{Background: "#222436", Foreground: "#c8d3f5", Dim: "#636da6", Accent: "#82aaff", Accent2: "#c099ff", Code: "#c3e88d", CodeBackground: "#1e2030", Quote: "#828bb8", Border: "#3b4261"}},
	{"tokyonight-day", "tokyonight-day", Palette{Background: "#e1e2e7", Foreground: "#3760bf", Dim: "#848cb5", Accent: "#2e7de9", Accent2: "#9854f1", Code: "#587539", CodeBackground: "#d0d5e3", Quote: "#6172b0", Border: "#a8aecb"}},
	{"catppuccin-mocha", "catppuccin-mocha", Palette{Background: "#1e1e2e", Foreground: "#cdd6f4", Dim: "#7f849c", Accent: "#cba6f7", Accent2: "#89b4fa", Code: "#a6e3a1", CodeBackground: "#181825", Quote: "#bac2de", Border: "#45475a", Bullet: "#f5c2e7"}},
	{"catppuccin-macchiato", "catppuccin-macchiato", Palette{Background: "#24273a", Foreground: "#cad3f5", Dim: "#8087a2", Accent: "#c6a0f6", Accent2: "#8aadf4", Code: "#a6da95", CodeBackground: "#1e2030", Quote: "#b8c0e0", Border: "#494d64", Bullet: "#f5bde6"}},
	{"catppuccin-frappe", "catppuccin-frappe", Palette{Background: "#303446", Foreground: "#c6d0f5", Dim: "#838ba7", Accent: "#ca9ee6", Accent2: "#8caaee", Code: "#a6d189", CodeBackground: "#292c3c", Quote: "#b5bfe2", Border: "#51576d", Bullet: "#f4b8e4"}},
	{"catppuccin-latte", "catppuccin-latte", Palette{Background: "#eff1f5", Foreground: "#4c4f69", Dim: "#8c8fa1", Accent: "#8839ef", Accent2: "#1e66f5", Code: "#40a02b", CodeBackground: "#e6e9ef", Quote: "#5c5f77", Border: "#bcc0cc", Bullet: "#ea76cb"}},
	{"gruvbox-dark", "gruvbox", Palette{Background: "#282828", Foreground: "#ebdbb2", Dim: "#928374", Accent: "#fabd2f", Accent2: "#fe8019", Link: "#83a598", Code: "#b8bb26", CodeBackground: "#3c3836", Quote: "#d5c4a1", Border: "#504945", Bullet: "#8ec07c"}},
	{"gruvbox-light", "gruvbox-light", Palette{Background: "#fbf1c7", Foreground: "#3c3836", Dim: "#928374", Accent: "#af3a03", Accent2: "#b57614", Link: "#076678", Code: "#79740e", CodeBackground: "#ebdbb2", Quote: "#504945", Border: "#d5c4a1", Bullet: "#427b58"}},
	{"dracula", "dracula", Palette{Background: "#282a36", Foreground: "#f8f8f2", Dim: "#6272a4", Accent: "#bd93f9", Accent2: "#ff79c6", Link: "#8be9fd", Code: "#50fa7b", CodeBackground: "#44475a", Border: "#44475a"}},
	{"nord", "nord", Palette{Background: "#2e3440", Foreground: "#d8dee9", Dim: "#616e88", Accent: "#88c0d0", Accent2: "#b48ead", Link: "#81a1c1", Code: "#a3be8c", CodeBackground: "#3b4252", Quote: "#e5e9f0", Border: "#4c566a", Bullet: "#81a1c1"}},
	{"rose-pine", "rose-pine", Palette{Background: "#191724", Foreground: "#e0def4", Dim: "#6e6a86", Accent: "#c4a7e7", Accent2: "#ebbcba", Link: "#9ccfd8", Code: "#f6c177", CodeBackground: "#1f1d2e", Quote: "#908caa", Border: "#403d52", Bullet: "#eb6f92"}},
	{"rose-pine-dawn", "rose-pine-dawn", Palette{Background: "#faf4ed", Foreground: "#575279", Dim: "#9893a5", Accent: "#907aa9", Accent2: "#d7827e", Link: "#56949f", Code: "#ea9d34", CodeBackground: "#f2e9e1", Quote: "#797593", Border: "#dfdad9", Bullet: "#b4637a"}},
	{"solarized-dark", "solarized-dark", Palette{Background: "#002b36", Foreground: "#839496", Dim: "#586e75", Accent: "#268bd2", Accent2: "#d33682", Link: "#2aa198", Code: "#859900", CodeBackground: "#073642", Quote: "#93a1a1", Border: "#073642", Bullet: "#b58900"}},
	{"solarized-light", "solarized-light", Palette{Background: "#fdf6e3", Foreground: "#657b83", Dim: "#93a1a1", Accent: "#268bd2", Accent2: "#d33682", Link: "#2aa198", Code: "#859900", CodeBackground: "#eee8d5", Quote: "#586e75", Border: "#eee8d5", Bullet: "#b58900"}},
	{"everforest-dark", "evergarden", Palette{Background: "#2d353b", Foreground: "#d3c6aa", Dim: "#859289", Accent: "#a7c080", Accent2: "#83c092", Link: "#7fbbb3", Code: "#dbbc7f", CodeBackground: "#343f44", Quote: "#9da9a0", Border: "#475258", Bullet: "#e69875"}},
	{"kanagawa", "kanagawa", Palette{Background: "#1f1f28", Foreground: "#dcd7ba", Dim: "#727169", Accent: "#7e9cd8", Accent2: "#957fb8", Link: "#7fb4ca", Code: "#98bb6c", CodeBackground: "#2a2a37", Quote: "#c8c093", Border: "#54546d", Bullet: "#ffa066"}},
}

// aliases keep names from older decks and common spellings working.
var aliases = map[string]string{
	"tokyo-night":       "tokyonight-storm",
	"tokyonight":        "tokyonight-storm",
	"tokyo-night-storm": "tokyonight-storm",
	"tokyo-night-night": "tokyonight-night",
	"tokyo-night-moon":  "tokyonight-moon",
	"tokyo-night-day":   "tokyonight-day",
	"tokyo-night-light": "tokyonight-day",
	"catppuccin":        "catppuccin-mocha",
	"gruvbox":           "gruvbox-dark",
	"rosepine":          "rose-pine",
	"rose-pine-main":    "rose-pine",
	"rosepine-dawn":     "rose-pine-dawn",
	"solarized":         "solarized-dark",
	"everforest":        "everforest-dark",
	"kanagawa-wave":     "kanagawa",
	"terminal-dark":     "dark",
	"terminal-light":    "light",
	"default":           "dark",
}

// ThemeNames returns the canonical built-in theme names in display order.
func ThemeNames() []string {
	out := make([]string, len(builtins))
	for i, b := range builtins {
		out[i] = b.name
	}
	return out
}

// ThemeAliases returns the accepted alternative names, each mapped to its theme.
func ThemeAliases() map[string]string {
	out := make(map[string]string, len(aliases))
	for k, v := range aliases {
		out[k] = v
	}
	return out
}

// LookupTheme returns the built-in theme for name or an alias of it.
// Matching is exact apart from case and surrounding space.
func LookupTheme(name string) (Theme, bool) {
	n := strings.ToLower(strings.TrimSpace(name))
	if a, ok := aliases[n]; ok {
		n = a
	}
	for _, b := range builtins {
		if b.name == n {
			return Theme{Name: b.name, Palette: b.p.withDefaults(), CodeStyle: b.style, raw: b.p}, true
		}
	}
	return Theme{}, false
}

func defaultTheme() Theme {
	t, _ := LookupTheme("dark")
	return t
}

// withDefaults fills the derived colors from the core ones.
func (p Palette) withDefaults() Palette {
	def := func(v *string, from string) {
		if *v == "" {
			*v = from
		}
	}
	def(&p.Heading, p.Accent)
	def(&p.Subheading, p.Accent2)
	def(&p.Link, p.Accent)
	def(&p.Bullet, p.Accent)
	def(&p.Border, p.Dim)
	def(&p.Quote, p.Dim)
	def(&p.Code, p.Accent2)
	if p.CodeBackground == "" {
		p.CodeBackground = mix(p.Background, p.Foreground, 0.08)
	}
	return p
}

// IsLight reports whether the theme is meant for a light background.
func (t Theme) IsLight() bool {
	c, err := colorful.Hex(t.Palette.Background)
	if err != nil {
		return false
	}
	l, _, _ := c.Lab()
	return l > 0.6
}

// Swatch is a one-line truecolor sample of a theme: text, dim text, the
// accent gradient, a bullet, a link and inline code, on its background.
func Swatch(t Theme) string {
	p := t.Palette
	fg := func(hex, s string) string { return sgrColor(38, hex) + s }
	var sb strings.Builder
	sb.WriteString(fg(p.Foreground, " Text "))
	sb.WriteString(fg(p.Dim, "dim "))
	a, _ := colorful.Hex(p.Accent)
	b, _ := colorful.Hex(p.Accent2)
	const n = 10
	for i := range n {
		sb.WriteString(fg(a.BlendLuv(b, float64(i)/(n-1)).Clamped().Hex(), "━"))
	}
	sb.WriteString(fg(p.Bullet, " •"))
	sb.WriteString(fg(p.Link, " link "))
	sb.WriteString(sgrColor(48, p.CodeBackground) + fg(p.Code, " code ") + sgrColor(48, p.Background) + " ")
	return paint(sb.String(), p.Foreground, p.Background)
}

func mix(a, b string, t float64) string {
	ca, err1 := colorful.Hex(a)
	cb, err2 := colorful.Hex(b)
	if err1 != nil || err2 != nil {
		return a
	}
	return ca.BlendLab(cb, t).Clamped().Hex()
}

var hexRe = regexp.MustCompile(`^#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`)

// normalizeHex validates a #rgb or #rrggbb color and returns it as #rrggbb.
func normalizeHex(v string) (string, bool) {
	v = strings.TrimSpace(v)
	if !hexRe.MatchString(v) {
		return "", false
	}
	v = strings.ToLower(v)
	if len(v) == 4 {
		v = "#" + strings.Repeat(v[1:2], 2) + strings.Repeat(v[2:3], 2) + strings.Repeat(v[3:4], 2)
	}
	return v, true
}

// themeFile is the custom theme file format.
type themeFile struct {
	Extends string `yaml:"extends"`
	Values  map[string]string
	Nested  []string // keys with non-scalar values, e.g. a presenterm theme file
}

func (f *themeFile) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind != yaml.MappingNode {
		return fmt.Errorf("line %d: a theme file must be a mapping of keys to values", n.Line)
	}
	f.Values = map[string]string{}
	for i := 0; i+1 < len(n.Content); i += 2 {
		k, v := n.Content[i].Value, n.Content[i+1]
		if v.Kind != yaml.ScalarNode {
			f.Nested = append(f.Nested, k)
			continue
		}
		if k == "extends" {
			f.Extends = v.Value
		} else {
			f.Values[k] = v.Value
		}
	}
	return nil
}

// ResolveTheme turns a deck's theme setting into a Theme. dir is the deck's
// directory, against which spec.Path resolves. Unknown names and other
// recoverable problems come back as warnings with a usable theme; an invalid
// custom theme (unreadable file, bad color, unknown key) is an error, and the
// returned theme is then dark.
func ResolveTheme(spec deck.Theme, dir string) (Theme, []string, error) {
	var warns []string
	base := defaultTheme()
	vals := map[string]string{}
	src := "front matter theme"

	if spec.Path != "" {
		path := spec.Path
		if !filepath.IsAbs(path) && dir != "" {
			path = filepath.Join(dir, path)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return defaultTheme(), nil, fmt.Errorf("theme file: %w", err)
		}
		var f themeFile
		if err := yaml.Unmarshal(b, &f); err != nil {
			return defaultTheme(), nil, fmt.Errorf("theme file %s: %w", spec.Path, err)
		}
		if f.Extends != "" {
			t, ok := LookupTheme(f.Extends)
			if !ok {
				return defaultTheme(), nil, fmt.Errorf("theme file %s: extends unknown theme %q (run tuipresent -themes for the list)", spec.Path, f.Extends)
			}
			base = t
		}
		base.Name = strings.TrimSuffix(filepath.Base(spec.Path), filepath.Ext(spec.Path))
		if err := applyValues(&base, f.Values, "theme file "+spec.Path, &warns); err != nil {
			return defaultTheme(), nil, err
		}
		if len(f.Nested) > 0 {
			warns = append(warns, fmt.Sprintf("theme file %s: ignoring %s (not a tuipresent theme key; presenterm theme files aren't supported)", spec.Path, strings.Join(f.Nested, ", ")))
		}
		if spec.Name != "" {
			warns = append(warns, fmt.Sprintf("theme has both name %q and path %q; using the path", spec.Name, spec.Path))
		}
	} else if spec.Name != "" {
		t, ok := LookupTheme(spec.Name)
		if !ok {
			warns = append(warns, unknownThemeWarning(spec.Name))
		} else {
			base = t
		}
	}
	for k, v := range spec.Overrides {
		vals[k] = v
	}
	if len(spec.Nested) > 0 {
		warns = append(warns, fmt.Sprintf("theme: ignoring %s (tuipresent theme keys are plain colors)", strings.Join(spec.Nested, ", ")))
	}
	if err := applyValues(&base, vals, src, &warns); err != nil {
		return defaultTheme(), nil, err
	}
	switch strings.ToLower(strings.TrimSpace(spec.Background)) {
	case "", "terminal":
	case "paint":
		base.Paint = true
	default:
		warns = append(warns, fmt.Sprintf("theme background %q is not terminal or paint; using terminal", spec.Background))
	}
	return base, warns, nil
}

// applyValues validates and applies palette overrides plus code_style.
func applyValues(t *Theme, vals map[string]string, src string, warns *[]string) error {
	keys := make([]string, 0, len(vals))
	for k := range vals {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	raw := t.raw
	for _, k := range keys {
		v := vals[k]
		if k == "extends" {
			*warns = append(*warns, fmt.Sprintf("%s: extends only works in a theme file; use name: to pick the base theme", src))
			continue
		}
		if k == "code_style" {
			s := strings.TrimSpace(v)
			if _, ok := chromastyles.Registry[s]; ok {
				t.CodeStyle = s
			} else {
				*warns = append(*warns, fmt.Sprintf("%s: unknown code_style %q; keeping %q", src, v, t.CodeStyle))
			}
			continue
		}
		var field *string
		for _, pk := range paletteKeys {
			if pk.key == k {
				field = pk.get(&raw)
			}
		}
		if field == nil {
			*warns = append(*warns, fmt.Sprintf("%s: ignoring unknown key %q (valid: %s)", src, k, strings.Join(ThemeKeys(), ", ")))
			continue
		}
		if strings.TrimSpace(v) == "" {
			return fmt.Errorf("%s: %s is empty; quote hex colors in YAML (%s: \"#88c0d0\"), since a bare # starts a comment", src, k, k)
		}
		hex, ok := normalizeHex(v)
		if !ok {
			return fmt.Errorf("%s: %s = %q is not a hex color like \"#88c0d0\" or \"#8cd\"", src, k, v)
		}
		*field = hex
	}
	t.raw, t.Palette = raw, raw.withDefaults()
	return nil
}

func unknownThemeWarning(name string) string {
	msg := fmt.Sprintf("unknown theme %q, using dark", name)
	if s := suggestTheme(name); s != "" {
		msg += fmt.Sprintf(" (did you mean %s?)", s)
	}
	return msg + "; tuipresent -themes lists them"
}

// suggestTheme returns the closest built-in name, or "" if none is close.
func suggestTheme(name string) string {
	n := strings.ToLower(strings.TrimSpace(name))
	best, bestD := "", 4
	for _, b := range builtins {
		if strings.HasPrefix(b.name, n) && len(n) >= 3 {
			return b.name
		}
		if d := levenshtein(n, b.name); d < bestD {
			best, bestD = b.name, d
		}
	}
	return best
}

func levenshtein(a, b string) int {
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur := make([]int, len(b)+1)
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(b)]
}

// kanagawa has no chroma style in the chroma version we build against; this
// one follows kanagawa.nvim's syntax colors. A newer chroma's own style wins.
func init() {
	if _, ok := chromastyles.Registry["kanagawa"]; ok {
		return
	}
	chromastyles.Register(chroma.MustNewStyle("kanagawa", chroma.StyleEntries{
		chroma.Background:          "#dcd7ba bg:#1f1f28",
		chroma.Text:                "#dcd7ba",
		chroma.Error:               "#e82424",
		chroma.Comment:             "italic #727169",
		chroma.CommentPreproc:      "#e46876",
		chroma.Keyword:             "#957fb8",
		chroma.KeywordConstant:     "#ffa066",
		chroma.KeywordType:         "#7aa89f",
		chroma.Operator:            "#c0a36e",
		chroma.OperatorWord:        "#957fb8",
		chroma.Punctuation:         "#9cabca",
		chroma.Name:                "#e6c384",
		chroma.NameBuiltin:         "#7fb4ca",
		chroma.NameFunction:        "#7e9cd8",
		chroma.NameClass:           "#7aa89f",
		chroma.NameConstant:        "#ffa066",
		chroma.NameDecorator:       "#e46876",
		chroma.NameTag:             "#957fb8",
		chroma.NameAttribute:       "#e6c384",
		chroma.NameVariable:        "#dcd7ba",
		chroma.NameNamespace:       "#7aa89f",
		chroma.LiteralString:       "#98bb6c",
		chroma.LiteralStringEscape: "#ff5d62",
		chroma.LiteralNumber:       "#d27e99",
		chroma.GenericDeleted:      "#c34043",
		chroma.GenericInserted:     "#76946a",
		chroma.GenericHeading:      "bold #7e9cd8",
		chroma.GenericSubheading:   "#957fb8",
		chroma.GenericEmph:         "italic",
		chroma.GenericStrong:       "bold",
	}))
}
