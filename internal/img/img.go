// Package img draws raster images in the terminal: kitty graphics Unicode
// placeholders, sixel or iTerm2 inline images where supported, truecolor
// half-blocks everywhere else.
//
// Geometry is identical in every mode (a cell is treated as 1 pixel wide and
// 2 pixels tall), so layout measured with half-blocks holds for the rest.
package img

import (
	"fmt"
	"image"
	"image/color"
	_ "image/gif" // register decoders
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strings"
)

type Mode string

const (
	Auto   Mode = "auto"
	Kitty  Mode = "kitty"
	Sixel  Mode = "sixel"
	ITerm2 Mode = "iterm2"
	Blocks Mode = "blocks"
	Off    Mode = "off"
)

func ParseMode(s string) (Mode, error) {
	switch m := Mode(strings.ToLower(strings.TrimSpace(s))); m {
	case Auto, Kitty, Sixel, ITerm2, Blocks, Off:
		return m, nil
	case "":
		return Auto, nil
	}
	return "", fmt.Errorf("unknown image mode %q (want auto, kitty, sixel, iterm2, blocks or off)", s)
}

// Detect picks the best image protocol the terminal is known to support:
// kitty graphics (Ghostty, kitty), the iTerm2 inline-image protocol (iTerm2,
// WezTerm), sixel (foot), or half-blocks. It is conservative: anything
// unrecognised, and anything inside tmux or screen (which swallow the
// escape sequences), gets half-blocks.
func Detect(getenv func(string) string) Mode {
	term := getenv("TERM")
	if getenv("TMUX") != "" || strings.HasPrefix(term, "screen") || strings.HasPrefix(term, "tmux") {
		return Blocks
	}
	tp := strings.ToLower(getenv("TERM_PROGRAM"))
	switch {
	case tp == "ghostty" || tp == "kitty" || strings.Contains(term, "kitty") || strings.Contains(term, "ghostty") ||
		getenv("KITTY_WINDOW_ID") != "" || getenv("GHOSTTY_RESOURCES_DIR") != "":
		return Kitty
	case tp == "wezterm" || tp == "iterm.app" || strings.EqualFold(getenv("LC_TERMINAL"), "iTerm2") ||
		getenv("ITERM_SESSION_ID") != "":
		return ITerm2
	case term == "foot" || strings.HasPrefix(term, "foot-"):
		return Sixel
	}
	return Blocks
}

// Load decodes a PNG, JPEG or GIF file.
func Load(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	m, _, err := image.Decode(f)
	return m, err
}

// MaxCells is the largest image side in cells (the kitty diacritic table size).
const MaxCells = 297

// Fit returns the largest cell box within maxCols x maxRows that preserves the
// image's aspect ratio, where a cell is twice as tall as it is wide.
func Fit(imgW, imgH, maxCols, maxRows int) (cols, rows int) {
	maxCols, maxRows = min(maxCols, MaxCells), min(maxRows, MaxCells)
	if imgW <= 0 || imgH <= 0 || maxCols <= 0 || maxRows <= 0 {
		return 0, 0
	}
	cols = maxCols
	rows = (cols*imgH + imgW) / (2 * imgW) // round(cols*h/w/2)
	if rows > maxRows {
		rows = maxRows
		cols = (2*rows*imgW + imgH/2) / imgH
	}
	return max(min(cols, maxCols), 1), max(rows, 1)
}

// Scale resamples src to w x h with an area-average (box) filter.
func Scale(src image.Image, w, h int) *image.NRGBA {
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	b := src.Bounds()
	sw, sh := b.Dx(), b.Dy()
	if sw == 0 || sh == 0 {
		return dst
	}
	for y := range h {
		y0, y1 := b.Min.Y+y*sh/h, b.Min.Y+max((y+1)*sh/h, y*sh/h+1)
		for x := range w {
			x0, x1 := b.Min.X+x*sw/w, b.Min.X+max((x+1)*sw/w, x*sw/w+1)
			var r, g, bl, a, n uint64
			for sy := y0; sy < y1; sy++ {
				for sx := x0; sx < x1; sx++ {
					c := color.NRGBA64Model.Convert(src.At(sx, sy)).(color.NRGBA64)
					// Weight colour by alpha so transparent pixels don't darken edges.
					r += uint64(c.R) * uint64(c.A)
					g += uint64(c.G) * uint64(c.A)
					bl += uint64(c.B) * uint64(c.A)
					a += uint64(c.A)
					n++
				}
			}
			out := color.NRGBA{A: uint8(a / n >> 8)}
			if a > 0 {
				out.R, out.G, out.B = uint8(r/a>>8), uint8(g/a>>8), uint8(bl/a>>8)
			}
			dst.SetNRGBA(x, y, out)
		}
	}
	return dst
}

// HalfBlocks renders src into rows lines of exactly cols cells using upper
// half blocks: foreground is the top pixel, background the bottom one.
// Mostly transparent pixels show the terminal background.
func HalfBlocks(src image.Image, cols, rows int) []string {
	px := Scale(src, cols, rows*2)
	lines := make([]string, rows)
	for y := range rows {
		var sb strings.Builder
		for x := range cols {
			t, b := px.NRGBAAt(x, 2*y), px.NRGBAAt(x, 2*y+1)
			switch to, bo := t.A >= 128, b.A >= 128; {
			case to && bo:
				fmt.Fprintf(&sb, "\x1b[38;2;%d;%d;%d;48;2;%d;%d;%dm▀", t.R, t.G, t.B, b.R, b.G, b.B)
			case to:
				fmt.Fprintf(&sb, "\x1b[49;38;2;%d;%d;%dm▀", t.R, t.G, t.B)
			case bo:
				fmt.Fprintf(&sb, "\x1b[49;38;2;%d;%d;%dm▄", b.R, b.G, b.B)
			default:
				sb.WriteString("\x1b[0m ")
			}
		}
		sb.WriteString("\x1b[0m")
		lines[y] = sb.String()
	}
	return lines
}
