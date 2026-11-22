package deck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImages(t *testing.T) {
	d := mustParse(t, "Pics\n===\n\n![a logo](assets/logo.png)\ntext\n  ![](<my pic.jpg> \"title\")\n\nsee ![inline](x.png) here\n")
	bs := d.Slides[0].Blocks
	if d.Slides[0].Title != "Pics" || len(bs) != 4 {
		t.Fatalf("got title %q, %d blocks: %+v", d.Slides[0].Title, len(bs), bs)
	}
	if bs[0].Kind != Image || bs[0].Text != "a logo" || bs[0].Path != "assets/logo.png" {
		t.Errorf("block 0 = %+v", bs[0])
	}
	if bs[2].Kind != Image || bs[2].Text != "" || bs[2].Path != "my pic.jpg" {
		t.Errorf("block 2 = %+v", bs[2])
	}
	if bs[3].Kind != Markdown || !strings.Contains(bs[3].Text, "![inline](x.png)") {
		t.Errorf("inline image should stay markdown: %+v", bs[3])
	}
}

func TestImageBeforeTitleBlocksTitle(t *testing.T) {
	d := mustParse(t, "![x](x.png)\nNot a title\n===\n")
	if d.Slides[0].Title != "" {
		t.Errorf("title after an image should not become the slide title")
	}
}

func TestLoadSetsDir(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "deck.md")
	if err := os.WriteFile(p, []byte("hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	d, err := Load(p)
	if err != nil || d.Dir != dir {
		t.Errorf("Dir = %q, err %v", d.Dir, err)
	}
}
