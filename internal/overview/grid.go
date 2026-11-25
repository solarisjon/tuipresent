// Package overview is the geometry and cursor logic of the slide overview
// grid: how many cards fit on a w x h screen, which card is under a screen
// cell, and where an arrow key moves the selection. It draws nothing; the
// render package paints the cards.
package overview

const (
	// CardW is the narrowest a card gets; the grid uses as many columns as fit.
	CardW = 24
	// CardH is a card's height: border, slide number, two title lines, border.
	CardH = 5
	// HeaderRows and FooterRows frame the grid with a title and a key hint.
	HeaderRows = 1
	FooterRows = 1
	// compactW is the column width of the one-line cards used when the screen
	// is too short for a full card.
	compactW = 30
)

// Grid is the layout of n cards on a w x h screen.
type Grid struct {
	N, W, H     int
	Cols        int // cards per row
	CardW       int // width of every card; columns are centered as a block
	CardH       int // CardH, or 1 in compact mode
	Rows        int // grid rows that fit between header and footer, at least 1
	Left, Start int // screen column of the first card, screen row of the first grid row
}

// NewGrid lays n slides out on a w x h screen. Cards are CardW or wider, in as
// many columns as fit. A screen too short for one full card gets one-line
// cards instead.
func NewGrid(n, w, h int) Grid {
	g := Grid{N: n, W: w, H: h, CardH: CardH, Start: HeaderRows}
	avail := h - HeaderRows - FooterRows
	minW := CardW
	if avail < CardH {
		g.CardH, minW = 1, compactW
	}
	g.Cols = max(min(w/minW, n), 1)
	g.CardW = min(w/g.Cols, 2*minW+minW/2)
	g.Rows = max(avail/g.CardH, 1)
	g.Left = max((w-g.Cols*g.CardW)/2, 0)
	return g
}

// Row is the grid row of card i.
func (g Grid) Row(i int) int { return i / g.Cols }

// FirstRow is the first visible grid row while card sel is selected. The grid
// scrolls a page at a time, so the visible rows depend only on sel.
func (g Grid) FirstRow(sel int) int { return g.Row(sel) / g.Rows * g.Rows }

// Visible reports the range [lo, hi) of card indexes on screen while sel is selected.
func (g Grid) Visible(sel int) (lo, hi int) {
	lo = g.FirstRow(sel) * g.Cols
	return lo, min(lo+g.Rows*g.Cols, g.N)
}

// Move returns the selection after moving dx cards along reading order and dy
// rows down, clamped to the grid. Moving down from a column that does not
// exist on the last row lands on the last card.
func (g Grid) Move(sel, dx, dy int) int {
	if g.N == 0 {
		return 0
	}
	sel = min(max(sel+dx, 0), g.N-1)
	if dy == 0 {
		return sel
	}
	next := sel + dy*g.Cols
	switch {
	case next < 0:
		return sel
	case next >= g.N:
		if g.Row(sel) == g.Row(g.N-1) {
			return sel
		}
		return g.N - 1
	}
	return next
}

// At is the card under screen cell (x, y) while sel is selected.
func (g Grid) At(sel, x, y int) (int, bool) {
	x -= g.Left
	y -= g.Start
	if x < 0 || y < 0 || x >= g.Cols*g.CardW || y >= g.Rows*g.CardH {
		return 0, false
	}
	i := (g.FirstRow(sel)+y/g.CardH)*g.Cols + x/g.CardW
	return i, i < g.N
}
