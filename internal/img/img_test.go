package img

import (
	"encoding/base64"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/kitty"
)

func solid(w, h int, c color.Color) *image.NRGBA {
	m := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			m.Set(x, y, c)
		}
	}
	return m
}

func TestLoadPNGAndJPEG(t *testing.T) {
	dir := t.TempDir()
	src := solid(40, 20, color.NRGBA{200, 100, 50, 255})
	write := func(name string, enc func(f *os.File) error) string {
		p := filepath.Join(dir, name)
		f, err := os.Create(p)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if err := enc(f); err != nil {
			t.Fatal(err)
		}
		return p
	}
	for _, p := range []string{
		write("a.png", func(f *os.File) error { return png.Encode(f, src) }),
		write("a.jpg", func(f *os.File) error { return jpeg.Encode(f, src, nil) }),
	} {
		m, err := Load(p)
		if err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		if b := m.Bounds(); b.Dx() != 40 || b.Dy() != 20 {
			t.Errorf("%s: bounds %v", p, b)
		}
	}
	if _, err := Load(filepath.Join(dir, "missing.png")); err == nil {
		t.Error("missing file loaded")
	}
	notImage := write("x.png", func(f *os.File) error { _, err := f.WriteString("not a png"); return err })
	if _, err := Load(notImage); err == nil {
		t.Error("garbage decoded")
	}
}

func TestFit(t *testing.T) {
	tests := []struct {
		w, h, maxC, maxR int
		cols, rows       int
	}{
		{320, 160, 40, 100, 40, 10}, // width-bound: 2:1 image is 4:1 in cells
		{320, 160, 100, 10, 40, 10}, // height-bound
		{100, 100, 50, 50, 50, 25},  // square image is twice as wide in cells
		{100, 400, 80, 20, 10, 20},  // tall image
		{1000, 10, 50, 50, 50, 1},   // never zero rows
		{0, 10, 50, 50, 0, 0},
		{10, 10, 0, 5, 0, 0},
		{100, 100, 1000, 1000, MaxCells, 149},
	}
	for _, tt := range tests {
		c, r := Fit(tt.w, tt.h, tt.maxC, tt.maxR)
		if c != tt.cols || r != tt.rows {
			t.Errorf("Fit(%d,%d,%d,%d) = %d,%d want %d,%d", tt.w, tt.h, tt.maxC, tt.maxR, c, r, tt.cols, tt.rows)
		}
	}
}

func TestScaleAverages(t *testing.T) {
	m := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	m.SetNRGBA(0, 0, color.NRGBA{255, 0, 0, 255})
	m.SetNRGBA(1, 0, color.NRGBA{0, 0, 255, 0}) // transparent: must not tint
	got := Scale(m, 1, 1).NRGBAAt(0, 0)
	if got.R < 250 || got.B != 0 || got.A < 120 || got.A > 135 {
		t.Errorf("got %v", got)
	}
}

func TestHalfBlocksGeometryAndColors(t *testing.T) {
	m := image.NewNRGBA(image.Rect(0, 0, 3, 2))
	for x := range 3 {
		m.SetNRGBA(x, 0, color.NRGBA{255, 0, 0, 255})
		m.SetNRGBA(x, 1, color.NRGBA{0, 0, 255, 255})
	}
	lines := HalfBlocks(m, 3, 1)
	if len(lines) != 1 || ansi.StringWidth(lines[0]) != 3 {
		t.Fatalf("got %q", lines)
	}
	if !strings.Contains(lines[0], "38;2;255;0;0;48;2;0;0;255m▀") {
		t.Errorf("want red over blue half-block, got %q", lines[0])
	}
	for _, sz := range [][2]int{{1, 1}, {17, 5}, {80, 23}} {
		lines := HalfBlocks(solid(64, 48, color.White), sz[0], sz[1])
		if len(lines) != sz[1] {
			t.Errorf("%v: %d lines", sz, len(lines))
		}
		for i, l := range lines {
			if w := ansi.StringWidth(l); w != sz[0] {
				t.Errorf("%v line %d: width %d", sz, i, w)
			}
		}
	}
	clear := HalfBlocks(image.NewNRGBA(image.Rect(0, 0, 2, 2)), 2, 1)[0]
	if ansi.Strip(clear) != "  " {
		t.Errorf("transparent image should be blank, got %q", clear)
	}
}

func TestPlaceholderGeometryAndCut(t *testing.T) {
	id := uint32(0x123456)
	lines := Placeholder(id, 7, 3)
	if len(lines) != 3 {
		t.Fatalf("%d lines", len(lines))
	}
	for i, l := range lines {
		// Bubble Tea measures with grapheme or wcwidth semantics; both must agree.
		if w, wc := ansi.StringWidth(l), ansi.StringWidthWc(l); w != 7 || wc != 7 {
			t.Errorf("line %d width %d (wcwidth %d)", i, w, wc)
		}
		if !strings.HasPrefix(l, "\x1b[38;2;18;52;86m") {
			t.Errorf("line %d: id not in foreground color: %q", i, l)
		}
	}
	// Transitions slice frames with ansi.Cut; a sliced cell must keep both
	// diacritics so the terminal still knows which tile it is.
	cut := ansi.Cut(lines[2], 4, 6)
	if ansi.StringWidth(cut) != 2 {
		t.Fatalf("cut width %d", ansi.StringWidth(cut))
	}
	want := string([]rune{kitty.Placeholder, kitty.Diacritic(2), kitty.Diacritic(4), kitty.Placeholder, kitty.Diacritic(2), kitty.Diacritic(5)})
	if ansi.Strip(cut) != want {
		t.Errorf("cut = %q, want %q", ansi.Strip(cut), want)
	}
	if !strings.Contains(cut, "38;2;18;52;86") {
		t.Errorf("cut lost the id color: %q", cut)
	}
}

var chunkRe = regexp.MustCompile(`\x1b_G([^;]*);([^\x1b]*)\x1b\\`)

func TestTransmitChunking(t *testing.T) {
	data := make([]byte, 10000) // 13336 base64 bytes: 4 chunks
	for i := range data {
		data[i] = byte(i * 7)
	}
	seq := Transmit(99, data, 20, 5)
	chunks := chunkRe.FindAllStringSubmatch(seq, -1)
	if len(chunks) != 4 || strings.Join(chunkRe.FindAllString(seq, -1), "") != seq {
		t.Fatalf("got %d chunks", len(chunks))
	}
	var payload strings.Builder
	for i, c := range chunks {
		ctrl, body := c[1], c[2]
		last := i == len(chunks)-1
		if len(body) > 4096 || (!last && len(body)%4 != 0) {
			t.Errorf("chunk %d: bad payload length %d", i, len(body))
		}
		switch {
		case i == 0:
			for _, k := range []string{"a=T", "U=1", "f=100", "i=99", "c=20", "r=5", "q=2", "m=1"} {
				if !strings.Contains(ctrl, k) {
					t.Errorf("first chunk %q missing %s", ctrl, k)
				}
			}
		case last:
			if ctrl != "m=0,q=2" {
				t.Errorf("last chunk control %q", ctrl)
			}
		default:
			if ctrl != "m=1,q=2" {
				t.Errorf("chunk %d control %q", i, ctrl)
			}
		}
		payload.WriteString(body)
	}
	got, err := base64.StdEncoding.DecodeString(payload.String())
	if err != nil || string(got) != string(data) {
		t.Errorf("payload does not round-trip: %v", err)
	}
	if one := Transmit(1, []byte("hi"), 1, 1); strings.Count(one, "\x1b_G") != 1 || !strings.Contains(one, "m=0;") {
		t.Errorf("small image should be one final chunk: %q", one)
	}
}

func TestTransmitterUploadsOncePerSize(t *testing.T) {
	tr := NewTransmitter()
	src := solid(30, 30, color.White)
	a := tr.Place("k", src, 10, 5)
	first := tr.Flush()
	if strings.Count(first, "a=T") != 1 {
		t.Fatalf("want one upload, got %q", first)
	}
	if b := tr.Place("k", src, 10, 5); strings.Join(a, "") != strings.Join(b, "") || tr.Flush() != "" {
		t.Error("same key and size should reuse the upload")
	}
	tr.Place("k", src, 12, 6)
	if strings.Count(tr.Flush(), "a=T") != 1 {
		t.Error("new size should upload again")
	}
	if c := tr.Cleanup(); strings.Count(c, "a=d") != 2 {
		t.Errorf("cleanup = %q", c)
	}
}

func TestParseMode(t *testing.T) {
	for in, want := range map[string]Mode{"": Auto, "AUTO": Auto, " kitty ": Kitty, "Sixel": Sixel, "iterm2": ITerm2, "blocks": Blocks, "off": Off} {
		if got, err := ParseMode(in); err != nil || got != want {
			t.Errorf("ParseMode(%q) = %q, %v", in, got, err)
		}
	}
}

func TestDetect(t *testing.T) {
	env := func(kv ...string) func(string) string {
		m := map[string]string{}
		for i := 0; i < len(kv); i += 2 {
			m[kv[i]] = kv[i+1]
		}
		return func(k string) string { return m[k] }
	}
	tests := []struct {
		env  func(string) string
		want Mode
	}{
		{env("TERM_PROGRAM", "ghostty"), Kitty},
		{env("TERM", "xterm-kitty"), Kitty},
		{env("KITTY_WINDOW_ID", "1"), Kitty},
		{env("TERM_PROGRAM", "ghostty", "TMUX", "/tmp/x"), Blocks},
		{env("TERM", "screen-256color", "KITTY_WINDOW_ID", "1"), Blocks},
		{env("TERM", "xterm-256color"), Blocks},
		{env("TERM_PROGRAM", "WezTerm", "TERM", "xterm-256color"), ITerm2},
		{env("TERM_PROGRAM", "iTerm.app"), ITerm2},
		{env("LC_TERMINAL", "iTerm2"), ITerm2}, // survives ssh
		{env("ITERM_SESSION_ID", "w0t0p0:ABC"), ITerm2},
		{env("TERM", "foot"), Sixel},
		{env("TERM", "foot-extra"), Sixel},
		{env("TERM", "footnote"), Blocks},
		{env("TERM_PROGRAM", "WezTerm", "TMUX", "/tmp/x"), Blocks},
		{env("TERM_PROGRAM", "iTerm.app", "TERM", "tmux-256color"), Blocks},
		{env("LC_TERMINAL", "iTerm2", "TERM", "screen"), Blocks},
		{env("TERM", "foot", "TMUX", "/tmp/x"), Blocks},
		{env("TERM_PROGRAM", "Apple_Terminal", "TERM", "xterm-256color"), Blocks},
		{env("TERM_PROGRAM", "vscode"), Blocks},
		{env("TERM", "xterm-kitty", "TERM_PROGRAM", "WezTerm"), Kitty}, // kitty wins: it has placeholders
		{env(), Blocks},
	}
	for i, tt := range tests {
		if got := Detect(tt.env); got != tt.want {
			t.Errorf("case %d: got %s want %s", i, got, tt.want)
		}
	}
	if _, err := ParseMode("sixels"); err == nil {
		t.Error("unknown mode accepted")
	}
}
