// Package deck parses presenterm-flavoured markdown into slides.
package deck

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

type Meta struct {
	Title      string `yaml:"title"`
	SubTitle   string `yaml:"sub_title"`
	Author     string `yaml:"author"`
	Date       string `yaml:"date"`
	Theme      Theme  `yaml:"theme"`
	Transition string `yaml:"transition"`
	Fade       string `yaml:"fade"`     // off/false disables the pause reveal fade deck-wide
	Progress   string `yaml:"progress"` // off/false disables the terminal tab progress bar
	TitleStyle string `yaml:"title_style"`
	TitleSize  string `yaml:"title_size"` // a string so "huge" warns instead of failing the YAML decode
}

// TitleStyles are the values of the title_style directive and front-matter key.
var TitleStyles = []string{"rule", "curly", "frame"}

// ValidTitleStyle reports whether v (case-insensitive) is a title style.
func ValidTitleStyle(v string) bool {
	for _, s := range TitleStyles {
		if strings.EqualFold(strings.TrimSpace(v), s) {
			return true
		}
	}
	return false
}

// TitleStyle is the title style of slide i: its own directive, else the front
// matter, else "rule".
func (d *Deck) TitleStyle(i int) string {
	for _, v := range []string{d.Slides[i].TitleStyle, d.Meta.TitleStyle} {
		if ValidTitleStyle(v) {
			return strings.ToLower(strings.TrimSpace(v))
		}
	}
	return "rule"
}

// Theme is the front-matter theme setting. It accepts `theme: name` or a
// mapping: `{name: nord, background: paint, accent: "#88c0d0"}` or
// `{path: brand.yaml}`. Keys other than name, path and background are palette
// overrides; the render package validates them.
type Theme struct {
	Name       string
	Path       string            // custom theme file, relative to the deck
	Background string            // "terminal" (default) or "paint"
	Overrides  map[string]string // palette key -> value
	Nested     []string          // keys with non-scalar values (e.g. presenterm's override), ignored
}

func (t *Theme) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind == yaml.ScalarNode {
		t.Name = n.Value
		return nil
	}
	if n.Kind != yaml.MappingNode {
		return fmt.Errorf("line %d: theme must be a name or a mapping", n.Line)
	}
	for i := 0; i+1 < len(n.Content); i += 2 {
		k, v := n.Content[i].Value, n.Content[i+1]
		if v.Kind != yaml.ScalarNode {
			t.Nested = append(t.Nested, k)
			continue
		}
		switch k {
		case "name":
			t.Name = v.Value
		case "path":
			t.Path = v.Value
		case "background":
			t.Background = v.Value
		default:
			if t.Overrides == nil {
				t.Overrides = map[string]string{}
			}
			t.Overrides[k] = v.Value
		}
	}
	return nil
}

type Deck struct {
	Meta   Meta
	Slides []Slide
	Dir    string // directory of the deck file; image paths resolve against it
	// Warnings are recoverable problems found while parsing, such as an
	// unknown transition or an unclosed speaker note, each prefixed with
	// where it is ("line 12: ...", "front matter: ...").
	Warnings []string
}

// FadeEnabled reports whether content revealed by a pause on slide i fades in:
// on unless the slide's fade directive, or failing that the front matter, turns it off.
func (d *Deck) FadeEnabled(i int) bool {
	v := d.Slides[i].Fade
	if v == "" {
		v = d.Meta.Fade
	}
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "false", "off", "no", "none":
		return false
	}
	return true
}

// ProgressEnabled reports whether the presenter reports deck progress to the
// terminal (OSC 9;4): on unless the front matter's progress is false, off, no or none.
func (d *Deck) ProgressEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(d.Meta.Progress)) {
	case "false", "off", "no", "none":
		return false
	}
	return true
}

type Slide struct {
	Title      string
	Blocks     []Block
	Steps      int // number of reveal steps; 1 means no pauses
	Centered   bool
	Transition string   // overrides Meta.Transition when set
	Fade       string   // overrides Meta.Fade when set
	TitleStyle string   // overrides Meta.TitleStyle when set
	TitleSize  string   // overrides Meta.TitleSize when set
	Subtitle   string   // line under the title (inline markdown)
	Cover      bool     // synthesized title slide from front matter
	Notes      []string // speaker notes (markdown), in source order; never rendered on the slide
}

type BlockKind int

const (
	Markdown BlockKind = iota
	Columns
	Exec
	Image
	Code // fenced code with line highlights; see Block.Highlights
)

// Block is a unit of content revealed at Step (0-based).
type Block struct {
	Kind     BlockKind
	Step     int
	Text     string    // Markdown: source; Exec: script body; Image: alt text
	Lang     string    // Exec: fence language
	Widths   []int     // Columns: relative widths
	Children [][]Block // Columns: blocks per column
	ExecID   int       // Exec: deck-unique id
	Path     string    // Image: path as written, relative to Deck.Dir
	// Highlights (Code, Exec): one entry per reveal group, each a per-line
	// selection of Text. Group k shows at step Step+k with the other lines
	// dimmed; the block takes len(Highlights) steps, and a single group is a
	// static highlight. Nil means no highlighting.
	Highlights [][]bool
}
