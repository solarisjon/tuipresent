package notes

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"tuipresent/internal/deck"
	"tuipresent/internal/render"
)

// Frame renders the notes view: exactly h lines of exactly w cells. running is
// false when no presenter is following deckPath, which shows a waiting screen.
func Frame(d *deck.Deck, r *render.Renderer, deckPath string, s State, running bool, now time.Time, w, h int) string {
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
	footer := []string{rule, " Next: " + accent.Render(next)}

	room := h - len(out) - len(footer)
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
	for len(out) < h-len(footer) {
		add("")
	}
	for _, l := range footer {
		add(l)
	}
	return pad(out, w, h)
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
