package main

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/solarisjon/tuipresent/internal/img"
	"github.com/solarisjon/tuipresent/internal/render"
)

// Sixel and iTerm2 images are pixels painted over cells, not cells, so the
// presenter manages them beside the frame: screen blanks the stand-in cells,
// and here the images are painted once the frame has been flushed and wiped
// before the frame changes. Nothing is painted during transitions and fades,
// or in the overview; the image area is blank then.

// overlayDelay lets the renderer flush the new frame before the images are
// painted, since text written afterwards would erase them.
const overlayDelay = 80 * time.Millisecond

type overlayMsg struct{ gen int }

// withOverlay schedules painting for the frame now shown, and clears the
// screen first if an earlier frame's images may still be on it.
func (m *model) withOverlay(cmd tea.Cmd) tea.Cmd {
	ov := render.Images.Overlay
	if ov == nil {
		return cmd
	}
	key, places := "", []img.Placement(nil)
	if m.w > 0 && m.anim == nil && !m.overview {
		var clean string
		clean, places = ov.Extract(m.r.Frame(m.slide, m.step, m.w, m.h))
		if len(places) > 0 {
			// Any change above the status line may have rewritten cells under an
			// image (even blanks, with erase-to-end-of-line), so it repaints them.
			lines := strings.Split(clean, "\n")
			key = fmt.Sprintf("%dx%d\n%s", m.w, m.h, strings.Join(lines[:len(lines)-1], "\n"))
		}
	}
	if key == m.ovKey {
		return cmd
	}
	m.ovKey, m.ovPlaces = key, places
	cmds := []tea.Cmd{cmd}
	if m.ovDrawn {
		m.ovDrawn = false
		cmds = append(cmds, tea.ClearScreen)
	}
	if key != "" {
		m.ovGen++
		gen := m.ovGen
		cmds = append(cmds, tea.Tick(overlayDelay, func(time.Time) tea.Msg { return overlayMsg{gen} }))
	}
	return tea.Batch(cmds...)
}

// drawOverlay paints the pending images, unless the screen moved on since
// gen was scheduled.
func (m *model) drawOverlay(gen int) tea.Cmd {
	ov := render.Images.Overlay
	if ov == nil || gen != m.ovGen || m.ovKey == "" || m.ovDrawn {
		return nil
	}
	m.ovDrawn = true
	places := m.ovPlaces
	return func() tea.Msg { return tea.Raw(ov.Draw(places))() }
}
