package render

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/charmbracelet/x/ansi"

	"tuipresent/internal/deck"
)

// Big titles. title_size 2 and 3 draw the slide title in block letters: a
// 5-pixel-tall font drawn with half blocks (size 2, 3 rows) or double-wide
// full blocks (size 3, 5 rows). Frame, -check and -dump always show these.
// Where the terminal speaks kitty's text-sizing protocol (OSC 66) the
// presenter draws the same title as scaled text over the block letters (see
// BigTitleAt), at the same row cost, so layout and -check agree.

// TextSizing reports whether the terminal scales text with OSC 66. The zero
// value is the block-letter fallback, which is what -dump and -check use;
// the presenter sets it from DetectTextSizing.
var TextSizing bool

// DetectTextSizing reports whether the terminal named by the environment is
// known to implement kitty's text-sizing protocol: kitty, outside tmux and
// screen. Other terminals are not guessed at. TUIPRESENT_TEXT_SIZING=on forces
// it (even where this does not detect support) and =off disables it.
func DetectTextSizing(getenv func(string) string) bool {
	switch strings.ToLower(strings.TrimSpace(getenv("TUIPRESENT_TEXT_SIZING"))) {
	case "on", "1", "true", "yes":
		return true
	case "off", "0", "false", "no":
		return false
	}
	term := getenv("TERM")
	if getenv("TMUX") != "" || strings.HasPrefix(term, "screen") || strings.HasPrefix(term, "tmux") {
		return false
	}
	return strings.EqualFold(getenv("TERM_PROGRAM"), "kitty") || strings.Contains(term, "kitty") || getenv("KITTY_WINDOW_ID") != ""
}

// TitleRows is the number of rows a title of the given size occupies.
func TitleRows(size int) int {
	switch size {
	case 2:
		return 3
	case 3:
		return 5
	}
	return 1
}

// glyphs is the block font: 5 pixel rows, variable width, '#' is ink.
var glyphs = map[rune][5]string{
	'A':  {".###.", "#...#", "#####", "#...#", "#...#"},
	'B':  {"####.", "#...#", "####.", "#...#", "####."},
	'C':  {".####", "#....", "#....", "#....", ".####"},
	'D':  {"####.", "#...#", "#...#", "#...#", "####."},
	'E':  {"#####", "#....", "###..", "#....", "#####"},
	'F':  {"#####", "#....", "###..", "#....", "#...."},
	'G':  {".####", "#....", "#..##", "#...#", ".###."},
	'H':  {"#...#", "#...#", "#####", "#...#", "#...#"},
	'I':  {"#####", "..#..", "..#..", "..#..", "#####"},
	'J':  {"..###", "...#.", "...#.", "#..#.", ".##.."},
	'K':  {"#...#", "#..#.", "###..", "#..#.", "#...#"},
	'L':  {"#....", "#....", "#....", "#....", "#####"},
	'M':  {"#...#", "##.##", "#.#.#", "#...#", "#...#"},
	'N':  {"#...#", "##..#", "#.#.#", "#..##", "#...#"},
	'O':  {".###.", "#...#", "#...#", "#...#", ".###."},
	'P':  {"####.", "#...#", "####.", "#....", "#...."},
	'Q':  {".###.", "#...#", "#.#.#", "#..#.", ".##.#"},
	'R':  {"####.", "#...#", "####.", "#..#.", "#...#"},
	'S':  {".####", "#....", ".###.", "....#", "####."},
	'T':  {"#####", "..#..", "..#..", "..#..", "..#.."},
	'U':  {"#...#", "#...#", "#...#", "#...#", ".###."},
	'V':  {"#...#", "#...#", "#...#", ".#.#.", "..#.."},
	'W':  {"#...#", "#...#", "#.#.#", "##.##", "#...#"},
	'X':  {"#...#", ".#.#.", "..#..", ".#.#.", "#...#"},
	'Y':  {"#...#", ".#.#.", "..#..", "..#..", "..#.."},
	'Z':  {"#####", "...#.", "..#..", ".#...", "#####"},
	'0':  {".###.", "#..##", "#.#.#", "##..#", ".###."},
	'1':  {"..#..", ".##..", "..#..", "..#..", ".###."},
	'2':  {".###.", "#...#", "..##.", ".#...", "#####"},
	'3':  {"####.", "....#", ".###.", "....#", "####."},
	'4':  {"#..#.", "#..#.", "#####", "...#.", "...#."},
	'5':  {"#####", "#....", "####.", "....#", "####."},
	'6':  {".###.", "#....", "####.", "#...#", ".###."},
	'7':  {"#####", "....#", "...#.", "..#..", "..#.."},
	'8':  {".###.", "#...#", ".###.", "#...#", ".###."},
	'9':  {".###.", "#...#", ".####", "....#", ".###."},
	' ':  {"...", "...", "...", "...", "..."},
	'.':  {".", ".", ".", ".", "#"},
	',':  {".", ".", ".", "#", "#"},
	':':  {".", "#", ".", "#", "."},
	'!':  {"#", "#", "#", ".", "#"},
	'?':  {".###.", "#...#", "..##.", ".....", "..#.."},
	'-':  {"...", "...", "###", "...", "..."},
	'_':  {".....", ".....", ".....", ".....", "#####"},
	'/':  {"....#", "...#.", "..#..", ".#...", "#...."},
	'+':  {"...", ".#.", "###", ".#.", "..."},
	'=':  {"...", "###", "...", "###", "..."},
	'\'': {"#", "#", ".", ".", "."},
	'"':  {"#.#", "#.#", "...", "...", "..."},
	'&':  {".##..", "#..#.", ".##.#", "#..#.", ".##.#"},
	'(':  {"..#", ".#.", ".#.", ".#.", "..#"},
	')':  {"#..", ".#.", ".#.", ".#.", "#.."},
}

// bigHead is where a big title sits within its head lines, for centering and
// for the scaled-text overlay.
type bigHead struct {
	first  int      // index of the first block row in the head lines
	rows   int      // TitleRows(size)
	xoff   int      // columns before the block on those lines (2 inside a frame)
	blockW int      // cells the block letters span
	lineW  int      // cells of the widest head line (what centering uses)
	scale  int      // OSC 66 scale: one cell per row of the title
	text   []rune   // upper-cased title, one entry per drawn glyph
	colors []string // hex color of each rune
}

// titleColors is the color of each rune of the title's runs: the gradient,
// except code and highlight runs, which use their theme colors.
func (r *Renderer) titleColors(runs []run) (text []rune, colors []string) {
	p := r.theme.Palette
	n := len([]rune(plainText(runs)))
	grad := r.gradientAt(n)
	i := 0
	for _, ru := range runs {
		for _, ch := range ru.text {
			text = append(text, unicode.ToUpper(ch))
			switch {
			case ru.code:
				colors = append(colors, p.Code)
			case ru.hi:
				colors = append(colors, p.Highlight)
			default:
				colors = append(colors, grad(i))
			}
			i++
		}
	}
	return text, colors
}

// blockRows draws text in block letters of the given size (2 or 3), colored
// per rune. ok is false when a rune has no glyph; bad is that rune.
func blockRows(text []rune, colors []string, size int) (rows []string, width int, bad rune, ok bool) {
	var cols [5][]bool
	var owner []int
	for k, ch := range text {
		g, found := glyphs[ch]
		if !found {
			return nil, 0, ch, false
		}
		if k > 0 {
			for y := range cols {
				cols[y] = append(cols[y], false)
			}
			owner = append(owner, k-1)
		}
		for x := range len(g[0]) {
			for y := range cols {
				cols[y] = append(cols[y], g[y][x] == '#')
			}
			owner = append(owner, k)
		}
	}
	n := len(owner)
	cell := func(on bool, col string, s string) string {
		if !on {
			return strings.Repeat(" ", ansi.StringWidth(s))
		}
		return sgrColor(38, col) + s + "\x1b[m"
	}
	if size == 2 {
		for y := 0; y < 5; y += 2 {
			var sb strings.Builder
			for x := range n {
				top := cols[y][x]
				bot := y+1 < 5 && cols[y+1][x]
				ch := " "
				switch {
				case top && bot:
					ch = "█"
				case top:
					ch = "▀"
				case bot:
					ch = "▄"
				}
				sb.WriteString(cell(top || bot, colors[owner[x]], ch))
			}
			rows = append(rows, sb.String())
		}
		return rows, n, 0, true
	}
	for y := range 5 {
		var sb strings.Builder
		for x := range n {
			sb.WriteString(cell(cols[y][x], colors[owner[x]], "██"))
		}
		rows = append(rows, sb.String())
	}
	return rows, 2 * n, 0, true
}

// bigHead lays out slide s's title at the given size, in the same shape as
// the normal head: blank row, title rows, optional subtitle, rule (or the
// frame's bottom), blank row. It returns reason instead when the title cannot
// be drawn big: no glyph for a character, or wider than the content width cw.
// curly is a rule here: an underline on block letters means nothing.
func (r *Renderer) bigHead(s deck.Slide, size int, style string, cw int) (lines []string, width int, bh *bigHead, reason string) {
	runs := parseInline(s.Title)
	text, colors := r.titleColors(runs)
	rows, bw, bad, ok := blockRows(text, colors, size)
	if !ok {
		return nil, 0, nil, fmt.Sprintf("no block letter for %q", bad)
	}
	sw := ansi.StringWidth(PlainTitle(s.Subtitle))
	inner := max(bw, sw)
	boxed := style == "frame"
	total := inner
	if boxed {
		total += 4
	}
	if total > cw {
		return nil, 0, nil, fmt.Sprintf("it needs %d columns but the slide is %d wide", total, cw)
	}
	var sub string
	if s.Subtitle != "" {
		sub = r.styledSubtitle(s.Subtitle)
	}
	bh = &bigHead{rows: len(rows), blockW: bw, scale: TitleRows(size), text: text, colors: colors}
	if boxed {
		border := func(l string) string { return r.gradient(l, false) }
		side := r.style(r.theme.Palette.Accent).Render("│")
		row := func(content string, w int) string {
			return side + " " + content + strings.Repeat(" ", inner-w) + " " + side
		}
		lines = []string{"", border("╭" + strings.Repeat("─", total-2) + "╮")}
		bh.first, bh.xoff = len(lines), 2
		for _, l := range rows {
			lines = append(lines, row(l, bw))
		}
		if sub != "" {
			lines = append(lines, row(sub, sw))
		}
		lines = append(lines, border("╰"+strings.Repeat("─", total-2)+"╯"), "")
		bh.lineW = total
		return lines, total, bh, ""
	}
	lines = []string{""}
	bh.first = len(lines)
	lines = append(lines, rows...)
	if sub != "" {
		lines = append(lines, sub)
	}
	rule := cw
	if s.Centered {
		rule = min(inner+4, cw)
	}
	bh.lineW = inner
	return append(lines, r.rule(rule), ""), inner, bh, ""
}

// TitleSizeWarnings lists the slides whose title_size cannot be honored on a
// w-column screen, each with the reason; those slides draw the normal title.
// The deck parser cannot know: it depends on the width.
func (r *Renderer) TitleSizeWarnings(w int) []string {
	var out []string
	cw := contentWidth(w)
	for i, s := range r.deck.Slides {
		if s.Title == "" || s.Cover {
			continue
		}
		size := r.deck.TitleSize(i)
		if size < 2 {
			continue
		}
		if _, _, _, reason := r.bigHead(s, size, r.deck.TitleStyle(i), cw); reason != "" {
			out = append(out, fmt.Sprintf("slide %d: title_size %d does not fit (%s); using the normal title", i+1, size, reason))
		}
	}
	return out
}

// BigTitle is the scaled-text overlay for a slide's big title.
type BigTitle struct {
	Draw  string // erases the block letters, then draws the scaled title
	Erase string // blanks the block-letter area
}

// BigTitleAt returns the OSC 66 overlay that replaces the block letters of
// slide at step on a w x h screen, or false when the slide has no big title
// there. The presenter writes it after the frame is painted, and Erase before
// anything else is drawn over that area. The overlay occupies the cells the
// block letters do (the scaled title is never wider), so the layout is the
// same on both paths.
func (r *Renderer) BigTitleAt(slide, step, w, h int) (BigTitle, bool) {
	r.Frame(slide, step, w, h)
	pl, ok := r.places[[4]int{slide, step, w, h}]
	if !ok {
		return BigTitle{}, false
	}
	bh := pl.bh
	p := r.theme.Palette
	blank := "\x1b[m"
	if r.theme.Paint {
		blank = "\x1b[m" + sgrColor(38, p.Foreground) + sgrColor(48, p.Background)
	}
	var sb strings.Builder
	sb.WriteString("\x1b7" + blank)
	for y := range bh.rows {
		sb.WriteString(ansi.CursorPosition(pl.col+1, pl.row+y+1) + strings.Repeat(" ", bh.blockW))
	}
	sb.WriteString("\x1b8")
	erase := sb.String()

	sb.Reset()
	sb.WriteString(erase)
	sb.WriteString("\x1b7" + ansi.CursorPosition(pl.col+1, pl.row+1) + blank)
	for i, ch := range bh.text {
		if ch == ' ' {
			sb.WriteString(ansi.CursorForward(bh.scale))
			continue
		}
		if r.theme.Paint {
			sb.WriteString(sgrColor(48, p.Background))
		}
		sb.WriteString(sgrColor(38, bh.colors[i]) + "\x1b[1m" + fmt.Sprintf("\x1b]66;s=%d;%c\x1b\\", bh.scale, ch) + "\x1b[m")
	}
	sb.WriteString("\x1b8")
	return BigTitle{Draw: sb.String(), Erase: erase}, true
}

// place is where a big title sits in a rendered frame.
type place struct {
	bh       *bigHead
	row, col int
}
