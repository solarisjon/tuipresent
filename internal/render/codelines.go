package render

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/solarisjon/tuipresent/internal/deck"
	"github.com/solarisjon/tuipresent/internal/transition"
)

// dimStrength is how much of a dimmed code line's color remains: its text is
// blended this far up from the code background.
const dimStrength = 0.38

// highlightGroup returns the line selection showing at step for a block that
// starts at b.Step: one group per step, the last one staying.
func highlightGroup(b deck.Block, step int) []bool {
	return b.Highlights[min(max(step-b.Step, 0), len(b.Highlights)-1)]
}

// appendCode adds b's highlighted code to out, set off from what precedes it
// by a blank line.
func (r *Renderer) appendCode(out []string, b deck.Block, step, width int) []string {
	lines := r.codeBlock(b, step, width)
	if len(lines) == 0 {
		return out
	}
	if len(out) > 0 {
		out = append(out, "")
	}
	return append(out, lines...)
}

// codeBlock renders b's code as a panel with the lines its current highlight
// group does not select dimmed.
func (r *Renderer) codeBlock(b deck.Block, step, width int) []string {
	sel := highlightGroup(b, step)
	lines := r.glamour(fenceAround(b.Lang, b.Text), width)
	body := strings.Split(strings.ReplaceAll(b.Text, "\t", "    "), "\n")
	// glamour drops blank lines at the edges of the block; recover the offset.
	lead := 0
	for lead < len(body) && strings.TrimSpace(body[lead]) == "" {
		lead++
	}
	trail := 0
	for trail < len(body)-lead && strings.TrimSpace(body[len(body)-1-trail]) == "" {
		trail++
	}
	// A line count that doesn't match means glamour reflowed the code, so no
	// line can be told from another: show it undimmed rather than dim the wrong ones.
	if len(lines) == len(body)-lead-trail {
		for i := range lines {
			if k := lead + i; k < len(sel) && !sel[k] {
				lines[i] = r.dim(lines[i])
			}
		}
	}
	return r.codePanel(lines, width)
}

// dim fades a line's colors toward the code background.
func (r *Renderer) dim(line string) string {
	blank := strings.Repeat(" ", ansi.StringWidth(line))
	bg := lipgloss.Color(r.theme.Palette.CodeBackground)
	return transition.FadeIn(blank, line, dimStrength, bg, r.Foreground())
}

// fenceAround wraps code in a fence longer than any backtick run inside it.
func fenceAround(lang, code string) string {
	n, run := 3, 0
	for _, c := range code {
		if c == '`' {
			run++
			n = max(n, run+1)
		} else {
			run = 0
		}
	}
	f := strings.Repeat("`", n)
	return f + lang + "\n" + code + "\n" + f
}
