package deck

import (
	"strings"
	"testing"
)

func TestTitleStyleAndSubtitle(t *testing.T) {
	src := "---\ntitle_style: frame\n---\n" +
		"One\n===\n<!-- subtitle: the *why* -->\n<!-- title_style: curly -->\nbody\n\n<!-- end_slide -->\n" +
		"Two\n===\nbody\n"
	d := mustParse(t, src)
	one, two := d.Slides[0], d.Slides[1]
	if one.Subtitle != "the *why*" || one.TitleStyle != "curly" {
		t.Errorf("slide 1 subtitle %q style %q", one.Subtitle, one.TitleStyle)
	}
	if got := d.TitleStyle(0); got != "curly" {
		t.Errorf("slide 1 style = %q, want curly", got)
	}
	if got := d.TitleStyle(1); got != "frame" {
		t.Errorf("slide 2 inherits %q, want frame (%+v)", got, two)
	}
	if len(d.Warnings) != 0 {
		t.Errorf("warnings = %q", d.Warnings)
	}
}

func TestUnknownTitleStyleWarns(t *testing.T) {
	d := mustParse(t, "---\ntitle_style: wavy\n---\nOne\n===\n<!-- title_style: neon -->\nbody\n")
	if len(d.Warnings) != 2 || !strings.HasPrefix(d.Warnings[0], "front matter: unknown title_style") || !strings.HasPrefix(d.Warnings[1], "line 6: unknown title_style") {
		t.Errorf("warnings = %q", d.Warnings)
	}
	if got := d.TitleStyle(0); got != "rule" {
		t.Errorf("style = %q, want rule", got)
	}
}
