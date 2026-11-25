package main

import (
	"strconv"

	tea "charm.land/bubbletea/v2"

	"tuipresent/internal/overview"
)

// maxNumberDigits bounds a typed slide number; decks never have 100k slides.
const maxNumberDigits = 5

// numberKey handles typing a slide number: digits build it up, enter jumps,
// esc cancels, backspace erases a digit. Any other key cancels the number and
// is left for the caller (handled is false). A leading 0 is ignored.
func (m *model) numberKey(k string) (cmd tea.Cmd, handled bool) {
	if len(k) == 1 && k[0] >= '0' && k[0] <= '9' {
		if (k != "0" || m.num != "") && len(m.num) < maxNumberDigits {
			m.num += k
		}
		return nil, true
	}
	if m.num == "" {
		return nil, false
	}
	num := m.num
	m.num = ""
	switch k {
	case "enter":
		n, _ := strconv.Atoi(num)
		return m.jump(min(max(n, 1), len(m.deck.Slides))-1, 0), true
	case "esc":
		return nil, true
	case "backspace":
		m.num = num[:len(num)-1]
		return nil, true
	}
	return nil, false
}

// statusText is the bottom line shown over the slide: a pending slide number
// while one is being typed, else the status message.
func (m *model) statusText() string {
	if m.num != "" {
		return "go to slide " + m.num + "▏ of " + strconv.Itoa(len(m.deck.Slides)) + "  (enter jumps, esc cancels)"
	}
	return m.status
}

func (m *model) openOverview() {
	m.overview, m.sel, m.anim = true, m.slide, nil
}

func (m *model) grid() overview.Grid {
	return overview.NewGrid(len(m.deck.Slides), m.w, m.h)
}

// overviewKey handles a key while the overview grid is open.
func (m *model) overviewKey(k string) tea.Cmd {
	g := m.grid()
	switch k {
	case "q", "ctrl+c":
		return tea.Quit
	case "esc", "o":
		m.overview = false
	case "left", "h":
		m.sel = g.Move(m.sel, -1, 0)
	case "right", "l":
		m.sel = g.Move(m.sel, 1, 0)
	case "up", "k":
		m.sel = g.Move(m.sel, 0, -1)
	case "down", "j":
		m.sel = g.Move(m.sel, 0, 1)
	case "pgup":
		m.sel = g.Move(m.sel, 0, -g.Rows)
	case "pgdown":
		m.sel = g.Move(m.sel, 0, g.Rows)
	case "g", "home":
		m.sel = 0
	case "G", "end":
		m.sel = len(m.deck.Slides) - 1
	case "enter", "space":
		return m.openSelected()
	}
	return nil
}

// openSelected closes the overview and goes to the selected slide.
func (m *model) openSelected() tea.Cmd {
	m.overview = false
	return m.jump(m.sel, 0)
}

// mouseClick handles a left click at screen cell (x, y): in the overview it
// opens the card under the pointer; on a slide the left third goes back and
// the right third goes forward.
func (m *model) mouseClick(x, y int) tea.Cmd {
	m.num = ""
	if m.overview {
		if i, ok := m.grid().At(m.sel, x, y); ok {
			m.sel = i
			return m.openSelected()
		}
		return nil
	}
	switch {
	case m.w == 0:
		return nil
	case x*3 < m.w:
		return m.back()
	case x*3 >= 2*m.w:
		return m.forward()
	}
	return nil
}

// mouseWheel handles a wheel notch: up is back and down is forward, or one
// grid row up or down in the overview.
func (m *model) mouseWheel(up bool) tea.Cmd {
	m.num = ""
	dir := 1
	if up {
		dir = -1
	}
	switch {
	case m.overview:
		m.sel = m.grid().Move(m.sel, 0, dir)
		return nil
	case up:
		return m.back()
	}
	return m.forward()
}
