// Package transition composites two equal-size frames at progress t in [0,1].
package transition

import (
	"math"
	"math/rand/v2"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

type Kind string

const (
	None     Kind = "none"
	Slide    Kind = "slide"
	Wipe     Kind = "wipe"
	Dissolve Kind = "dissolve"
	Cover    Kind = "cover"
)

func Parse(s string) Kind {
	switch k := Kind(strings.ToLower(strings.TrimSpace(s))); k {
	case None, Slide, Wipe, Dissolve, Cover:
		return k
	}
	return Slide
}

// EaseOutCubic decelerates into the destination frame.
func EaseOutCubic(t float64) float64 { return 1 - math.Pow(1-t, 3) }

// Composite blends from -> to. Frames must be w cells wide; reverse animates leftward/upward.
func Composite(k Kind, from, to string, w int, t float64, reverse bool) string {
	if t >= 1 || k == None {
		return to
	}
	a := strings.Split(from, "\n")
	b := strings.Split(to, "\n")
	n := min(len(a), len(b))
	out := make([]string, n)
	switch k {
	case Slide:
		off := int(math.Round(t * float64(w)))
		for i := range n {
			if reverse {
				// to enters from the left
				out[i] = ansi.Cut(b[i]+a[i], w-off, 2*w-off)
			} else {
				out[i] = ansi.Cut(a[i]+b[i], off, off+w)
			}
		}
	case Wipe:
		x := int(math.Round(t * float64(w)))
		for i := range n {
			if reverse {
				out[i] = ansi.Cut(a[i], 0, w-x) + ansi.Cut(b[i], w-x, w)
			} else {
				out[i] = ansi.Cut(b[i], 0, x) + ansi.Cut(a[i], x, w)
			}
		}
	case Cover:
		rows := int(math.Round(t * float64(n)))
		for i := range n {
			if reverse {
				if i >= n-rows {
					out[i] = b[i-(n-rows)]
				} else {
					out[i] = a[i]
				}
				continue
			}
			if i < rows {
				out[i] = b[n-rows+i]
			} else {
				out[i] = a[i]
			}
		}
	case Dissolve:
		order := rand.New(rand.NewPCG(uint64(n), uint64(w))).Perm(n)
		show := int(math.Round(t * float64(n)))
		revealed := make([]bool, n)
		for _, idx := range order[:show] {
			revealed[idx] = true
		}
		for i := range n {
			if revealed[i] {
				out[i] = b[i]
			} else {
				out[i] = a[i]
			}
		}
	}
	return strings.Join(out, "\n")
}
