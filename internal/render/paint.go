package render

import (
	"image/color"
	"strconv"
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/lucasb-eyer/go-colorful"
)

// sgrColor returns the SGR sequence setting the foreground (base 38) or
// background (base 48) to a #rrggbb color, or "" if hex is empty or invalid.
func sgrColor(base int, hex string) string {
	if hex == "" {
		return ""
	}
	c, err := colorful.Hex(hex)
	if err != nil {
		return ""
	}
	r, g, b := c.RGB255()
	return "\x1b[" + strconv.Itoa(base) + ";2;" + strconv.Itoa(int(r)) + ";" + strconv.Itoa(int(g)) + ";" + strconv.Itoa(int(b)) + "m"
}

// paint makes fg and bg (#rrggbb; fg may be "") the default colors of line:
// it sets them up front and again after every SGR sequence that resets them
// (0, 39, 49), so glamour's and chroma's resets don't punch holes in the
// painted area. Explicit colors in the line, such as the foreground that
// encodes a kitty image placeholder's id, are left alone. Geometry is
// unchanged: only escape sequences are added.
func paint(line, fg, bg string) string {
	fgSeq, bgSeq := sgrColor(38, fg), sgrColor(48, bg)
	if fgSeq == "" && bgSeq == "" {
		return line
	}
	var sb strings.Builder
	sb.Grow(len(line) + 64)
	sb.WriteString(fgSeq)
	sb.WriteString(bgSeq)
	p := ansi.NewParser()
	var state byte
	for len(line) > 0 {
		seq, _, n, newState := ansi.DecodeSequence(line, state, p)
		state = newState
		line = line[n:]
		sb.WriteString(seq)
		if !ansi.HasCsiPrefix(seq) || ansi.Cmd(p.Command()) != 'm' {
			continue
		}
		fgReset, bgReset := sgrResets(p.Params())
		if fgReset {
			sb.WriteString(fgSeq)
		}
		if bgReset {
			sb.WriteString(bgSeq)
		}
	}
	sb.WriteString("\x1b[m")
	return sb.String()
}

// sgrResets reports whether an SGR sequence leaves the foreground or
// background at the terminal default.
func sgrResets(params ansi.Params) (fg, bg bool) {
	if len(params) == 0 {
		return true, true
	}
	for i := 0; i < len(params); {
		v := params[i].Param(0)
		switch {
		case v == 0:
			fg, bg = true, true
		case v == 38 || v == 48 || v == 58:
			var c color.Color
			n := ansi.ReadStyleColor(params[i:], &c)
			if v == 38 {
				fg = false
			} else if v == 48 {
				bg = false
			}
			if n > 0 {
				i += n
				continue
			}
		case v == 39:
			fg = true
		case v == 49:
			bg = true
		case v >= 30 && v <= 37, v >= 90 && v <= 97:
			fg = false
		case v >= 40 && v <= 47, v >= 100 && v <= 107:
			bg = false
		}
		i++
	}
	return fg, bg
}
