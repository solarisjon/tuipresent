package render

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/solarisjon/tuipresent/internal/overview"
)

// Overview returns exactly h lines, each exactly w cells wide: a grid of one
// card per slide with slide sel highlighted. The layout is overview.NewGrid(len(slides), w, h).
func (r *Renderer) Overview(sel, w, h int) string {
	if w < 1 || h < 1 {
		return "" // a terminal reporting no room (a collapsing pane) gets nothing
	}
	n := len(r.deck.Slides)
	g := overview.NewGrid(n, w, h)
	lines := []string{r.style(r.theme.Palette.Heading).Bold(true).Render(fmt.Sprintf(" Overview  %d / %d", sel+1, n))}
	lo, hi := g.Visible(sel)
	for row := lo; row < hi; row += g.Cols {
		var cards [][]string
		for i := row; i < row+g.Cols; i++ {
			cards = append(cards, r.card(i, i == sel, g))
		}
		for y := range g.CardH {
			var sb strings.Builder
			sb.WriteString(strings.Repeat(" ", g.Left))
			for _, c := range cards {
				sb.WriteString(c[y])
			}
			lines = append(lines, sb.String())
		}
	}
	for len(lines) < h-overview.FooterRows {
		lines = append(lines, "")
	}
	lines = append(lines[:min(len(lines), h-overview.FooterRows)], r.style(r.theme.Palette.Dim).Render(" arrows move · enter open · esc close"))
	for i, l := range lines {
		lines[i] = fit(l, w)
		if r.theme.Paint {
			lines[i] = paint(lines[i], r.theme.Palette.Foreground, r.theme.Palette.Background)
		}
	}
	return strings.Join(lines[:min(len(lines), h)], "\n")
}

// slideLabel is what slide i is called in the overview: its title, the deck
// title for the cover, or a dash for a slide without a title.
func (r *Renderer) slideLabel(i int) string {
	s := r.deck.Slides[i]
	switch {
	case s.Title != "":
		return s.Title
	case s.Cover && r.deck.Meta.Title != "":
		return r.deck.Meta.Title
	}
	return "—"
}

// card is slide i's card, g.CardH lines of g.CardW cells (blank past the last slide).
func (r *Renderer) card(i int, selected bool, g overview.Grid) []string {
	blank := strings.Repeat(" ", g.CardW)
	if i >= g.N {
		return repeatLine(blank, g.CardH)
	}
	color := r.theme.Palette.Dim
	if selected {
		color = r.theme.Palette.Accent
	}
	edge := r.style(color)
	if selected {
		edge = edge.Bold(true)
	}
	num, label := fmt.Sprint(i+1), r.slideLabel(i)
	if g.CardH == 1 {
		return []string{fit(edge.Render(" "+num+"  "+ansi.Truncate(label, g.CardW-len(num)-4, "…")), g.CardW)}
	}
	inner := g.CardW - 4
	title := ansi.Wrap(label, inner, "")
	t := strings.Split(title, "\n")
	t = append(t, "", "")
	if len(strings.Split(title, "\n")) > 2 {
		t[1] = ansi.Truncate(t[1]+" "+t[2], inner, "…")
	}
	side := edge.Render("│")
	row := func(s string) string {
		return side + " " + fit(s, inner) + " " + side
	}
	horiz := strings.Repeat("─", g.CardW-2)
	return []string{
		edge.Render("╭" + horiz + "╮"),
		row(edge.Render(num)),
		row(t[0]),
		row(t[1]),
		edge.Render("╰" + horiz + "╯"),
	}
}

func repeatLine(s string, n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = s
	}
	return out
}
