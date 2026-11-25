// Package render turns a slide at a reveal step into a fixed-size frame.
package render

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"charm.land/glamour/v2"
	gansi "charm.land/glamour/v2/ansi"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/lucasb-eyer/go-colorful"

	"github.com/solarisjon/tuipresent/internal/deck"
	"github.com/solarisjon/tuipresent/internal/img"
)

const (
	maxContentWidth = 110
	columnGap       = 4
)

type Renderer struct {
	deck       *deck.Deck
	theme      Theme
	glam       map[int]*glamour.TermRenderer
	cache      map[[4]int]string
	places     map[[4]int]place // big-title placement of each cached frame
	execOutput map[int]string
	clip       int // widest column overflow seen since Overflow reset it
	imgRows    int // max rows per image on the slide being laid out; 0 while measuring
	images     map[string]*loadedImage
	themeWarn  []string
	themeErr   error
}

// New renders d in the theme its front matter selects. Theme problems never
// stop rendering: see ThemeWarnings and ThemeErr.
func New(d *deck.Deck) *Renderer {
	th, warns, err := ResolveTheme(d.Meta.Theme, d.Dir)
	return &Renderer{
		deck:       d,
		theme:      th,
		themeWarn:  warns,
		themeErr:   err,
		glam:       map[int]*glamour.TermRenderer{},
		cache:      map[[4]int]string{},
		places:     map[[4]int]place{},
		execOutput: map[int]string{},
		images:     map[string]*loadedImage{},
	}
}

func (r *Renderer) SetExecOutput(id int, out string) {
	r.execOutput[id] = out
	clear(r.cache)
	clear(r.places)
}

// Frame returns exactly h lines, each exactly w cells wide.
func (r *Renderer) Frame(slide, step, w, h int) string {
	key := [4]int{slide, step, w, h}
	if f, ok := r.cache[key]; ok {
		return f
	}
	f := r.frame(slide, step, w, h)
	r.cache[key] = f
	return f
}

func (r *Renderer) frame(idx, step, w, h int) string {
	s := r.deck.Slides[idx]
	cw := contentWidth(w)
	left := (w - cw) / 2

	var footer []string
	if h >= footerMinHeight {
		footer = r.footer(idx, w)
	}
	avail := max(h-len(footer), 0)

	head, _, bh := r.headBig(idx, s, cw)

	headRows := len(head)
	if !s.Centered {
		headRows++
	}
	r.sizeImages(s, cw, avail-headRows)

	var body []string
	if s.Cover {
		body = r.cover(cw)
	} else {
		body = r.body(s.Blocks, step, cw)
	}
	if s.Centered && !s.Cover {
		// Center head lines individually and the body as one block, offset from the
		// fully revealed body so pauses don't shift it sideways.
		for i, l := range head {
			lw := visibleWidth(l)
			if bh != nil && i >= bh.first && i < bh.first+bh.rows {
				lw = bh.lineW // block rows have ragged trailing spaces: center them as one block
			}
			head[i] = padLeft(l, (cw-lw)/2)
		}
		if statementOnly(s.Blocks) {
			for i, l := range body {
				body[i] = padLeft(l, (cw-visibleWidth(l))/2)
			}
		} else {
			off := 0
			for _, l := range r.bodyFull(s, cw) {
				off = max(off, visibleWidth(l))
			}
			off = max((cw-off)/2, 0)
			for i, l := range body {
				body[i] = padLeft(l, off)
			}
		}
	}

	top := []string{}
	if !s.Centered {
		top = append(top, "")
	}
	top = append(top, head...)
	if s.Centered {
		full := len(head) + len(r.bodyFull(s, cw))
		pad := max((avail-full)/2, 0)
		top = append(make([]string, pad), head...)
	}

	if bh != nil && bh.first+bh.rows+len(top)-len(head) <= avail {
		pad := 0
		if s.Centered {
			pad = (cw - bh.lineW) / 2
		}
		r.places[[4]int{idx, step, w, h}] = place{bh: bh, row: len(top) - len(head) + bh.first, col: left + pad + bh.xoff}
	}

	lines := append(top, body...)
	if len(lines) > avail {
		lines = lines[:avail]
	}
	for len(lines) < avail {
		lines = append(lines, "")
	}
	indent := strings.Repeat(" ", left)
	for i, l := range lines {
		lines[i] = fit(indent+l, w)
	}
	for _, l := range footer {
		lines = append(lines, fit(l, w))
	}
	if r.theme.Paint {
		for i, l := range lines {
			lines[i] = paint(l, r.theme.Palette.Foreground, r.theme.Palette.Background)
		}
	}
	return strings.Join(lines, "\n")
}

const footerMinHeight = 6

var structuredLineRe = regexp.MustCompile("^\\s*([-*+] |\\d+[.)] |\\||```|~~~|>)")

// statementOnly reports whether a slide is plain prose and headings, which read
// best centered line by line; lists, tables, quotes and code keep block alignment.
func statementOnly(blocks []deck.Block) bool {
	for _, b := range blocks {
		if b.Kind != deck.Markdown {
			return false
		}
		for _, l := range strings.Split(b.Text, "\n") {
			if structuredLineRe.MatchString(l) {
				return false
			}
		}
	}
	return true
}

func visibleWidth(s string) int {
	return ansi.StringWidth(strings.TrimRight(ansi.Strip(s), " "))
}

func padLeft(s string, n int) string {
	if n <= 0 {
		return s
	}
	return strings.Repeat(" ", n) + s
}

// contentWidth is the width slide content is laid out in: w less a 4-cell
// margin each side, at most maxContentWidth. Narrow screens keep at least
// 20 cells (or w - 2 when that is less), so the width never shrinks as w grows.
func contentWidth(w int) int {
	return max(min(w-8, maxContentWidth), min(w-2, 20), 1)
}

// ContentWidth is the content width of a w-cell-wide screen.
func ContentWidth(w int) int { return contentWidth(w) }

// Overflow reports how many rows and columns the fully revealed slide exceeds a w×h screen by.
func (r *Renderer) Overflow(idx, w, h int) (rows, cols int) {
	s := r.deck.Slides[idx]
	cw := contentWidth(w)
	avail := h
	if h >= footerMinHeight {
		avail -= 2
	}
	used := 0
	if !s.Centered {
		used++
	}
	head, headW := r.head(idx, s, cw)
	used += len(head)
	r.sizeImages(s, cw, avail-used)
	r.clip = 0
	body := r.bodyFull(s, cw)
	used += len(body)
	cols = max(headW-cw, r.clip)
	for _, l := range body {
		cols = max(cols, ansi.StringWidth(l)-cw)
	}
	if !s.Cover {
		cols = max(cols, codeOverflow(s.Blocks, cw, r.execOutput))
	}
	return max(used-avail, 0), max(cols, 0)
}

// codeOverflow reports how far fenced code lines exceed the width glamour renders
// them at without wrapping (wrap width minus its 2-cell code margin).
func codeOverflow(blocks []deck.Block, width int, execOut map[int]string) int {
	limit := max(width-2, 10) - 2
	over := 0
	measure := func(code string) {
		for _, l := range strings.Split(strings.ReplaceAll(code, "\t", "    "), "\n") {
			over = max(over, ansi.StringWidth(l)-limit)
		}
	}
	for _, b := range blocks {
		switch b.Kind {
		case deck.Markdown:
			measure(fencedCode(b.Text))
		case deck.Code:
			measure(b.Text)
		case deck.Exec:
			measure(b.Text)
			measure(execOut[b.ExecID])
		case deck.Columns:
			for i, w := range columnWidths(b, width) {
				over = max(over, codeOverflow(b.Children[i], w, execOut))
			}
		}
	}
	return over
}

// fencedCode returns the lines inside fenced code blocks in md, using the
// parser's fence rules, so lines of a nested fence count as code.
func fencedCode(md string) string {
	var out []string
	fence := ""
	for _, l := range strings.Split(md, "\n") {
		if fence == "" {
			if m, _, ok := deck.FenceOpen(l); ok {
				fence = m
			}
			continue
		}
		if deck.ClosesFence(fence, l) {
			fence = ""
			continue
		}
		out = append(out, l)
	}
	return strings.Join(out, "\n")
}

func (r *Renderer) bodyFull(s deck.Slide, cw int) []string {
	if s.Cover {
		return r.cover(cw)
	}
	return r.body(s.Blocks, s.Steps-1, cw)
}

func (r *Renderer) body(blocks []deck.Block, step, width int) []string {
	var out []string
	var md []string
	afterColumns := false
	flushMD := func() {
		if len(md) == 0 {
			return
		}
		if afterColumns {
			out = append(out, "")
			afterColumns = false
		}
		out = append(out, r.markdown(strings.Join(md, "\n"), width)...)
		md = nil
	}
	for _, b := range blocks {
		if b.Step > step {
			continue
		}
		switch b.Kind {
		case deck.Markdown:
			md = append(md, b.Text)
		case deck.Code:
			flushMD()
			out = r.appendCode(out, b, step, width)
			afterColumns = true
		case deck.Exec:
			if len(b.Highlights) > 0 {
				flushMD()
				out = r.appendCode(out, b, step, width)
				afterColumns = true
			} else {
				md = append(md, "", "```"+b.Lang, b.Text, "```", "")
			}
			if o, ok := r.execOutput[b.ExecID]; ok {
				md = append(md, "```text", strings.TrimRight(o, "\n"), "```", "")
			}
		case deck.Columns:
			flushMD()
			out = append(out, r.columns(b, step, width)...)
			afterColumns = true
		case deck.Image:
			flushMD()
			if len(out) > 0 {
				out = append(out, "")
			}
			out = append(out, r.image(b, width)...)
			afterColumns = true
		}
	}
	flushMD()
	return out
}

func columnWidths(b deck.Block, width int) []int {
	return ColumnWidths(b.Widths, width)
}

// ColumnWidths is the width of each column of a column_layout with these
// relative widths, laid out in width cells.
func ColumnWidths(weights []int, width int) []int {
	total := 0
	for _, wt := range weights {
		total += wt
	}
	avail := width - columnGap*(len(weights)-1)
	out := make([]int, len(weights))
	for i, wt := range weights {
		out[i] = max(avail*wt/total, 1)
	}
	return out
}

func (r *Renderer) columns(b deck.Block, step, width int) []string {
	var cols []string
	// Images in a column may also use the rows its taller neighbours leave free.
	widths := columnWidths(b, width)
	baseRows, extra := r.imgRows, make([]int, len(widths))
	if baseRows > 0 {
		r.imgRows = 0
		tallest := 0
		for i, cw := range widths {
			extra[i] = len(r.body(b.Children[i], 1<<30, cw))
			tallest = max(tallest, extra[i])
		}
		for i := range extra {
			extra[i] = (tallest - extra[i]) / max(imageStack(b.Children[i]), 1)
		}
	}
	defer func() { r.imgRows = baseRows }()
	for i, cw := range widths {
		if baseRows > 0 {
			r.imgRows = baseRows + extra[i]
		}
		// Height from the fully revealed column keeps layout stable across pauses.
		full := len(r.body(b.Children[i], 1<<30, cw))
		lines := r.body(b.Children[i], step, cw)
		for len(lines) < full {
			lines = append(lines, "")
		}
		for j := range lines {
			r.clip = max(r.clip, ansi.StringWidth(lines[j])-cw)
			lines[j] = fit(lines[j], cw)
		}
		cols = append(cols, strings.Join(lines, "\n"))
		if i < len(b.Widths)-1 {
			cols = append(cols, strings.Repeat(" ", columnGap))
		}
	}
	return strings.Split(lipgloss.JoinHorizontal(lipgloss.Top, cols...), "\n")
}

// markdown renders prose and top-level fenced code separately so list rewrapping
// (hangIndent) never touches code lines that happen to look like list items.
func (r *Renderer) markdown(src string, width int) []string {
	var out []string
	for _, seg := range splitFences(src) {
		var groups [][]string
		if seg.code {
			groups = append(groups, r.codePanel(r.glamour(seg.text, width), width))
		} else {
			for _, pc := range splitCallouts(seg.text) {
				if pc.kind != "" {
					groups = append(groups, r.callout(pc.kind, pc.text, width))
					continue
				}
				lines := r.applyMarks(r.glamour(markText(pc.text), width))
				groups = append(groups, r.decorate(hangIndent(lines, max(width-2, 10))))
			}
		}
		for _, lines := range groups {
			if len(lines) == 0 {
				continue
			}
			if len(out) > 0 {
				out = append(out, "")
			}
			out = append(out, lines...)
		}
	}
	return out
}

type segment struct {
	text string
	code bool
}

var topFenceRe = regexp.MustCompile("^ {0,3}(```+|~~~+)")

func splitFences(src string) []segment {
	var segs []segment
	var cur []string
	fence := ""
	flush := func(code bool) {
		if strings.TrimSpace(strings.Join(cur, "")) != "" {
			segs = append(segs, segment{strings.Join(cur, "\n"), code})
		}
		cur = nil
	}
	for _, l := range strings.Split(src, "\n") {
		m := topFenceRe.FindStringSubmatch(l)
		switch {
		case fence == "" && m != nil:
			flush(false)
			fence = m[1]
			cur = append(cur, l)
		case fence != "" && m != nil && strings.HasPrefix(m[1], fence) && strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(l), fence[:1])) == "":
			cur = append(cur, l)
			flush(true)
			fence = ""
		default:
			cur = append(cur, l)
		}
	}
	flush(fence != "")
	return segs
}

func (r *Renderer) glamour(src string, width int) []string {
	// Glamour lets trailing punctuation overhang the wrap width; leave slack so fit() never clips it.
	width = max(width-2, 10)
	g, ok := r.glam[width]
	if !ok {
		var err error
		g, err = glamour.NewTermRenderer(glamour.WithStyles(r.glamourStyle()), glamour.WithWordWrap(width), glamour.WithChromaFormatter("terminal16m"))
		if err != nil {
			return strings.Split(src, "\n")
		}
		r.glam[width] = g
	}
	// Raw tabs pass through glamour and misalign frames; CommonMark tab stops are 4.
	src = strings.ReplaceAll(src, "\t", "    ")
	var urls []string
	if Hyperlinks {
		src, urls = hideLinkURLs(src)
	}
	s, err := g.Render(src)
	if err != nil {
		return strings.Split(src, "\n")
	}
	lines := strings.Split(s, "\n")
	if Hyperlinks {
		lines = finishLinks(lines, urls)
	} else if strings.Contains(s, "\x1b]8;") {
		for i, l := range lines {
			lines[i] = StripLinks(l)
		}
	}
	for len(lines) > 0 && strings.TrimSpace(ansi.Strip(lines[0])) == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && strings.TrimSpace(ansi.Strip(lines[len(lines)-1])) == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// codePanel paints a fenced code block's lines on the code background, as a
// panel two cells wider than its longest line (glamour already indents code
// by two cells), so code stands apart from prose in every theme.
func (r *Renderer) codePanel(lines []string, width int) []string {
	pw := 0
	for _, l := range lines {
		pw = max(pw, visibleWidth(l))
	}
	pw = min(pw+2, width)
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = paint(fit(l, pw), "", r.theme.Palette.CodeBackground)
	}
	return out
}

const boxChars = "─│┼├┤┬┴╭╮╰╯┌┐└┘"

// decorate colors what glamour can't style per theme: list bullets and
// numbers get the bullet color, and table borders (the only box-drawing
// characters glamour leaves at the default foreground) get the border color.
func (r *Renderer) decorate(lines []string) []string {
	bullet := sgrColor(38, r.theme.Palette.Bullet)
	border := sgrColor(38, r.theme.Palette.Border)
	out := make([]string, len(lines))
	for i, l := range lines {
		if m := listMarkerRe.FindStringSubmatch(ansi.Strip(l)); m != nil {
			lead, pw := len(m[1]), ansi.StringWidth(m[0])
			marker := strings.TrimRight(m[2], " ")
			l = ansi.Cut(l, 0, lead) + bullet + marker + "\x1b[39m" + strings.Repeat(" ", pw-lead-ansi.StringWidth(marker)) + ansi.Cut(l, pw, ansi.StringWidth(l))
		}
		out[i] = recolorBorders(l, border)
	}
	return out
}

// recolorBorders wraps runs of box-drawing characters drawn in the default
// foreground in the color seq.
func recolorBorders(line, seq string) string {
	if seq == "" || !strings.ContainsAny(line, boxChars) {
		return line
	}
	var sb strings.Builder
	p := ansi.NewParser()
	var state byte
	fgSet, inRun := false, false
	for len(line) > 0 {
		s, width, n, newState := ansi.DecodeSequence(line, state, p)
		state = newState
		line = line[n:]
		if width == 0 {
			if inRun {
				sb.WriteString("\x1b[39m")
				inRun = false
			}
			if ansi.HasCsiPrefix(s) && ansi.Cmd(p.Command()) == 'm' {
				if reset, _ := sgrResets(p.Params()); reset {
					fgSet = false
				} else if sgrSetsFg(p.Params()) {
					fgSet = true
				}
			}
			sb.WriteString(s)
			continue
		}
		box := !fgSet && strings.Contains(boxChars, s)
		if box && !inRun {
			sb.WriteString(seq)
			inRun = true
		} else if !box && inRun {
			sb.WriteString("\x1b[39m")
			inRun = false
		}
		sb.WriteString(s)
	}
	if inRun {
		sb.WriteString("\x1b[39m")
	}
	return sb.String()
}

func sgrSetsFg(params ansi.Params) bool {
	for i := range params {
		v := params[i].Param(0)
		if v == 38 || (v >= 30 && v <= 37) || (v >= 90 && v <= 97) {
			return true
		}
	}
	return false
}

var listMarkerRe = regexp.MustCompile(`^( *)(• |\d+\. |\[[ x✓✗]\] )`)

// hangIndent rewraps list items so wrapped lines align under the item text
// instead of returning to the list's left edge, as glamour leaves them.
func hangIndent(lines []string, width int) []string {
	out := make([]string, 0, len(lines))
	for i := 0; i < len(lines); i++ {
		plain := ansi.Strip(lines[i])
		m := listMarkerRe.FindStringSubmatch(plain)
		if m == nil {
			out = append(out, lines[i])
			continue
		}
		lead := len(m[1])
		pw := ansi.StringWidth(m[0])
		parts := []string{strings.TrimRight(ansi.Cut(lines[i], pw, pw+visibleWidth(lines[i])-pw), " ")}
		j := i + 1
		for ; j < len(lines); j++ {
			p := ansi.Strip(lines[j])
			if strings.TrimSpace(p) == "" || listMarkerRe.MatchString(p) || len(p)-len(strings.TrimLeft(p, " ")) != lead {
				break
			}
			parts = append(parts, ansi.Cut(lines[j], lead, visibleWidth(lines[j])))
		}
		if j == i+1 {
			out = append(out, lines[i])
			continue
		}
		prefix := ansi.Cut(lines[i], 0, pw)
		indent := strings.Repeat(" ", pw)
		for k, l := range strings.Split(ansi.Wordwrap(strings.Join(parts, " "), max(width-pw, 1), ""), "\n") {
			if k == 0 {
				out = append(out, prefix+l)
			} else {
				out = append(out, indent+strings.TrimLeft(l, " "))
			}
		}
		i = j - 1
	}
	return out
}

// glamourStyle builds the whole glamour style from the theme palette, so
// every theme styles headings, quotes, tables, links and code the same way.
// Document margins are dropped: we lay out ourselves.
func (r *Renderer) glamourStyle() gansi.StyleConfig {
	p := r.theme.Palette
	str := func(s string) *string { return &s }
	t := true
	zero, two := uint(0), uint(2)
	prim := func(col string) gansi.StylePrimitive { return gansi.StylePrimitive{Color: str(col)} }
	head := func(prefix, col string) gansi.StyleBlock {
		return gansi.StyleBlock{StylePrimitive: gansi.StylePrimitive{Prefix: prefix, Color: str(col), Bold: &t}}
	}
	var c gansi.StyleConfig
	c.Document = gansi.StyleBlock{StylePrimitive: prim(p.Foreground), Margin: &zero}
	c.BlockQuote = gansi.StyleBlock{StylePrimitive: gansi.StylePrimitive{Color: str(p.Quote), Italic: &t}, Indent: func() *uint { u := uint(1); return &u }(), IndentToken: str("│ ")}
	c.List = gansi.StyleList{StyleBlock: gansi.StyleBlock{StylePrimitive: prim(p.Foreground)}, LevelIndent: 2}
	c.Heading = gansi.StyleBlock{StylePrimitive: gansi.StylePrimitive{BlockSuffix: "\n", Color: str(p.Subheading), Bold: &t}}
	c.H1 = head("", p.Heading)
	c.H2 = head("▌ ", p.Subheading)
	c.H3 = head("", p.Subheading)
	c.H4 = head("", p.Subheading)
	c.H5 = head("", p.Subheading)
	c.H6 = head("", p.Subheading)
	c.Strikethrough = gansi.StylePrimitive{CrossedOut: &t}
	c.Emph = gansi.StylePrimitive{Italic: &t}
	c.Strong = gansi.StylePrimitive{Bold: &t}
	c.HorizontalRule = gansi.StylePrimitive{Color: str(p.Border), Format: "\n--------\n"}
	c.Item = gansi.StylePrimitive{BlockPrefix: "• "}
	c.Enumeration = gansi.StylePrimitive{BlockPrefix: ". "}
	c.Task = gansi.StyleTask{Ticked: "[✓] ", Unticked: "[ ] "}
	c.Link = gansi.StylePrimitive{Color: str(p.Link), Underline: &t}
	if Hyperlinks {
		c.Link.Format = linkFormat
	} else {
		// The URL printed after link text is secondary: dim it.
		c.Link = gansi.StylePrimitive{Color: str(p.Dim)}
	}
	c.LinkText = gansi.StylePrimitive{Color: str(p.Link), Bold: &t}
	c.Image = gansi.StylePrimitive{Color: str(p.Link), Underline: &t}
	c.ImageText = gansi.StylePrimitive{Color: str(p.Dim), Format: "Image: {{.text}} →"}
	c.Code = gansi.StyleBlock{StylePrimitive: gansi.StylePrimitive{Prefix: " ", Suffix: " ", Color: str(p.Code), BackgroundColor: str(p.CodeBackground)}}
	c.CodeBlock = gansi.StyleCodeBlock{StyleBlock: gansi.StyleBlock{StylePrimitive: prim(p.Foreground), Margin: &two}, Theme: r.theme.CodeStyle}
	c.DefinitionDescription = gansi.StylePrimitive{BlockPrefix: "\n🠶 "}
	return c
}

func (r *Renderer) rule(cw int) string {
	return r.gradient(strings.Repeat("━", cw), false)
}

// gradient colors each rune along a blend from Accent to Accent2.
func (r *Renderer) gradient(s string, bold bool) string {
	a, _ := colorful.Hex(r.theme.Palette.Accent)
	b, _ := colorful.Hex(r.theme.Palette.Accent2)
	runes := []rune(s)
	var sb strings.Builder
	for i, ch := range runes {
		t := 0.0
		if len(runes) > 1 {
			t = float64(i) / float64(len(runes)-1)
		}
		st := lipgloss.NewStyle().Foreground(lipgloss.Color(a.BlendLuv(b, t).Clamped().Hex())).Bold(bold)
		sb.WriteString(st.Render(string(ch)))
	}
	return sb.String()
}

func (r *Renderer) cover(cw int) []string {
	m := r.deck.Meta
	center := func(s string) string {
		return strings.Repeat(" ", max((cw-ansi.StringWidth(s))/2, 0)) + s
	}
	title := spaced(strings.ToUpper(m.Title))
	if ansi.StringWidth(title) > cw {
		title = m.Title
	}
	lines := []string{center(r.gradient(title, true)), "", center(r.gradient(strings.Repeat("━", min(cw, ansi.StringWidth(title)+8)), false))}
	if m.SubTitle != "" {
		lines = append(lines, "", center(r.style(r.theme.Palette.Foreground).Italic(true).Render(m.SubTitle)))
	}
	var by []string
	for _, s := range []string{m.Author, m.Date} {
		if s != "" {
			by = append(by, s)
		}
	}
	if len(by) > 0 {
		lines = append(lines, "", "", center(r.style(r.theme.Palette.Dim).Render(strings.Join(by, "  ·  "))))
	}
	return lines
}

func spaced(s string) string {
	return strings.Join(strings.Split(s, ""), " ")
}

func (r *Renderer) footer(idx, w int) []string {
	n := len(r.deck.Slides)
	done := w * (idx + 1) / n
	bar := r.style(r.theme.Palette.Accent).Render(strings.Repeat("━", done)) +
		r.style(r.theme.Palette.Dim).Render(strings.Repeat("─", w-done))
	leftText := " " + r.deck.Meta.Title
	if a := r.deck.Meta.Author; a != "" {
		leftText += "  ·  " + a
	}
	right := fmt.Sprintf("%d / %d ", idx+1, n)
	gap := max(w-ansi.StringWidth(leftText)-ansi.StringWidth(right), 1)
	text := r.style(r.theme.Palette.Dim).Render(leftText + strings.Repeat(" ", gap) + right)
	return []string{bar, text}
}

func (r *Renderer) style(hex string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(hex))
}

// StatusLine renders text as the bottom line of a w-wide frame, in the theme's
// accent and painted like the frame when the theme paints its background.
func (r *Renderer) StatusLine(text string, w int) string {
	l := fit(r.style(r.theme.Palette.Accent).Render(" "+text), w)
	if r.theme.Paint {
		l = paint(l, r.theme.Palette.Foreground, r.theme.Palette.Background)
	}
	return l
}

// Theme is the resolved theme the deck renders in.
func (r *Renderer) Theme() Theme { return r.theme }

// Paints reports whether frames paint the theme background into every cell.
func (r *Renderer) Paints() bool { return r.theme.Paint }

// ThemeWarnings are recoverable theme problems, such as an unknown name
// (which falls back to dark), for the status line or stderr.
func (r *Renderer) ThemeWarnings() []string { return r.themeWarn }

// ThemeErr is set when a custom theme is invalid; the deck then renders in dark.
func (r *Renderer) ThemeErr() error { return r.themeErr }

func (r *Renderer) Accent() color.Color { return lipgloss.Color(r.theme.Palette.Accent) }

// Background is the theme's slide background, the fallback when the terminal's is unknown.
func (r *Renderer) Background() color.Color { return lipgloss.Color(r.theme.Palette.Background) }

// Foreground stands in for text drawn in the terminal's default color.
func (r *Renderer) Foreground() color.Color { return lipgloss.Color(r.theme.Palette.Foreground) }

// fit truncates or pads s to exactly w cells.
func fit(s string, w int) string {
	if ansi.StringWidth(s) > w {
		s = dropTrailingSGR(ansi.Truncate(s, w, ""))
	}
	if pad := w - ansi.StringWidth(s); pad > 0 {
		s += strings.Repeat(" ", pad)
	}
	return s
}

// dropTrailingSGR removes the styling sequences ansi.Truncate keeps from the
// cut-off text (glamour pads every line with individually styled spaces, so
// there can be dozens), ending the line with a single reset instead. Other
// sequences, such as hyperlink ends, are kept.
func dropTrailingSGR(s string) string {
	last := 0
	p := ansi.NewParser()
	var state byte
	for rest := s; len(rest) > 0; {
		_, width, n, ns := ansi.DecodeSequence(rest, state, p)
		state, rest = ns, rest[n:]
		if width > 0 {
			last = len(s) - len(rest)
		}
	}
	var tail strings.Builder
	state = 0
	for rest := s[last:]; len(rest) > 0; {
		seq, _, n, ns := ansi.DecodeSequence(rest, state, p)
		state, rest = ns, rest[n:]
		if !ansi.HasCsiPrefix(seq) || ansi.Cmd(p.Command()) != 'm' {
			tail.WriteString(seq)
		}
	}
	return s[:last] + tail.String() + "\x1b[m"
}

// Fit is exported for transitions.
func Fit(s string, w int) string { return fit(s, w) }

// Markdown renders src in the deck's theme wrapped to width, for the notes view.
func (r *Renderer) Markdown(src string, width int) []string { return r.markdown(src, width) }

// ImageConfig selects how image blocks are drawn.
type ImageConfig struct {
	Mode    img.Mode         // img.Kitty, img.Sixel, img.ITerm2, img.Off; anything else draws half-blocks
	Kitty   *img.Transmitter // required for img.Kitty
	Overlay *img.Overlay     // required for img.Sixel and img.ITerm2
}

// Images is the process-wide image configuration. The zero value draws
// truecolor half-blocks, which is what -dump and -check use.
var Images ImageConfig

const minImageRows = 3

type loadedImage struct {
	src    image.Image
	key    string // changes when the file does
	err    error
	blocks map[[2]int][]string
}

func (r *Renderer) loadImage(path string) *loadedImage {
	if !filepath.IsAbs(path) && r.deck.Dir != "" {
		path = filepath.Join(r.deck.Dir, path)
	}
	if li, ok := r.images[path]; ok {
		return li
	}
	li := &loadedImage{blocks: map[[2]int][]string{}}
	li.src, li.err = img.Load(path)
	if fi, err := os.Stat(path); err == nil {
		abs, _ := filepath.Abs(path)
		li.key = fmt.Sprintf("%s@%d", abs, fi.ModTime().UnixNano())
	}
	r.images[path] = li
	return li
}

// imageStack counts images that stack vertically: top-level images, plus the
// most images in any one column of each column layout.
func imageStack(blocks []deck.Block) int {
	n := 0
	for _, b := range blocks {
		switch b.Kind {
		case deck.Image:
			n++
		case deck.Columns:
			most := 0
			for _, c := range b.Children {
				most = max(most, imageStack(c))
			}
			n += most
		}
	}
	return n
}

// sizeImages shares the rows the fully revealed slide leaves free among its
// images, so every reveal step lays out identically.
func (r *Renderer) sizeImages(s deck.Slide, cw, rows int) {
	r.imgRows = 0
	if s.Cover {
		return
	}
	n := imageStack(s.Blocks)
	if n == 0 {
		return
	}
	used := len(r.bodyFull(s, cw))
	r.imgRows = max((rows-used)/n, minImageRows)
}

var imageWidthRe = regexp.MustCompile(`^image:width:(\d+)%$`)

// image draws an image block centered in width, at most r.imgRows tall.
// While measuring (imgRows 0) a decodable image takes no rows.
func (r *Renderer) image(b deck.Block, width int) []string {
	alt, maxCols := b.Text, width
	if m := imageWidthRe.FindStringSubmatch(alt); m != nil {
		pct, _ := strconv.Atoi(m[1])
		maxCols, alt = max(width*min(pct, 100)/100, 1), ""
	}
	if alt == "" {
		alt = filepath.Base(b.Path)
	}
	li := r.loadImage(b.Path)
	if li.err != nil || Images.Mode == img.Off {
		return r.missingImage(alt, width)
	}
	if r.imgRows == 0 {
		return nil
	}
	bd := li.src.Bounds()
	cols, rows := img.Fit(bd.Dx(), bd.Dy(), maxCols, r.imgRows)
	if cols == 0 {
		return nil
	}
	var lines []string
	if Images.Mode == img.Kitty && Images.Kitty != nil {
		lines = Images.Kitty.Place(li.key, li.src, cols, rows)
	} else if (Images.Mode == img.Sixel || Images.Mode == img.ITerm2) && Images.Overlay != nil {
		// Stand-in cells; the presenter paints the pixels over them.
		lines = Images.Overlay.Place(li.key, li.src, cols, rows)
	} else {
		var ok bool
		if lines, ok = li.blocks[[2]int{cols, rows}]; !ok {
			lines = img.HalfBlocks(li.src, cols, rows)
			li.blocks[[2]int{cols, rows}] = lines
		}
	}
	pad := strings.Repeat(" ", (width-cols)/2)
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = pad + l
	}
	return out
}

// missingImage is a dim three-row box standing in for an image that can't be shown.
func (r *Renderer) missingImage(alt string, width int) []string {
	st := r.style(r.theme.Palette.Dim)
	label := ansi.Truncate("[image: "+alt+"]", max(width-4, 1), "…")
	inner := ansi.StringWidth(label) + 2
	if width < inner+2 {
		return []string{st.Render(ansi.Truncate(label, width, ""))}
	}
	pad := strings.Repeat(" ", (width-inner-2)/2)
	return []string{
		pad + st.Render("╭"+strings.Repeat("─", inner)+"╮"),
		pad + st.Render("│ "+label+" │"),
		pad + st.Render("╰"+strings.Repeat("─", inner)+"╯"),
	}
}
