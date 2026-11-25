// Package report produces what -check, -dump and the warning lines print, so
// the command and the docs tests (internal/docscheck) share one implementation.
package report

import (
	"fmt"
	"io"

	"github.com/solarisjon/tuipresent/internal/deck"
	"github.com/solarisjon/tuipresent/internal/render"
	"github.com/solarisjon/tuipresent/internal/transition"
)

// Warnings lists every recoverable problem with a deck as loaded: parser
// warnings (such as an unclosed speaker note or an unknown transition), an
// unknown -transition flag value, and theme warnings. Each one is a single
// line without a prefix; the caller adds "path: warning: " or joins them for
// the status line.
func Warnings(d *deck.Deck, r *render.Renderer, transitionFlag string) []string {
	var out []string
	if transitionFlag != "" {
		if _, ok := transition.Lookup(transitionFlag); !ok {
			out = append(out, "-transition: "+transition.UnknownWarning(transitionFlag))
		}
	}
	out = append(out, d.Warnings...)
	return append(out, r.ThemeWarnings()...)
}

// LayoutWarnings lists the problems that depend on the screen width, which
// -check and -dump know: a title_size that does not fit, so the slide draws
// its normal title.
func LayoutWarnings(r *render.Renderer, w int) []string {
	return r.TitleSizeWarnings(w)
}

// PrintWarnings writes each warning as "name: warning: text".
func PrintWarnings(w io.Writer, name string, warnings []string) {
	for _, s := range warnings {
		fmt.Fprintf(w, "%s: warning: %s\n", name, s)
	}
}

// Check writes one line per slide that overflows a w x h screen, or a single
// "N slides fit" line, and returns the number of overflowing slides.
func Check(out io.Writer, name string, d *deck.Deck, r *render.Renderer, w, h int) int {
	bad := 0
	for i := range d.Slides {
		rows, cols := r.Overflow(i, w, h)
		if rows > 0 || cols > 0 {
			bad++
			fmt.Fprintf(out, "%s: slide %d overflows %dx%d by %d rows, %d cols\n", name, i+1, w, h, rows, cols)
		}
	}
	if bad == 0 {
		fmt.Fprintf(out, "%s: %d slides fit %dx%d\n", name, len(d.Slides), w, h)
	}
	return bad
}

// Dump writes every slide at every reveal step, each frame headed by
// "=== slide N step k/n ===".
func Dump(out io.Writer, d *deck.Deck, r *render.Renderer, w, h int) {
	for i, s := range d.Slides {
		for st := range s.Steps {
			fmt.Fprintf(out, "=== slide %d step %d/%d ===\n%s\n", i+1, st+1, s.Steps, r.Frame(i, st, w, h))
		}
	}
}
