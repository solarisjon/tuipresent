// Package render turns a slide at a reveal step into a fixed-size frame.
package render

import (
	"fmt"
	"image/color"
	"regexp"
	"strings"

	"charm.land/glamour/v2"
	gansi "charm.land/glamour/v2/ansi"
	"charm.land/glamour/v2/styles"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/lucasb-eyer/go-colorful"

	"tuipresent/internal/deck"
)

type Theme struct {
	Glamour string
	Accent  string
	Accent2 string
	Dim     string
	Fg      string
	Bg      string // slide background that pause reveals fade up from
}

func themeFor(name string) Theme {
	n := strings.ToLower(name)
	switch {
	case strings.Contains(n, "light") || strings.Contains(n, "day"):
		return Theme{"light", "#2e7de9", "#9854f1", "#8990b3", "#3760bf", "#e1e2e7"}
	case strings.Contains(n, "tokyo"):
		return Theme{"tokyo-night", "#7aa2f7", "#bb9af7", "#565f89", "#c0caf5", "#24283b"}
	case strings.Contains(n, "dracula"):
		return Theme{"dracula", "#bd93f9", "#ff79c6", "#6272a4", "#f8f8f2", "#282a36"}
	default:
		return Theme{"dark", "#7aa2f7", "#bb9af7", "#5c6370", "#d0d0d0", "#1a1b26"}
	}
}

const (
	maxContentWidth = 110
	columnGap       = 4
)

type Renderer struct {
	deck       *deck.Deck
	theme      Theme
	glam       map[int]*glamour.TermRenderer
	cache      map[[4]int]string
	execOutput map[int]string
	clip       int // widest column overflow seen since Overflow reset it
}

func New(d *deck.Deck) *Renderer {
	return &Renderer{
		deck:       d,
		theme:      themeFor(d.Meta.Theme.Name),
		glam:       map[int]*glamour.TermRenderer{},
		cache:      map[[4]int]string{},
		execOutput: map[int]string{},
	}
}

func (r *Renderer) SetExecOutput(id int, out string) {
	r.execOutput[id] = out
	clear(r.cache)
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

	var head []string
	if s.Title != "" {
		rule := cw
		if s.Centered {
			rule = min(ansi.StringWidth(s.Title)+4, cw)
		}
		head = append(head, "", r.title(s.Title, cw), r.rule(rule), "")
	}

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
			head[i] = padLeft(l, (cw-visibleWidth(l))/2)
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

func contentWidth(w int) int {
	if cw := min(w-8, maxContentWidth); cw >= 20 {
		return cw
	}
	return max(w-2, 1)
}

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
	if s.Title != "" {
		used += 4
	}
	r.clip = 0
	body := r.bodyFull(s, cw)
	used += len(body)
	cols = max(ansi.StringWidth(s.Title)-cw, r.clip)
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

func fencedCode(md string) string {
	var out []string
	fence := ""
	for _, l := range strings.Split(md, "\n") {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~") {
			if fence == "" {
				fence = t[:3]
			} else if strings.HasPrefix(t, fence) {
				fence = ""
			}
			continue
		}
		if fence != "" {
			out = append(out, l)
		}
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
		case deck.Exec:
			md = append(md, "", "```"+b.Lang, b.Text, "```", "")
			if o, ok := r.execOutput[b.ExecID]; ok {
				md = append(md, "```text", strings.TrimRight(o, "\n"), "```", "")
			}
		case deck.Columns:
			flushMD()
			out = append(out, r.columns(b, step, width)...)
			afterColumns = true
		}
	}
	flushMD()
	return out
}

func columnWidths(b deck.Block, width int) []int {
	total := 0
	for _, wt := range b.Widths {
		total += wt
	}
	avail := width - columnGap*(len(b.Widths)-1)
	out := make([]int, len(b.Widths))
	for i, wt := range b.Widths {
		out[i] = max(avail*wt/total, 1)
	}
	return out
}

func (r *Renderer) columns(b deck.Block, step, width int) []string {
	var cols []string
	for i, cw := range columnWidths(b, width) {
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
		lines := r.glamour(seg.text, width)
		if !seg.code {
			lines = hangIndent(lines, max(width-2, 10))
		}
		if len(lines) == 0 {
			continue
		}
		if len(out) > 0 {
			out = append(out, "")
		}
		out = append(out, lines...)
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
		g, err = glamour.NewTermRenderer(glamour.WithStyles(r.glamourStyle()), glamour.WithWordWrap(width))
		if err != nil {
			return strings.Split(src, "\n")
		}
		r.glam[width] = g
	}
	// Raw tabs pass through glamour and misalign frames; CommonMark tab stops are 4.
	s, err := g.Render(strings.ReplaceAll(src, "\t", "    "))
	if err != nil {
		return strings.Split(src, "\n")
	}
	lines := strings.Split(s, "\n")
	for len(lines) > 0 && strings.TrimSpace(ansi.Strip(lines[0])) == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && strings.TrimSpace(ansi.Strip(lines[len(lines)-1])) == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
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

// glamourStyle drops document margins (we lay out ourselves) and restyles headings.
func (r *Renderer) glamourStyle() gansi.StyleConfig {
	base, ok := styles.DefaultStyles[r.theme.Glamour]
	if !ok {
		base = styles.DefaultStyles[styles.DarkStyle]
	}
	c := *base
	zero := uint(0)
	c.Document.Margin = &zero
	c.Document.BlockPrefix, c.Document.BlockSuffix = "", ""
	accent, accent2, t := r.theme.Accent, r.theme.Accent2, true
	head := func(prefix, col string) gansi.StyleBlock {
		return gansi.StyleBlock{StylePrimitive: gansi.StylePrimitive{Prefix: prefix, Color: &col, Bold: &t}}
	}
	c.H1 = head("", accent)
	c.H2 = head("▌ ", accent2)
	c.H3 = head("", accent2)
	c.H4 = head("", accent2)
	c.H5 = head("", accent2)
	c.H6 = head("", accent2)
	return c
}

func (r *Renderer) title(t string, cw int) string {
	return r.gradient(t, true)
}

func (r *Renderer) rule(cw int) string {
	return r.gradient(strings.Repeat("━", cw), false)
}

// gradient colors each rune along a blend from Accent to Accent2.
func (r *Renderer) gradient(s string, bold bool) string {
	a, _ := colorful.Hex(r.theme.Accent)
	b, _ := colorful.Hex(r.theme.Accent2)
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
		lines = append(lines, "", center(r.style(r.theme.Fg).Italic(true).Render(m.SubTitle)))
	}
	var by []string
	for _, s := range []string{m.Author, m.Date} {
		if s != "" {
			by = append(by, s)
		}
	}
	if len(by) > 0 {
		lines = append(lines, "", "", center(r.style(r.theme.Dim).Render(strings.Join(by, "  ·  "))))
	}
	return lines
}

func spaced(s string) string {
	return strings.Join(strings.Split(s, ""), " ")
}

func (r *Renderer) footer(idx, w int) []string {
	n := len(r.deck.Slides)
	done := w * (idx + 1) / n
	bar := r.style(r.theme.Accent).Render(strings.Repeat("━", done)) +
		r.style(r.theme.Dim).Render(strings.Repeat("─", w-done))
	leftText := " " + r.deck.Meta.Title
	if a := r.deck.Meta.Author; a != "" {
		leftText += "  ·  " + a
	}
	right := fmt.Sprintf("%d / %d ", idx+1, n)
	gap := max(w-ansi.StringWidth(leftText)-ansi.StringWidth(right), 1)
	text := r.style(r.theme.Dim).Render(leftText + strings.Repeat(" ", gap) + right)
	return []string{bar, text}
}

func (r *Renderer) style(hex string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(hex))
}

func (r *Renderer) Accent() color.Color { return lipgloss.Color(r.theme.Accent) }

// Background is the theme's slide background, the fallback when the terminal's is unknown.
func (r *Renderer) Background() color.Color { return lipgloss.Color(r.theme.Bg) }

// Foreground stands in for text drawn in the terminal's default color.
func (r *Renderer) Foreground() color.Color { return lipgloss.Color(r.theme.Fg) }

// fit truncates or pads s to exactly w cells.
func fit(s string, w int) string {
	s = ansi.Truncate(s, w, "")
	if pad := w - ansi.StringWidth(s); pad > 0 {
		s += strings.Repeat(" ", pad)
	}
	return s
}

// Fit is exported for transitions.
func Fit(s string, w int) string { return fit(s, w) }
