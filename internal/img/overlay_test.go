package img

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestEncodeSixelGolden(t *testing.T) {
	red := solid(2, 2, color.NRGBA{255, 0, 0, 255})
	want := "\x1bP0;1;0q\"1;1;2;2#0;2;100;0;0#0BB\x1b\\"
	if got := EncodeSixel(red); got != want {
		t.Errorf("solid red:\n got %q\nwant %q", got, want)
	}

	// Two colours side by side, 8 px wide so the left half is a run.
	m := image.NewNRGBA(image.Rect(0, 0, 8, 6))
	for y := range 6 {
		for x := range 8 {
			if x < 4 {
				m.SetNRGBA(x, y, color.NRGBA{0, 0, 255, 255})
			} else {
				m.SetNRGBA(x, y, color.NRGBA{0, 255, 0, 255})
			}
		}
	}
	got := EncodeSixel(m)
	// Palette order follows the median cut; each colour draws only its half.
	if !strings.HasPrefix(got, "\x1bP0;1;0q\"1;1;8;6#0;2;") || !strings.HasSuffix(got, "\x1b\\") {
		t.Fatalf("bad framing %q", got)
	}
	if !strings.Contains(got, "!4~") || !strings.Contains(got, "$") {
		t.Errorf("want run-length encoded halves joined by $: %q", got)
	}
}

func TestEncodeSixelTransparentAndEmpty(t *testing.T) {
	if EncodeSixel(image.NewNRGBA(image.Rect(0, 0, 0, 0))) != "" {
		t.Error("empty image should encode to nothing")
	}
	m := image.NewNRGBA(image.Rect(0, 0, 3, 1))
	m.SetNRGBA(1, 0, color.NRGBA{255, 255, 255, 255})
	// Only the middle pixel is drawn; leading transparent column is a "?" (no bits).
	want := "\x1bP0;1;0q\"1;1;3;1#0;2;100;100;100#0?@\x1b\\"
	if got := EncodeSixel(m); got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

// decodeSixel is a minimal sixel decoder for the encoder's output: it
// returns the RGB of each pixel, nil where nothing was drawn.
func decodeSixel(t *testing.T, s string) (w, h int, px map[[2]int]color.RGBA) {
	t.Helper()
	s = strings.TrimSuffix(strings.TrimPrefix(s, "\x1bP0;1;0q"), "\x1b\\")
	px = map[[2]int]color.RGBA{}
	pal := map[int]color.RGBA{}
	x, band, cur := 0, 0, 0
	for i := 0; i < len(s); {
		c := s[i]
		num := func() int {
			j := i + 1
			for j < len(s) && s[j] >= '0' && s[j] <= '9' {
				j++
			}
			n, _ := strconv.Atoi(s[i+1 : j])
			i = j
			return n
		}
		switch {
		case c == '"':
			i++
			parts := strings.SplitN(s[i:], "#", 2)
			f := strings.Split(parts[0], ";")
			w, _ = strconv.Atoi(f[2])
			h, _ = strconv.Atoi(f[3])
			i += len(parts[0])
		case c == '#':
			cur = num()
			if i < len(s) && s[i] == ';' {
				var v [4]int
				for k := range v {
					i++ // ';'
					j := i
					for j < len(s) && s[j] >= '0' && s[j] <= '9' {
						j++
					}
					v[k], _ = strconv.Atoi(s[i:j])
					i = j
				}
				pal[cur] = color.RGBA{uint8(v[1] * 255 / 100), uint8(v[2] * 255 / 100), uint8(v[3] * 255 / 100), 255}
			}
		case c == '$':
			x = 0
			i++
		case c == '-':
			x, band = 0, band+1
			i++
		case c == '!':
			n := num()
			ch := s[i]
			i++
			for range n {
				for k := range 6 {
					if (ch-'?')>>k&1 == 1 {
						px[[2]int{x, band*6 + k}] = pal[cur]
					}
				}
				x++
			}
		default:
			for k := range 6 {
				if (c-'?')>>k&1 == 1 {
					px[[2]int{x, band*6 + k}] = pal[cur]
				}
			}
			x++
			i++
		}
	}
	return
}

func TestEncodeSixelRoundTrip(t *testing.T) {
	// A 13x14 image of four flat colours (spanning three bands, with a
	// transparent corner) must decode to the same pixels.
	cols := []color.NRGBA{{255, 0, 0, 255}, {0, 255, 0, 255}, {0, 0, 255, 255}, {255, 255, 0, 255}}
	m := image.NewNRGBA(image.Rect(0, 0, 13, 14))
	for y := range 14 {
		for x := range 13 {
			m.SetNRGBA(x, y, cols[(x/7)+2*(y/7)])
		}
	}
	m.SetNRGBA(12, 13, color.NRGBA{})
	w, h, px := decodeSixel(t, EncodeSixel(m))
	if w != 13 || h != 14 {
		t.Fatalf("size %dx%d", w, h)
	}
	for y := range 14 {
		for x := range 13 {
			want := m.NRGBAAt(x, y)
			got, ok := px[[2]int{x, y}]
			if want.A == 0 {
				if ok {
					t.Errorf("(%d,%d) should be undrawn", x, y)
				}
				continue
			}
			if !ok || got.R>>4 != want.R>>4 || got.G>>4 != want.G>>4 || got.B>>4 != want.B>>4 {
				t.Errorf("(%d,%d) = %v, want %v", x, y, got, want)
			}
		}
	}
}

func TestEncodeSixelPaletteCap(t *testing.T) {
	m := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	for y := range 64 {
		for x := range 64 {
			m.SetNRGBA(x, y, color.NRGBA{uint8(x * 4), uint8(y * 4), uint8(x * y), 255})
		}
	}
	if n := len(regexp.MustCompile(`#\d+;2;`).FindAllString(EncodeSixel(m), -1)); n > MaxSixelColors || n < 2 {
		t.Errorf("palette has %d colours", n)
	}
}

func TestITerm2Seq(t *testing.T) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, solid(2, 2, color.NRGBA{1, 2, 3, 255})); err != nil {
		t.Fatal(err)
	}
	s := ITerm2Seq(buf.Bytes(), 10, 4)
	head := "\x1b]1337;File=inline=1;size=" + strconv.Itoa(buf.Len()) + ";width=10;height=4;preserveAspectRatio=0:"
	if !strings.HasPrefix(s, head) || !strings.HasSuffix(s, "\a") {
		t.Fatalf("bad sequence %q", s[:min(len(s), 100)])
	}
	data, err := base64.StdEncoding.DecodeString(strings.TrimSuffix(strings.TrimPrefix(s, head), "\a"))
	if err != nil || !bytes.Equal(data, buf.Bytes()) {
		t.Error("payload is not the PNG")
	}
}

func TestOverlayExtractAndDraw(t *testing.T) {
	for _, mode := range []Mode{Sixel, ITerm2} {
		o := NewOverlay(mode)
		o.SetCellSize(10, 20)
		src := solid(40, 40, color.NRGBA{9, 99, 199, 255})
		lines := o.Place("a", src, 4, 2)
		if again := o.Place("a", src, 4, 2); again[0] != lines[0] {
			t.Error("same key and size should reuse the id")
		}
		// Two cells of text, then the image, on rows 3-4 of the frame.
		frame := strings.Join([]string{"top", "mid", "  hi" + lines[0] + "!", "  " + lines[1]}, "\n")
		clean, ps := o.Extract(frame)
		if len(ps) != 1 {
			t.Fatalf("%s: placements %v", mode, ps)
		}
		if p := ps[0]; p.Row != 2 || p.Col != 4 || p.Cols != 4 || p.Rows != 2 {
			t.Errorf("%s: placement %+v", mode, p)
		}
		cl := strings.Split(clean, "\n")
		for i, l := range cl {
			if strings.ContainsRune(l, '\U0010EEEE') {
				t.Errorf("%s: line %d still has stand-ins", mode, i)
			}
			if w := ansi.StringWidth(l); w != ansi.StringWidth(strings.Split(frame, "\n")[i]) {
				t.Errorf("%s: line %d width %d changed", mode, i, w)
			}
		}
		if got := ansi.Strip(cl[2]); got != "  hi    !" {
			t.Errorf("%s: line 2 = %q", mode, got)
		}
		seq := o.Draw(ps)
		if !strings.HasPrefix(seq, "\x1b7\x1b[3;5H") || !strings.HasSuffix(seq, "\x1b8") {
			t.Errorf("%s: draw framing %q", mode, seq[:min(len(seq), 30)])
		}
		if mode == Sixel && !strings.Contains(seq, "\x1bPq") && !strings.Contains(seq, "\x1bP0;1;0q") {
			t.Errorf("sixel draw has no DCS")
		}
		if mode == ITerm2 && !strings.Contains(seq, "\x1b]1337;File=") {
			t.Errorf("iterm2 draw has no OSC 1337")
		}
		// Encoding is cached per cell size.
		if o.Draw(ps) != seq {
			t.Error("draw not repeatable")
		}
	}
	o := NewOverlay(Sixel)
	if f, ps := o.Extract("plain\ntext"); f != "plain\ntext" || ps != nil {
		t.Error("frame without images changed")
	}
}

func TestOverlaySixelBoundedAndCentred(t *testing.T) {
	o := NewOverlay(Sixel)
	o.SetCellSize(20, 40) // hi-dpi: a 100x30 cell box would be 2000x1200 px
	lines := o.Place("big", solid(64, 32, color.NRGBA{255, 0, 0, 255}), 100, 30)
	_, ps := o.Extract(strings.Join(lines, "\n"))
	e, ok := o.encode(ps[0].ID)
	if !ok {
		t.Fatal("no encoding")
	}
	var w, h int
	for _, f := range strings.SplitN(strings.SplitN(e.seq, "\"1;1;", 2)[1], "#", 2)[:1] {
		parts := strings.Split(f, ";")
		w, _ = strconv.Atoi(parts[0])
		h, _ = strconv.Atoi(parts[1])
	}
	if w*h > maxSixelPixels || w > 2000 || h > 1200 {
		t.Errorf("sixel %dx%d exceeds bounds", w, h)
	}
	if e.dx < 0 || e.dy < 0 || e.dx*20+w > 2000 || e.dy*40+h > 1200 {
		t.Errorf("offset %d,%d puts %dx%d outside the box", e.dx, e.dy, w, h)
	}
}
