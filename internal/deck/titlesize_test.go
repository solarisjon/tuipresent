package deck

import (
	"strings"
	"testing"
)

func TestTitleSizeResolution(t *testing.T) {
	d := mustParse(t, "---\ntitle_size: 2\n---\nOne\n===\n<!-- title_size: 3 -->\nbody\n\n<!-- end_slide -->\nTwo\n===\nbody\n\n<!-- end_slide -->\nThree\n===\n<!-- title_size: 1 -->\nbody\n")
	for i, want := range []int{3, 2, 1} {
		if got := d.TitleSize(i); got != want {
			t.Errorf("slide %d size = %d, want %d", i+1, got, want)
		}
	}
	if len(d.Warnings) != 0 {
		t.Errorf("warnings = %q", d.Warnings)
	}
}

func TestUnknownTitleSizeWarns(t *testing.T) {
	d := mustParse(t, "---\ntitle_size: huge\n---\nOne\n===\n<!-- title_size: 9 -->\nbody\n\n<!-- end_slide -->\nTwo\n===\n<!-- title_size: 0 -->\nbody\n")
	if len(d.Warnings) != 3 || !strings.HasPrefix(d.Warnings[0], "front matter: unknown title_size") ||
		!strings.HasPrefix(d.Warnings[1], "line 6: unknown title_size") || !strings.HasPrefix(d.Warnings[2], "line 12: unknown title_size") {
		t.Errorf("warnings = %q", d.Warnings)
	}
	for i := range d.Slides {
		if got := d.TitleSize(i); got != 1 {
			t.Errorf("slide %d size = %d, want 1", i+1, got)
		}
	}
}

func TestTitleSizeFrontMatterIntegerAndNoCoverEffect(t *testing.T) {
	d := mustParse(t, "---\ntitle: Cover\ntitle_size: 2\n---\nOne\n===\nbody\n")
	if len(d.Warnings) != 0 || d.TitleSize(1) != 2 {
		t.Errorf("warnings %q size %d", d.Warnings, d.TitleSize(1))
	}
}
