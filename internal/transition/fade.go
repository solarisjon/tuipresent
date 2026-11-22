package transition

import (
	"image/color"
	"strconv"
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/lucasb-eyer/go-colorful"
)

// FadeIn reveals the cells of to that differ from from by blending their
// foreground (and background, when set) colors up from bg at progress t in
// [0,1]. Cells shared by both frames, and lines that are byte-identical, are
// left untouched. fg stands in for cells drawn in the terminal's default
// foreground. Both frames must have the same geometry; the result keeps it.
func FadeIn(from, to string, t float64, bg, fg color.Color) string {
	if t >= 1 {
		return to
	}
	t = max(t, 0)
	a := strings.Split(from, "\n")
	b := strings.Split(to, "\n")
	p := ansi.NewParser()
	for i := range b {
		if i < len(a) && a[i] == b[i] {
			continue
		}
		var prev string
		if i < len(a) {
			prev = a[i]
		}
		b[i] = fadeLine(p, prev, b[i], t, bg, fg)
	}
	return strings.Join(b, "\n")
}

type cellStyle struct {
	fg, bg color.Color // nil: terminal default
	attrs  string      // other SGR params in order since the last reset
}

type cell struct {
	pre   string // non-SGR escape sequences before the grapheme, kept verbatim
	text  string
	width int
	col   int
	style cellStyle
}

func fadeLine(p *ansi.Parser, from, to string, t float64, bg, fg color.Color) string {
	old := map[int]cell{}
	fc, _ := parseCells(p, from)
	for _, c := range fc {
		old[c.col] = c
	}
	cells, tail := parseCells(p, to)
	changed := false
	for i, c := range cells {
		o, ok := old[c.col]
		if ok && o.text == c.text && o.width == c.width && sameStyle(o.style, c.style) {
			continue
		}
		changed = true
		cells[i].style.fg = blend(bg, cmp(c.style.fg, fg), t)
		if c.style.bg != nil {
			cells[i].style.bg = blend(bg, c.style.bg, t)
		}
	}
	if !changed {
		return to
	}
	var sb strings.Builder
	cur := cellStyle{}
	for _, c := range cells {
		sb.WriteString(c.pre)
		if !sameStyle(cur, c.style) {
			sb.WriteString(sgr(c.style))
			cur = c.style
		}
		sb.WriteString(c.text)
	}
	if !sameStyle(cur, cellStyle{}) {
		sb.WriteString("\x1b[m")
	}
	sb.WriteString(tail)
	return sb.String()
}

func cmp(c, fallback color.Color) color.Color {
	if c == nil {
		return fallback
	}
	return c
}

// parseCells splits one line into graphemes with their effective SGR style.
// Lines are treated as self-contained: style starts at the terminal default.
func parseCells(p *ansi.Parser, line string) (cells []cell, tail string) {
	p.Reset()
	var state byte
	var st cellStyle
	var pre strings.Builder
	col := 0
	for len(line) > 0 {
		seq, width, n, newState := ansi.DecodeSequence(line, state, p)
		state = newState
		line = line[n:]
		switch {
		case width > 0:
			cells = append(cells, cell{pre: pre.String(), text: seq, width: width, col: col, style: st})
			pre.Reset()
			col += width
		case ansi.HasCsiPrefix(seq) && ansi.Cmd(p.Command()) == 'm':
			applySGR(&st, p.Params())
		default:
			pre.WriteString(seq)
		}
	}
	return cells, pre.String()
}

func applySGR(st *cellStyle, params ansi.Params) {
	if len(params) == 0 {
		*st = cellStyle{}
		return
	}
	var attrs []string
	if st.attrs != "" {
		attrs = []string{st.attrs}
	}
	for i := 0; i < len(params); {
		v := params[i].Param(0)
		switch {
		case v == 0:
			*st, attrs = cellStyle{}, nil
		case v == 38 || v == 48 || v == 58:
			var c color.Color
			n := ansi.ReadStyleColor(params[i:], &c)
			if n == 0 {
				i++
				continue
			}
			switch v {
			case 38:
				st.fg = c
			case 48:
				st.bg = c
			default:
				attrs = append(attrs, colorParams(58, c))
			}
			i += n
			continue
		case v >= 30 && v <= 37:
			st.fg = ansi.BasicColor(v - 30)
		case v >= 90 && v <= 97:
			st.fg = ansi.BasicColor(v - 90 + 8)
		case v == 39:
			st.fg = nil
		case v >= 40 && v <= 47:
			st.bg = ansi.BasicColor(v - 40)
		case v >= 100 && v <= 107:
			st.bg = ansi.BasicColor(v - 100 + 8)
		case v == 49:
			st.bg = nil
		default:
			// Keep the attribute with its colon sub-parameters, e.g. 4:3.
			group := strconv.Itoa(v)
			for params[i].HasMore() && i+1 < len(params) {
				i++
				group += ":" + strconv.Itoa(params[i].Param(0))
			}
			attrs = append(attrs, group)
		}
		i++
	}
	st.attrs = strings.Join(attrs, ";")
}

func colorParams(base int, c color.Color) string {
	r, g, b, _ := c.RGBA()
	return strconv.Itoa(base) + ";2;" + strconv.Itoa(int(r>>8)) + ";" + strconv.Itoa(int(g>>8)) + ";" + strconv.Itoa(int(b>>8))
}

func sgr(st cellStyle) string {
	parts := []string{"0"}
	if st.attrs != "" {
		parts = append(parts, st.attrs)
	}
	if st.fg != nil {
		parts = append(parts, colorParams(38, st.fg))
	}
	if st.bg != nil {
		parts = append(parts, colorParams(48, st.bg))
	}
	return "\x1b[" + strings.Join(parts, ";") + "m"
}

func sameColor(a, b color.Color) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()
	return ar == br && ag == bg && ab == bb && aa == ba
}

func sameStyle(a, b cellStyle) bool {
	return a.attrs == b.attrs && sameColor(a.fg, b.fg) && sameColor(a.bg, b.bg)
}

// blend mixes from toward to in Lab space and returns an opaque RGB color.
func blend(from, to color.Color, t float64) color.Color {
	if from == nil || to == nil {
		return to
	}
	a, ok1 := colorful.MakeColor(from)
	b, ok2 := colorful.MakeColor(to)
	if !ok1 || !ok2 {
		return to
	}
	r, g, bl := a.BlendLab(b, t).Clamped().RGB255()
	return color.RGBA{R: r, G: g, B: bl, A: 0xff}
}
