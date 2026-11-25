package main

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/solarisjon/tuipresent/internal/deck"
)

// navModel is a presenter for a deck of n one-line slides on a 100x30 screen.
func navModel(t *testing.T, n int) *model {
	t.Helper()
	var sb strings.Builder
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&sb, "Slide %d\n===\n\nbody\n<!-- end_slide -->\n", i)
	}
	d, err := deck.Parse(sb.String())
	if err != nil {
		t.Fatal(err)
	}
	m := newModel("deck.md", d, deckFlags{})
	m.w, m.h = 100, 30
	return m
}

func press(m *model, keys ...string) {
	for _, k := range keys {
		m.key(k)
	}
}

func TestJumpToSlideByNumber(t *testing.T) {
	for _, tc := range []struct {
		name string
		keys []string
		want int // 0-based slide
		num  string
	}{
		{"one digit", []string{"5", "enter"}, 4, ""},
		{"two digits", []string{"1", "2", "enter"}, 11, ""},
		{"past the end clamps", []string{"9", "9", "9", "enter"}, 19, ""},
		{"lone zero ignored", []string{"0", "enter"}, 1, ""},
		{"leading zero ignored", []string{"0", "0", "7", "enter"}, 6, ""},
		{"esc cancels", []string{"3", "esc"}, 0, ""},
		{"pending is kept", []string{"1", "2"}, 0, "12"},
		{"backspace erases", []string{"1", "2", "backspace", "enter"}, 0, ""},
		{"other key cancels and acts", []string{"4", "l"}, 1, ""},
		{"enter without digits is next", []string{"enter"}, 1, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := navModel(t, 20)
			press(m, tc.keys...)
			if m.slide != tc.want || m.num != tc.num {
				t.Errorf("slide %d num %q, want %d %q", m.slide, m.num, tc.want, tc.num)
			}
		})
	}
}

func TestEscCancelsNumberInsteadOfQuitting(t *testing.T) {
	m := navModel(t, 5)
	m.key("2")
	if cmd := m.key("esc"); cmd != nil {
		t.Error("esc with a pending number quit")
	}
	if cmd := m.key("esc"); cmd == nil {
		t.Error("esc with nothing pending should quit")
	}
}

func TestPendingNumberOnStatusLine(t *testing.T) {
	m := navModel(t, 20)
	m.w, m.h = 0, 0
	m.key("1")
	m.key("2")
	if got := m.statusText(); !strings.Contains(got, "12") || !strings.Contains(got, "of 20") {
		t.Errorf("status = %q", got)
	}
	m.w, m.h = 100, 30
	if v := m.screen(); !strings.Contains(v, "go to slide 12") {
		t.Error("screen does not show the pending number")
	}
	m.key("esc")
	if m.statusText() != "" {
		t.Errorf("status after cancel = %q", m.statusText())
	}
}

func TestNumberCapped(t *testing.T) {
	m := navModel(t, 5)
	press(m, "1", "2", "3", "4", "5", "6", "7")
	if len(m.num) != maxNumberDigits {
		t.Errorf("num = %q", m.num)
	}
}

func TestOverviewKeys(t *testing.T) {
	m := navModel(t, 10) // 4 columns at 100 wide
	m.slide = 2
	press(m, "o")
	if !m.overview || m.sel != 2 {
		t.Fatalf("overview %v sel %d", m.overview, m.sel)
	}
	for _, tc := range []struct {
		key  string
		want int
	}{
		{"right", 3}, {"l", 4}, {"down", 8}, {"j", 8}, {"up", 4}, {"k", 0},
		{"left", 0}, {"h", 0}, {"l", 1}, {"j", 5}, {"j", 9}, {"G", 9}, {"g", 0}, {"end", 9}, {"home", 0},
	} {
		m.key(tc.key)
		if m.sel != tc.want {
			t.Fatalf("after %q sel = %d, want %d", tc.key, m.sel, tc.want)
		}
	}
	if m.slide != 2 {
		t.Errorf("moving the selection changed the slide to %d", m.slide)
	}
}

func TestOverviewOpenAndClose(t *testing.T) {
	m := navModel(t, 10)
	press(m, "o", "l", "l", "enter")
	if m.overview || m.slide != 2 || m.step != 0 {
		t.Errorf("enter: overview %v slide %d", m.overview, m.slide)
	}
	press(m, "o", "j", "esc")
	if m.overview || m.slide != 2 {
		t.Errorf("esc: overview %v slide %d", m.overview, m.slide)
	}
	press(m, "o", "o")
	if m.overview {
		t.Error("o does not toggle the overview closed")
	}
	press(m, "o")
	if cmd := m.key("q"); cmd == nil {
		t.Error("q in the overview should quit")
	}
}

func TestOverviewIgnoresNumbersAndShows(t *testing.T) {
	m := navModel(t, 10)
	press(m, "o", "5")
	if m.num != "" {
		t.Errorf("digit typed in the overview left num %q", m.num)
	}
	if v := m.screen(); !strings.Contains(v, "Overview") {
		t.Error("overview not on screen")
	}
}

func TestMouseClickThirds(t *testing.T) {
	m := navModel(t, 5)
	m.slide = 2
	m.mouseClick(10, 5) // left third
	if m.slide != 1 {
		t.Errorf("left click: slide %d", m.slide)
	}
	m.mouseClick(50, 5) // middle
	if m.slide != 1 {
		t.Errorf("middle click moved to %d", m.slide)
	}
	m.mouseClick(90, 5) // right third
	m.mouseClick(67, 5) // edge of the right third
	if m.slide != 3 {
		t.Errorf("right clicks: slide %d", m.slide)
	}
	m.mouseClick(33, 5) // 33*3 = 99 < 100: still the left third
	if m.slide != 2 {
		t.Errorf("left edge click: slide %d", m.slide)
	}
}

func TestMouseClickAdvancesSteps(t *testing.T) {
	d, err := deck.Parse("a\n<!-- pause -->\nb\n<!-- end_slide -->\nnext\n")
	if err != nil {
		t.Fatal(err)
	}
	m := newModel("deck.md", d, deckFlags{})
	m.w, m.h = 100, 30
	m.mouseClick(95, 0)
	if m.slide != 0 || m.step != 1 {
		t.Errorf("slide %d step %d, want a step first", m.slide, m.step)
	}
	m.mouseClick(95, 0)
	if m.slide != 1 {
		t.Errorf("slide %d", m.slide)
	}
}

func TestMouseWheel(t *testing.T) {
	m := navModel(t, 5)
	m.mouseWheel(false)
	m.mouseWheel(false)
	m.mouseWheel(true)
	if m.slide != 1 {
		t.Errorf("slide %d", m.slide)
	}
}

func TestMouseInOverview(t *testing.T) {
	m := navModel(t, 20) // 4 columns of 25, cards 5 rows high from row 1
	press(m, "o")
	g := m.grid()
	m.mouseClick(g.Left+g.CardW*2+3, g.Start+g.CardH+1) // row 1, column 2: slide 7
	if m.overview || m.slide != 6 {
		t.Errorf("overview %v slide %d, want closed on 6", m.overview, m.slide)
	}
	press(m, "o")
	m.mouseClick(0, 0) // the header selects nothing
	if !m.overview {
		t.Error("click outside the cards closed the overview")
	}
	m.mouseWheel(false)
	if m.sel != 10 {
		t.Errorf("wheel down: sel %d, want one row down (10)", m.sel)
	}
	m.mouseWheel(true)
	if m.sel != 6 || m.slide != 6 {
		t.Errorf("wheel up: sel %d slide %d", m.sel, m.slide)
	}
}

func TestUpdateRoutesMouseMessages(t *testing.T) {
	m := navModel(t, 5)
	m.Update(tea.MouseClickMsg{X: 90, Y: 3, Button: tea.MouseLeft})
	m.Update(tea.MouseClickMsg{X: 90, Y: 3, Button: tea.MouseRight}) // ignored
	m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelLeft}) // ignored
	if m.slide != 2 {
		t.Errorf("slide %d, want 2", m.slide)
	}
}

func TestViewMouseMode(t *testing.T) {
	m := navModel(t, 3)
	if m.View().MouseMode != tea.MouseModeNone {
		t.Error("mouse reporting on without -mouse")
	}
	m.mouse = true
	if m.View().MouseMode != tea.MouseModeCellMotion {
		t.Error("mouse reporting off with -mouse")
	}
}
