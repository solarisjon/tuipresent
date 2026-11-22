// Package deck parses presenterm-flavoured markdown into slides.
package deck

import (
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
	Fade       string `yaml:"fade"` // off/false disables the pause reveal fade deck-wide
}

// Theme accepts `theme: name` or `theme: {name: name}`.
type Theme struct {
	Name string `yaml:"name"`
}

func (t *Theme) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind == yaml.ScalarNode {
		t.Name = n.Value
		return nil
	}
	type plain Theme
	return n.Decode((*plain)(t))
}

type Deck struct {
	Meta   Meta
	Slides []Slide
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

type Slide struct {
	Title      string
	Blocks     []Block
	Steps      int // number of reveal steps; 1 means no pauses
	Centered   bool
	Transition string   // overrides Meta.Transition when set
	Fade       string   // overrides Meta.Fade when set
	Cover      bool     // synthesized title slide from front matter
	Notes      []string // speaker notes (markdown), in source order; never rendered on the slide
}

type BlockKind int

const (
	Markdown BlockKind = iota
	Columns
	Exec
)

// Block is a unit of content revealed at Step (0-based).
type Block struct {
	Kind     BlockKind
	Step     int
	Text     string    // Markdown: source; Exec: script body
	Lang     string    // Exec: fence language
	Widths   []int     // Columns: relative widths
	Children [][]Block // Columns: blocks per column
	ExecID   int       // Exec: deck-unique id
}
