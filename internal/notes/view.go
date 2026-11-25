package notes

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/solarisjon/tuipresent/internal/deck"
	"github.com/solarisjon/tuipresent/internal/render"
)

// Frame renders the notes view: exactly h lines of exactly w cells. running is
// false when no presenter is following deckPath, which shows a waiting screen.
// warnings (see report.Warnings) are listed above the footer, or under the
// waiting message, for as long as they apply.
func Frame(d *deck.Deck, r *render.Renderer, deckPath string, s State, running bool, now time.Time, w, h int, warnings []string) string {
	return FrameStatus(d, r, deckPath, s, running, now, w, h, warnings, "")
}

// FrameStatus is Frame with a status (a typed slide number, or a key hint)
// right-aligned on the "Next:" line when it fits.
func FrameStatus(d *deck.Deck, r *render.Renderer, deckPath string, s State, running bool, now time.Time, w, h int, warnings []string, status string) string {
	if w <= 0 || h <= 0 {
		return ""
	}
	accent := lipgloss.NewStyle().Foreground(r.Accent()).Bold(true)
	dim := lipgloss.NewStyle().Faint(true)
	rule := dim.Render(strings.Repeat("─", w))
	var out []string
	add := func(l string) { out = append(out, render.Fit(l, w)) }

	if !running {
		add(accent.Render(" SPEAKER NOTES"))
		add(rule)
		add("")
		add(" Waiting for the presenter…")
		add("")
		add(dim.Render(" Start it in another pane:  tuipresent " + deckPath))
		if len(warnings) > 0 {
			add("")
			for _, l := range warningLines(warnings, h-len(out), w) {
				add(l)
			}
		}
		return pad(out, w, h)
	}

	slide := min(max(s.Slide, 0), len(d.Slides)-1)
	cur := d.Slides[slide]
	step := min(max(s.Step, 0), cur.Steps-1)
	info := fmt.Sprintf("slide %d/%d · step %d/%d · %s ", slide+1, len(d.Slides), step+1, cur.Steps, elapsed(now.Sub(s.Started)))
	left := " " + slideTitle(d, slide)
	gap := max(w-ansi.StringWidth(left)-ansi.StringWidth(info), 1)
	add(accent.Render(left) + strings.Repeat(" ", gap) + info)
	add(rule)

	next := "end of deck"
	if slide+1 < len(d.Slides) {
		next = slideTitle(d, slide+1)
	}
	nextLine := " Next: " + accent.Render(next)
	if g := w - ansi.StringWidth(nextLine) - ansi.StringWidth(status) - 1; status != "" && g > 0 {
		nextLine += strings.Repeat(" ", g) + dim.Render(status)
	}
	footer := []string{rule, nextLine}

	room := h - len(out) - len(footer)
	// Warnings take at most half of the room, so the notes stay readable.
	warn := warningLines(warnings, room/2, w)
	room -= len(warn)
	var body []string
	if len(cur.Notes) == 0 {
		body = []string{dim.Render(" (no notes for this slide)")}
	} else {
		for _, l := range r.Markdown(strings.Join(cur.Notes, "\n\n"), max(w-2, 10)) {
			body = append(body, " "+l)
		}
	}
	if len(body) > room && room > 0 {
		body = append(body[:room-1], dim.Render(" …"))
	}
	for _, l := range body[:min(len(body), max(room, 0))] {
		add(l)
	}
	for len(out) < h-len(footer)-len(warn) {
		add("")
	}
	for _, l := range warn {
		add(l)
	}
	for _, l := range footer {
		add(l)
	}
	return pad(out, w, h)
}

// warningLines formats warnings as " warning: text" lines, at most limit of
// them; when some don't fit, the last line counts the hidden ones.
func warningLines(warnings []string, limit, w int) []string {
	if len(warnings) == 0 || limit <= 0 {
		return nil
	}
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true)
	shown := warnings
	if len(shown) > limit {
		shown = warnings[:limit-1]
	}
	var out []string
	for _, s := range shown {
		out = append(out, style.Render(" warning: ")+s)
	}
	if hidden := len(warnings) - len(shown); hidden > 0 {
		out = append(out, style.Render(fmt.Sprintf(" … %d more warning(s); run tuipresent -check", hidden)))
	}
	return out
}

func pad(lines []string, w, h int) string {
	for len(lines) < h {
		lines = append(lines, strings.Repeat(" ", w))
	}
	return strings.Join(lines[:h], "\n")
}

func slideTitle(d *deck.Deck, i int) string {
	s := d.Slides[i]
	switch {
	case s.Cover:
		return d.Meta.Title
	case s.Title != "":
		return s.Title
	}
	return fmt.Sprintf("slide %d (untitled)", i+1)
}

func elapsed(d time.Duration) string {
	d = max(d, 0).Truncate(time.Second)
	return fmt.Sprintf("%d:%02d:%02d", int(d.Hours()), int(d.Minutes())%60, int(d.Seconds())%60)
}
