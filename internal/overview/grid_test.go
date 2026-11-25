package overview

import "testing"

func TestNewGridColumns(t *testing.T) {
	for _, tc := range []struct {
		n, w, h        int
		cols, cardH, r int
	}{
		{20, 100, 30, 4, CardH, 5},
		{20, 80, 24, 3, CardH, 4},
		{2, 100, 30, 2, CardH, 5},
		{20, 100, 5, 3, 1, 3},
		{20, 10, 30, 1, CardH, 5},
	} {
		g := NewGrid(tc.n, tc.w, tc.h)
		if g.Cols != tc.cols || g.CardH != tc.cardH || g.Rows != tc.r {
			t.Errorf("NewGrid(%d,%d,%d) = cols %d cardH %d rows %d, want %d %d %d", tc.n, tc.w, tc.h, g.Cols, g.CardH, g.Rows, tc.cols, tc.cardH, tc.r)
		}
		if g.Left+g.Cols*g.CardW > tc.w && tc.w >= CardW {
			t.Errorf("grid wider than screen: %+v", g)
		}
	}
}

func TestMove(t *testing.T) {
	g := NewGrid(10, 100, 30) // 4 columns: rows of 4, 4, 2
	for _, tc := range []struct{ sel, dx, dy, want int }{
		{0, 1, 0, 1},
		{0, -1, 0, 0},
		{3, 1, 0, 4}, // continues into the next row, like reading order
		{9, 1, 0, 9},
		{1, 0, 1, 5},
		{5, 0, -1, 1},
		{1, 0, -1, 1},
		{7, 0, 1, 9}, // column 3 is missing on the last row: land on the last card
		{9, 0, 1, 9}, // already on the last row
		{8, 0, 1, 8},
	} {
		if got := g.Move(tc.sel, tc.dx, tc.dy); got != tc.want {
			t.Errorf("Move(%d,%d,%d) = %d, want %d", tc.sel, tc.dx, tc.dy, got, tc.want)
		}
	}
}

func TestPagingAndHit(t *testing.T) {
	g := NewGrid(60, 100, 17) // 4 columns, 3 grid rows per page
	if g.Rows != 3 {
		t.Fatalf("rows = %d", g.Rows)
	}
	if lo, hi := g.Visible(0); lo != 0 || hi != 12 {
		t.Errorf("page 1 = [%d,%d)", lo, hi)
	}
	if lo, hi := g.Visible(13); lo != 12 || hi != 24 {
		t.Errorf("page 2 = [%d,%d)", lo, hi)
	}
	if lo, hi := g.Visible(59); lo != 48 || hi != 60 {
		t.Errorf("last page = [%d,%d)", lo, hi)
	}
	x := g.Left + g.CardW + 1
	y := g.Start + g.CardH + 2
	if i, ok := g.At(0, x, y); !ok || i != 5 {
		t.Errorf("At page 1 = %d,%v want 5", i, ok)
	}
	if i, ok := g.At(13, x, y); !ok || i != 17 {
		t.Errorf("At page 2 = %d,%v want 17", i, ok)
	}
	for _, p := range [][2]int{{0, 0}, {x, g.Start - 1}, {g.Left - 1, y}, {100, y}, {x, 16}} {
		if _, ok := g.At(0, p[0], p[1]); ok {
			t.Errorf("At(%v) hit outside the grid", p)
		}
	}
	short := NewGrid(5, 100, 30) // last row has one card
	if _, ok := short.At(0, short.Left+short.CardW*3+1, short.Start+short.CardH+1); ok {
		t.Error("hit an empty slot past the last card")
	}
}
