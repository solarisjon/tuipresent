package deck

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	directiveRe = regexp.MustCompile(`^\s*<!--\s*([a-z_]+)\s*(?::\s*(.*?))?\s*-->\s*$`)
	setextH1Re  = regexp.MustCompile(`^=+\s*$`)
	fenceRe     = regexp.MustCompile("^\\s*(```+|~~~+)\\s*(.*)$")
	// noteOpenRe matches the first line of a multi-line speaker_note comment.
	noteOpenRe = regexp.MustCompile(`^\s*<!--\s*speaker_note\s*(?::(.*))?$`)
	imageRe    = regexp.MustCompile(`^\s*!\[([^\]]*)\]\(\s*(?:<([^>]+)>|([^)\s]+))(?:\s+"[^"]*")?\s*\)\s*$`)
)

func Load(path string) (*Deck, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	d, err := Parse(string(b))
	if d != nil {
		d.Dir = filepath.Dir(path)
	}
	return d, err
}

func Parse(src string) (*Deck, error) {
	src = strings.ReplaceAll(src, "\r\n", "\n")
	d := &Deck{}
	body, err := splitFrontMatter(src, &d.Meta)
	if err != nil {
		return nil, err
	}
	if d.Meta.Title != "" {
		d.Slides = append(d.Slides, Slide{Cover: true, Steps: 1, Centered: true})
	}
	execID := 0
	for _, chunk := range splitSlides(body) {
		s := parseSlide(chunk, &execID)
		if len(s.Blocks) == 0 && s.Title == "" {
			continue
		}
		d.Slides = append(d.Slides, s)
	}
	if len(d.Slides) == 0 {
		return nil, fmt.Errorf("deck has no slides")
	}
	return d, nil
}

func splitFrontMatter(src string, meta *Meta) (string, error) {
	if !strings.HasPrefix(src, "---\n") {
		return src, nil
	}
	rest := src[4:]
	end := strings.Index(rest, "\n---\n")
	if end < 0 {
		if strings.HasSuffix(rest, "\n---") {
			end = len(rest) - 4
		} else {
			return src, nil
		}
	}
	if err := yaml.Unmarshal([]byte(rest[:end]), meta); err != nil {
		return "", fmt.Errorf("front matter: %w", err)
	}
	if end+5 > len(rest) {
		return "", nil
	}
	return rest[end+5:], nil
}

// splitSlides splits on end_slide directives that are outside code fences.
func splitSlides(body string) [][]string {
	var slides [][]string
	var cur []string
	fence := ""
	lines := strings.Split(body, "\n")
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if fence == "" {
			if _, end, ok := multiLineNote(lines, i); ok {
				cur = append(cur, lines[i:end+1]...)
				i = end
				continue
			}
		}
		if m := fenceRe.FindStringSubmatch(line); m != nil {
			if fence == "" {
				fence = m[1]
			} else if toggleFence(fence, m[1]) == "" && strings.TrimSpace(m[2]) == "" {
				fence = ""
			}
		}
		if fence == "" {
			if m := directiveRe.FindStringSubmatch(line); m != nil && m[1] == "end_slide" {
				slides = append(slides, cur)
				cur = nil
				continue
			}
		}
		cur = append(cur, line)
	}
	return append(slides, cur)
}

func toggleFence(open, marker string) string {
	if open == "" {
		return marker
	}
	if marker[0] == open[0] && len(marker) >= len(open) {
		return ""
	}
	return open
}

type slideParser struct {
	slide   Slide
	step    int
	lines   []string
	colIdx  int // index into slide.Blocks of the active Columns block, -1 if none
	curCol  int
	execID  *int
	content bool // any content seen yet (for title detection)
}

func parseSlide(lines []string, execID *int) Slide {
	p := &slideParser{colIdx: -1, curCol: -1, execID: execID}
	for i := 0; i < len(lines); i++ {
		line := lines[i]

		if m := fenceRe.FindStringSubmatch(line); m != nil {
			i = p.fence(lines, i, m[1], m[2])
			continue
		}

		if note, end, ok := multiLineNote(lines, i); ok {
			if note != "" {
				p.slide.Notes = append(p.slide.Notes, note)
			}
			i = end
			continue
		}

		if m := directiveRe.FindStringSubmatch(line); m != nil {
			p.directive(m[1], m[2])
			continue
		}

		if m := imageRe.FindStringSubmatch(line); m != nil {
			p.content = true
			p.flush()
			p.add(Block{Kind: Image, Step: p.step, Text: m[1], Path: m[2] + m[3]})
			continue
		}

		if !p.content && p.slide.Title == "" && strings.TrimSpace(line) != "" &&
			i+1 < len(lines) && setextH1Re.MatchString(lines[i+1]) {
			p.slide.Title = strings.TrimSpace(line)
			i++
			continue
		}

		if strings.TrimSpace(line) != "" {
			p.content = true
		}
		p.lines = append(p.lines, line)
	}
	p.flush()
	p.slide.Steps = p.step + 1
	return p.slide
}

// fence consumes a fenced code block starting at lines[i] and returns the index of its last line.
func (p *slideParser) fence(lines []string, i int, marker, info string) int {
	p.content = true
	fields := strings.Fields(info)
	isExec := false
	for _, f := range fields {
		if f == "+exec" {
			isExec = true
		}
	}
	j := i + 1
	for ; j < len(lines); j++ {
		if m := fenceRe.FindStringSubmatch(lines[j]); m != nil && toggleFence(marker, m[1]) == "" && strings.TrimSpace(m[2]) == "" {
			break
		}
	}
	end := min(j, len(lines)-1)
	if !isExec {
		p.lines = append(p.lines, lines[i:end+1]...)
		return end
	}
	p.flush()
	lang := ""
	if len(fields) > 0 && !strings.HasPrefix(fields[0], "+") {
		lang = fields[0]
	}
	*p.execID++
	p.add(Block{Kind: Exec, Step: p.step, Lang: lang, Text: strings.Join(lines[i+1:min(j, len(lines))], "\n"), ExecID: *p.execID})
	return end
}

func (p *slideParser) directive(name, arg string) {
	switch name {
	case "pause":
		p.flush()
		p.step++
	case "column_layout":
		p.flush()
		widths := parseWidths(arg)
		p.slide.Blocks = append(p.slide.Blocks, Block{Kind: Columns, Step: p.step, Widths: widths, Children: make([][]Block, len(widths))})
		p.colIdx = len(p.slide.Blocks) - 1
		p.curCol = -1
	case "column":
		p.flush()
		n, err := strconv.Atoi(strings.TrimSpace(arg))
		if err == nil && p.colIdx >= 0 {
			p.curCol = min(max(n, 0), len(p.slide.Blocks[p.colIdx].Widths)-1)
		}
	case "reset_layout":
		p.flush()
		p.colIdx, p.curCol = -1, -1
	case "jump_to_middle":
		p.slide.Centered = true
	case "transition":
		p.slide.Transition = strings.TrimSpace(arg)
	case "speaker_note":
		if note := strings.TrimSpace(arg); note != "" {
			p.slide.Notes = append(p.slide.Notes, note)
		}
	case "fade":
		p.slide.Fade = strings.TrimSpace(arg)
	}
}

// multiLineNote reports whether lines[i] opens a speaker_note comment that
// closes on a later line. It returns the note text (dedented and trimmed) and
// the index of the closing line. An unclosed note runs to the end of lines.
func multiLineNote(lines []string, i int) (string, int, bool) {
	if strings.Contains(lines[i], "-->") {
		return "", i, false
	}
	m := noteOpenRe.FindStringSubmatch(lines[i])
	if m == nil {
		return "", i, false
	}
	body := []string{m[1]}
	j := i + 1
	for ; j < len(lines); j++ {
		if k := strings.Index(lines[j], "-->"); k >= 0 {
			body = append(body, lines[j][:k])
			break
		}
		body = append(body, lines[j])
	}
	return strings.TrimSpace(dedent(body)), min(j, len(lines)-1), true
}

// dedent removes the indentation shared by all non-blank lines after the first
// (the first line is the remainder of the opening comment line).
func dedent(lines []string) string {
	indent := -1
	for _, l := range lines[1:] {
		if strings.TrimSpace(l) == "" {
			continue
		}
		n := len(l) - len(strings.TrimLeft(l, " \t"))
		if indent < 0 || n < indent {
			indent = n
		}
	}
	out := []string{strings.TrimSpace(lines[0])}
	for _, l := range lines[1:] {
		if len(l) >= indent && indent > 0 {
			l = l[indent:]
		}
		out = append(out, strings.TrimRight(l, " \t"))
	}
	return strings.Join(out, "\n")
}

func (p *slideParser) flush() {
	// Keep raw lines so consecutive blocks rejoin into the original source.
	text := strings.Join(p.lines, "\n")
	p.lines = nil
	if strings.TrimSpace(text) == "" {
		return
	}
	p.add(Block{Kind: Markdown, Step: p.step, Text: text})
}

func (p *slideParser) add(b Block) {
	if p.colIdx >= 0 && p.curCol >= 0 {
		col := &p.slide.Blocks[p.colIdx]
		col.Children[p.curCol] = append(col.Children[p.curCol], b)
		return
	}
	if p.colIdx >= 0 {
		// Content between column_layout and the first column: goes above the layout.
		p.slide.Blocks = append(p.slide.Blocks[:p.colIdx+1], p.slide.Blocks[p.colIdx:]...)
		p.slide.Blocks[p.colIdx] = b
		p.colIdx++
		return
	}
	p.slide.Blocks = append(p.slide.Blocks, b)
}

func parseWidths(arg string) []int {
	arg = strings.Trim(strings.TrimSpace(arg), "[]")
	var out []int
	for _, f := range strings.Split(arg, ",") {
		if n, err := strconv.Atoi(strings.TrimSpace(f)); err == nil && n > 0 {
			out = append(out, n)
		}
	}
	if len(out) == 0 {
		out = []int{1}
	}
	return out
}
