package img

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"math"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/kitty"
)

// ITerm2Seq returns the iTerm2 inline-image sequence (OSC 1337 File=) that
// draws a PNG at the cursor, scaled by the terminal to exactly cols x rows
// cells. WezTerm understands the same sequence.
func ITerm2Seq(pngData []byte, cols, rows int) string {
	return fmt.Sprintf("\x1b]1337;File=inline=1;size=%d;width=%d;height=%d;preserveAspectRatio=0:%s\a",
		len(pngData), cols, rows, base64.StdEncoding.EncodeToString(pngData))
}

// Sixel and iTerm2 images have no placeholder cells: the terminal paints
// pixels over whatever cells lie under the cursor. So layout, transitions and
// fades work on stand-in cells (the kitty placeholder runes, which the
// renderer already keeps intact), and the presenter calls Extract on the frame
// it is about to show: that turns the stand-ins into blanks and reports where
// each image's top-left cell is, and Draw then paints the pixels there once
// the frame is on screen.

// Placement is one image's position in a frame.
type Placement struct {
	ID         uint32
	Row, Col   int // 0-based cell of the image's top-left corner
	Cols, Rows int
}

// Overlay hands out placeholder cells for sixel or iTerm2 images and turns
// them into escape sequences on demand. Encoding is lazy and cached, so
// slides that are never shown cost nothing.
type Overlay struct {
	Mode Mode // Sixel or ITerm2

	mu           sync.Mutex
	next         uint32
	ids          map[string]uint32
	ents         map[uint32]*overlayEnt
	cellW, cellH int
}

type overlayEnt struct {
	src        image.Image
	cols, rows int
	enc        map[[2]int]encoded // by cell pixel size
}

type encoded struct {
	seq    string
	dx, dy int // offset in cells inside the box, when the image is smaller
}

// Fallback cell size in pixels when the terminal does not report one. Small
// on purpose: a sixel too small is merely smaller, one too large overruns
// the text below it.
const fallbackCellW, fallbackCellH = 8, 16

// maxSixelPixels bounds one sixel image's area so a full-screen image on a
// big display still encodes and transmits quickly.
const maxSixelPixels = 1 << 20

func NewOverlay(mode Mode) *Overlay {
	return &Overlay{Mode: mode, next: 1, ids: map[string]uint32{}, ents: map[uint32]*overlayEnt{}}
}

// SetCellSize records the terminal's cell size in pixels (0 if unknown).
func (o *Overlay) SetCellSize(w, h int) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.cellW, o.cellH = w, h
}

// Place returns stand-in lines for src shown at cols x rows, like
// Transmitter.Place. key should change when the image does.
func (o *Overlay) Place(key string, src image.Image, cols, rows int) []string {
	o.mu.Lock()
	defer o.mu.Unlock()
	k := fmt.Sprintf("%s\x00%dx%d", key, cols, rows)
	id, ok := o.ids[k]
	if !ok {
		id = o.next
		o.next++
		o.ids[k] = id
		o.ents[id] = &overlayEnt{src: src, cols: cols, rows: rows, enc: map[[2]int]encoded{}}
	}
	return Placeholder(id, cols, rows)
}

var diacriticIndex = sync.OnceValue(func() map[rune]int {
	m := make(map[rune]int, MaxCells)
	for i := range MaxCells {
		m[kitty.Diacritic(i)] = i
	}
	return m
})

// Extract replaces the stand-in cells of frame with blanks and returns the
// frame with the placements found. A frame with no images comes back as is.
func (o *Overlay) Extract(frame string) (string, []Placement) {
	if !strings.ContainsRune(frame, kitty.Placeholder) {
		return frame, nil
	}
	dia := diacriticIndex()
	var ps []Placement
	lines := strings.Split(frame, "\n")
	o.mu.Lock()
	defer o.mu.Unlock()
	for y, line := range lines {
		if !strings.ContainsRune(line, kitty.Placeholder) {
			continue
		}
		var sb strings.Builder
		for pos := 0; pos < len(line); {
			r, n := utf8.DecodeRuneInString(line[pos:])
			if r != kitty.Placeholder {
				sb.WriteString(line[pos : pos+n])
				pos += n
				continue
			}
			// A cell is the placeholder rune plus up to two diacritics (row, column).
			at, end := pos, pos+n
			var idx [2]int
			k := 0
			for ; k < 2 && end < len(line); k++ {
				d, dn := utf8.DecodeRuneInString(line[end:])
				v, ok := dia[d]
				if !ok {
					break
				}
				idx[k], end = v, end+dn
			}
			if k == 2 && idx == [2]int{0, 0} {
				if id, ok := idBefore(line[:at]); ok && o.ents[id] != nil {
					e := o.ents[id]
					ps = append(ps, Placement{ID: id, Row: y, Col: ansi.StringWidth(line[:at]), Cols: e.cols, Rows: e.rows})
				}
			}
			sb.WriteByte(' ')
			pos = end
		}
		lines[y] = sb.String()
	}
	return strings.Join(lines, "\n"), ps
}

// idBefore is the image id carried by the last 24-bit foreground colour in s.
func idBefore(s string) (uint32, bool) {
	i := strings.LastIndex(s, "\x1b[38;2;")
	if i < 0 {
		return 0, false
	}
	var r, g, b uint32
	if n, _ := fmt.Sscanf(s[i+7:], "%d;%d;%dm", &r, &g, &b); n != 3 {
		return 0, false
	}
	return r<<16 | g<<8 | b, true
}

// Draw returns the escape sequences that paint the placements, each at its
// cell, leaving the cursor where it was.
func (o *Overlay) Draw(ps []Placement) string {
	var sb strings.Builder
	for _, p := range ps {
		e, ok := o.encode(p.ID)
		if !ok {
			continue
		}
		fmt.Fprintf(&sb, "\x1b7\x1b[%d;%dH%s\x1b8", p.Row+1+e.dy, p.Col+1+e.dx, e.seq)
	}
	return sb.String()
}

func (o *Overlay) encode(id uint32) (encoded, bool) {
	o.mu.Lock()
	ent := o.ents[id]
	cw, ch := o.cellW, o.cellH
	if cw <= 0 || ch <= 0 {
		cw, ch = fallbackCellW, fallbackCellH
	}
	var e encoded
	var ok bool
	if ent != nil {
		e, ok = ent.enc[[2]int{cw, ch}]
	}
	o.mu.Unlock()
	if ent == nil {
		return encoded{}, false
	}
	if ok {
		return e, true
	}
	if o.Mode == ITerm2 {
		// The terminal scales to the cell box; upload at about cell resolution.
		src, w, h := ent.src, ent.src.Bounds().Dx(), ent.src.Bounds().Dy()
		if tw := ent.cols * 16; w > tw {
			h, w = max(h*tw/w, 1), tw
			src = Scale(src, w, h)
		}
		var buf bytes.Buffer
		if png.Encode(&buf, src) != nil {
			return encoded{}, false
		}
		e = encoded{seq: ITerm2Seq(buf.Bytes(), ent.cols, ent.rows)}
	} else {
		bw, bh := ent.cols*cw, ent.rows*ch // the box in pixels
		w, h := bw, bh
		if w*h > maxSixelPixels {
			f := math.Sqrt(float64(maxSixelPixels) / float64(w*h))
			w, h = max(int(float64(w)*f), 1), max(int(float64(h)*f), 1)
		}
		e = encoded{seq: EncodeSixel(Scale(ent.src, w, h)), dx: (bw - w) / 2 / cw, dy: (bh - h) / 2 / ch}
	}
	o.mu.Lock()
	ent.enc[[2]int{cw, ch}] = e
	o.mu.Unlock()
	return e, true
}
