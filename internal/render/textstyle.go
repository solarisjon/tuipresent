package render

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/lucasb-eyer/go-colorful"

	"github.com/solarisjon/tuipresent/internal/deck"
)

// Curly reports whether the terminal draws curly, dotted, dashed and double
// underlines (SGR 4:n). Where it does not, those fall back to a plain
// underline, and the curly title style to the rule. The zero value is the
// fallback, which is what -dump and -check use; the presenter sets it from
// DetectCurly.
var Curly bool

// DetectCurly reports whether the terminal named by the environment is known
// to draw styled underlines: kitty, Ghostty, WezTerm and iTerm2, outside
// tmux and screen.
func DetectCurly(getenv func(string) string) bool {
	term := getenv("TERM")
	if getenv("TMUX") != "" || strings.HasPrefix(term, "screen") || strings.HasPrefix(term, "tmux") {
		return false
	}
	switch strings.ToLower(getenv("TERM_PROGRAM")) {
	case "ghostty", "kitty", "wezterm", "iterm.app":
		return true
	}
	return strings.Contains(term, "kitty") || strings.Contains(term, "ghostty") || strings.Contains(term, "wezterm") ||
		getenv("KITTY_WINDOW_ID") != "" || getenv("GHOSTTY_RESOURCES_DIR") != ""
}

// run is a stretch of title text sharing one style.
type run struct {
	text                           string
	bold, italic, strike, code, hi bool
}

var inlineDelims = []string{"**", "~~", "==", "`", "*"}

// parseInline splits s into styled runs: **bold**, *italic*, `code`,
// ~~strike~~ and ==highlight==, nested, with backslash escapes. A delimiter
// without a closing partner is literal text.
func parseInline(s string) []run {
	var out []run
	var walk func(s string, st run)
	walk = func(s string, st run) {
		var cur strings.Builder
		flush := func() {
			if cur.Len() > 0 {
				r := st
				r.text = cur.String()
				out = append(out, r)
				cur.Reset()
			}
		}
	next:
		for i := 0; i < len(s); {
			if s[i] == '\\' && i+1 < len(s) && strings.ContainsRune("\\`*~=_[]#>-+!.|", rune(s[i+1])) {
				cur.WriteByte(s[i+1])
				i += 2
				continue
			}
			for _, d := range inlineDelims {
				if !strings.HasPrefix(s[i:], d) {
					continue
				}
				j := strings.Index(s[i+len(d):], d)
				if j <= 0 {
					continue
				}
				flush()
				inner := s[i+len(d) : i+len(d)+j]
				n := st
				switch d {
				case "**":
					n.bold = true
				case "*":
					n.italic = true
				case "~~":
					n.strike = true
				case "==":
					n.hi = true
				case "`":
					n.code = true
					n.text = inner
					out = append(out, n)
					i += 2*len(d) + j
					continue next
				}
				walk(inner, n)
				i += 2*len(d) + j
				continue next
			}
			cur.WriteByte(s[i])
			i++
		}
		flush()
	}
	walk(s, run{})
	return out
}

// plainText is the text of runs without markup.
func plainText(runs []run) string {
	var sb strings.Builder
	for _, r := range runs {
		sb.WriteString(r.text)
	}
	return sb.String()
}

// PlainTitle is a slide title with its inline markdown removed.
func PlainTitle(s string) string { return plainText(parseInline(s)) }

// renderRuns draws runs one rune at a time, colored by col(i) for the i-th
// rune, with attributes layered on top. ul is an SGR sequence for an
// underline, or "".
func (r *Renderer) renderRuns(runs []run, col func(i int) string, bold, italic bool, ul string) string {
	p := r.theme.Palette
	var sb strings.Builder
	i := 0
	for _, ru := range runs {
		for _, ch := range ru.text {
			if bold || ru.bold {
				sb.WriteString("\x1b[1m")
			}
			if italic || ru.italic {
				sb.WriteString("\x1b[3m")
			}
			if ru.strike {
				sb.WriteString("\x1b[9m")
			}
			sb.WriteString(ul)
			switch {
			case ru.code:
				sb.WriteString(sgrColor(38, p.Code) + sgrColor(48, p.CodeBackground))
			case ru.hi:
				sb.WriteString(sgrColor(38, p.Background) + sgrColor(48, p.Highlight))
			default:
				sb.WriteString(sgrColor(38, col(i)))
			}
			sb.WriteRune(ch)
			sb.WriteString("\x1b[m")
			i++
		}
	}
	return sb.String()
}

// gradientAt is the Accent-to-Accent2 blend at rune i of n.
func (r *Renderer) gradientAt(n int) func(int) string {
	a, _ := colorful.Hex(r.theme.Palette.Accent)
	b, _ := colorful.Hex(r.theme.Palette.Accent2)
	return func(i int) string {
		t := 0.0
		if n > 1 {
			t = float64(i) / float64(n-1)
		}
		return a.BlendLuv(b, t).Clamped().Hex()
	}
}

// styledTitle draws a title with inline markdown on the accent gradient.
func (r *Renderer) styledTitle(title string, curly bool) string {
	runs := parseInline(title)
	ul := ""
	if curly {
		ul = sgrColor(58, r.theme.Palette.Accent) + "\x1b[4:3m"
	}
	return r.renderRuns(runs, r.gradientAt(len([]rune(plainText(runs)))), true, false, ul)
}

// styledSubtitle draws the line under a title: foreground, italic.
func (r *Renderer) styledSubtitle(sub string) string {
	fg := r.theme.Palette.Foreground
	return r.renderRuns(parseInline(sub), func(int) string { return fg }, false, true, "")
}

// head is the rows above the body of a titled slide: a blank row, the title
// in its style, an optional subtitle, the rule, and a blank row. The default
// style is 4 rows (5 with a subtitle); frame is 5 (6). It is empty for an
// untitled slide. width is the widest line of the head.
func (r *Renderer) head(idx int, s deck.Slide, cw int) (lines []string, width int) {
	lines, width, _ = r.headBig(idx, s, cw)
	return lines, width
}

// headBig is head plus, for a slide whose title_size applies, where its block
// letters sit. A title that cannot be drawn big (see TitleSizeWarnings) gets
// the normal head. title_size adds TitleRows(size)-1 rows to each style.
func (r *Renderer) headBig(idx int, s deck.Slide, cw int) (lines []string, width int, bh *bigHead) {
	if s.Title == "" {
		return nil, 0, nil
	}
	style := r.deck.TitleStyle(idx)
	if size := r.deck.TitleSize(idx); size > 1 && !s.Cover {
		if l, w, b, reason := r.bigHead(s, size, style, cw); reason == "" {
			return l, w, b
		}
	}
	tw := ansi.StringWidth(PlainTitle(s.Title))
	sw := ansi.StringWidth(PlainTitle(s.Subtitle))
	inner := max(tw, sw)
	var sub string
	if s.Subtitle != "" {
		sub = r.styledSubtitle(s.Subtitle)
	}
	switch style {
	case "frame":
		bw := inner + 4
		border := func(l string) string { return r.gradient(l, false) }
		side := r.style(r.theme.Palette.Accent).Render("│")
		row := func(content string, w int) string {
			return side + " " + content + strings.Repeat(" ", inner-w) + " " + side
		}
		lines = []string{"", border("╭" + strings.Repeat("─", bw-2) + "╮"), row(r.styledTitle(s.Title, false), tw)}
		if sub != "" {
			lines = append(lines, row(sub, sw))
		}
		lines = append(lines, border("╰"+strings.Repeat("─", bw-2)+"╯"), "")
		return lines, bw, nil
	case "curly":
		if Curly {
			lines = []string{"", r.styledTitle(s.Title, true)}
			if sub != "" {
				lines = append(lines, sub)
			}
			return append(lines, "", ""), inner, nil
		}
	}
	rule := cw
	if s.Centered {
		rule = min(inner+4, cw)
	}
	lines = []string{"", r.styledTitle(s.Title, false)}
	if sub != "" {
		lines = append(lines, sub)
	}
	return append(lines, r.rule(rule), ""), inner, nil
}

// ---- inline marks in body text ----

// Body text marks: ==highlight==, ++underline++ and [text]{.class} spans for
// faint and the underline variants. glamour knows none of them, so they are
// swapped for private-use marker runes before it renders and turned into SGR
// afterwards.
var markClasses = []string{"highlight", "faint", "underline", "curly", "dotted", "dashed", "double"}

func markRune(k int, open bool) rune {
	r := rune(0xE000 + 2*k)
	if !open {
		r++
	}
	return r
}

var (
	markHighlightRe = regexp.MustCompile(`==([^\s=](?:[^=\n]*[^\s=])?)==`)
	markInsRe       = regexp.MustCompile(`\+\+([^\s+](?:[^+\n]*[^\s+])?)\+\+`)
	markSpanRe      = regexp.MustCompile(`\[([^\]\n]+)\]\{\.([a-z]+)\}`)
)

func wrapMark(k int, text string) string {
	return string(markRune(k, true)) + text + string(markRune(k, false))
}

func markIndex(class string) int {
	for k, c := range markClasses {
		if c == class {
			return k
		}
	}
	return -1
}

// markText replaces mark syntax in prose with marker runes, leaving inline
// code spans alone. It returns src unchanged if there is none.
func markText(src string) string {
	if !strings.Contains(src, "==") && !strings.Contains(src, "++") && !strings.Contains(src, "]{.") {
		return src
	}
	sub := func(s string) string {
		s = markHighlightRe.ReplaceAllString(s, string(markRune(0, true))+"$1"+string(markRune(0, false)))
		s = markInsRe.ReplaceAllString(s, string(markRune(2, true))+"$1"+string(markRune(2, false)))
		return markSpanRe.ReplaceAllStringFunc(s, func(m string) string {
			g := markSpanRe.FindStringSubmatch(m)
			if k := markIndex(g[2]); k >= 0 {
				return wrapMark(k, g[1])
			}
			return m
		})
	}
	var sb strings.Builder
	for len(src) > 0 {
		i := strings.IndexByte(src, '`')
		if i < 0 {
			sb.WriteString(sub(src))
			break
		}
		n := 0
		for i+n < len(src) && src[i+n] == '`' {
			n++
		}
		sb.WriteString(sub(src[:i]))
		ticks := src[i : i+n]
		end := strings.Index(src[i+n:], ticks)
		if end < 0 {
			sb.WriteString(src[i:])
			break
		}
		sb.WriteString(src[i : i+n+end+n])
		src = src[i+n+end+n:]
	}
	return sb.String()
}

func (r *Renderer) markOpen(k int) string {
	p := r.theme.Palette
	ul := func(style string) string {
		if Curly {
			return sgrColor(58, p.Accent) + "\x1b[4:" + style + "m"
		}
		return "\x1b[4m"
	}
	switch markClasses[k] {
	case "highlight":
		return sgrColor(48, p.Highlight) + sgrColor(38, p.Background)
	case "faint":
		return "\x1b[2m"
	case "underline":
		return "\x1b[4m"
	case "curly":
		return ul("3")
	case "dotted":
		return ul("4")
	case "dashed":
		return ul("5")
	default: // double
		return ul("2")
	}
}

func (r *Renderer) markClose(k int) string {
	switch markClasses[k] {
	case "highlight":
		return "\x1b[49m\x1b[39m"
	case "faint":
		return "\x1b[22m"
	}
	return "\x1b[24m\x1b[59m"
}

// applyMarks turns marker runes in rendered lines into SGR. A mark that
// spans a wrapped line is closed at the end of one line and reopened on the
// next, so no line leaks style.
func (r *Renderer) applyMarks(lines []string) []string {
	var active []int
	out := make([]string, len(lines))
	for i, l := range lines {
		if len(active) == 0 && !strings.ContainsFunc(l, isMark) {
			out[i] = l
			continue
		}
		var sb strings.Builder
		for _, k := range active {
			sb.WriteString(r.markOpen(k))
		}
		for _, ch := range l {
			if !isMark(ch) {
				sb.WriteRune(ch)
				continue
			}
			k := int(ch-0xE000) / 2
			if ch%2 == 0 {
				active = append(active, k)
				sb.WriteString(r.markOpen(k))
				continue
			}
			for j := len(active) - 1; j >= 0; j-- {
				if active[j] == k {
					active = append(active[:j], active[j+1:]...)
					break
				}
			}
			sb.WriteString(r.markClose(k))
		}
		for j := len(active) - 1; j >= 0; j-- {
			sb.WriteString(r.markClose(active[j]))
		}
		out[i] = sb.String()
	}
	return out
}

func isMark(r rune) bool { return r >= 0xE000 && r < 0xE000+2*rune(len(markClasses)) }

// ---- callouts ----

var (
	calloutOpenRe = regexp.MustCompile(`(?i)^ {0,3}>\s*\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]\s*$`)
	quoteLineRe   = regexp.MustCompile(`^ {0,3}> ?`)
)

// piece is a stretch of prose, or a callout when kind is set.
type piece struct {
	text string
	kind string // "note", "tip", "important", "warning", "caution"
}

// splitCallouts pulls GitHub-style callouts (`> [!NOTE]` and the quoted
// lines after it) out of prose.
func splitCallouts(src string) []piece {
	if !strings.Contains(src, "[!") {
		return []piece{{text: src}}
	}
	var out []piece
	var cur []string
	flush := func() {
		if strings.TrimSpace(strings.Join(cur, "")) != "" {
			out = append(out, piece{text: strings.Join(cur, "\n")})
		}
		cur = nil
	}
	lines := strings.Split(src, "\n")
	for i := 0; i < len(lines); i++ {
		m := calloutOpenRe.FindStringSubmatch(lines[i])
		if m == nil {
			cur = append(cur, lines[i])
			continue
		}
		flush()
		var body []string
		for i+1 < len(lines) && quoteLineRe.MatchString(lines[i+1]) {
			i++
			body = append(body, quoteLineRe.ReplaceAllString(lines[i], ""))
		}
		out = append(out, piece{text: strings.Join(body, "\n"), kind: strings.ToLower(m[1])})
	}
	flush()
	return out
}

func (r *Renderer) calloutColor(kind string) string {
	p := r.theme.Palette
	switch kind {
	case "tip":
		return p.Tip
	case "important":
		return p.Important
	case "warning":
		return p.Warning
	case "caution":
		return p.Caution
	}
	return p.Note
}

// callout draws a rounded box in the callout's color with its name in the
// top border and the body inside: two rows more than the body, and up to
// width cells wide.
func (r *Renderer) callout(kind, body string, width int) []string {
	label := strings.ToUpper(kind[:1]) + kind[1:]
	iw := max(width-4, 10)
	inner := r.markdown(body, iw)
	if len(inner) == 0 {
		inner = []string{""}
	}
	cw := 0
	for _, l := range inner {
		cw = max(cw, visibleWidth(l))
	}
	// The label always fits, even in a region too narrow for the box (then
	// the column or frame clips it).
	cw = max(min(cw, iw), ansi.StringWidth(label)+2)
	col := r.style(r.calloutColor(kind))
	bar := col.Render("│")
	top := col.Render("╭─ ") + col.Bold(true).Render(label) + col.Render(" "+strings.Repeat("─", cw-ansi.StringWidth(label)-2)+"─╮")
	out := []string{top}
	for _, l := range inner {
		out = append(out, bar+" "+fit(l, cw)+" "+bar)
	}
	return append(out, col.Render("╰"+strings.Repeat("─", cw+2)+"╯"))
}
