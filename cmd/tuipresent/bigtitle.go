package main

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/solarisjon/tuipresent/internal/render"
)

// bigTitleDelay is how long a slide must sit still before its scaled title
// is drawn: the renderer paints the block letters first, and the overlay has
// to land after that paint, not before it.
const bigTitleDelay = 120 * time.Millisecond

type bigTitleMsg struct{ gen int }

// bigTitleState tracks the OSC 66 overlay on the terminal. bubbletea's cell
// buffer drops OSC 66, so the scaled title cannot travel in View: the frame
// carries block letters, and the overlay is written over them with tea.Raw.
type bigTitleState struct {
	shown, erase string // overlay on the terminal, and the sequence that blanks it
	pending      string // overlay waiting for its delay
	gen          int
}

// withBigTitle keeps the scaled title in step with the slide: drawn once the
// slide has settled, blanked as soon as something moves. It does nothing
// unless the terminal supports text sizing (render.TextSizing).
func (m *model) withBigTitle(msg tea.Msg, cmd tea.Cmd) tea.Cmd {
	if !render.TextSizing {
		return cmd
	}
	st := &m.big
	var want, wantErase string
	// A pause reveal fades only the body, so the title stays put.
	if m.w > 0 && !m.overview && (m.anim == nil || m.anim.fade) {
		if bt, ok := m.r.BigTitleAt(m.slide, m.step, m.w, m.h); ok {
			want, wantErase = bt.Draw, bt.Erase
		}
	}
	var cmds []tea.Cmd
	if st.shown != "" && want != st.shown {
		cmds = append(cmds, tea.Raw(st.erase))
		st.shown, st.erase = "", ""
	}
	if tm, ok := msg.(bigTitleMsg); ok && tm.gen == st.gen && want != "" && want == st.pending && want != st.shown {
		cmds = append(cmds, tea.Raw(want))
		st.shown, st.erase, st.pending = want, wantErase, ""
	} else if want != "" && want != st.shown && want != st.pending {
		st.gen++
		st.pending = want
		gen := st.gen
		cmds = append(cmds, tea.Tick(bigTitleDelay, func(time.Time) tea.Msg { return bigTitleMsg{gen} }))
	} else if want == "" {
		st.pending = ""
	}
	if len(cmds) == 0 {
		return cmd
	}
	return tea.Batch(append(cmds, cmd)...)
}
