//go:build ignore

// gen_logo draws examples/assets/terminal.png: a stylised terminal window
// with a gradient body, title-bar buttons and a prompt. Regenerate with
//
//	go run examples/assets/gen_logo.go
package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

const w, h = 320, 160

func mix(a, b color.NRGBA, t float64) color.NRGBA {
	l := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t) }
	return color.NRGBA{l(a.R, b.R), l(a.G, b.G), l(a.B, b.B), 255}
}

// inRounded reports whether (x, y) lies in the rectangle r with corner radius rad.
func inRounded(x, y float64, r image.Rectangle, rad float64) bool {
	x0, y0, x1, y1 := float64(r.Min.X), float64(r.Min.Y), float64(r.Max.X), float64(r.Max.Y)
	if x < x0 || x >= x1 || y < y0 || y >= y1 {
		return false
	}
	cx := math.Max(x0+rad, math.Min(x, x1-rad))
	cy := math.Max(y0+rad, math.Min(y, y1-rad))
	return math.Hypot(x-cx, y-cy) <= rad
}

func main() {
	m := image.NewNRGBA(image.Rect(0, 0, w, h))
	blue := color.NRGBA{0x7a, 0xa2, 0xf7, 255}
	purple := color.NRGBA{0xbb, 0x9a, 0xf7, 255}
	bar := color.NRGBA{0x2a, 0x2e, 0x42, 255}
	screen := color.NRGBA{0x1a, 0x1b, 0x26, 255}
	fg := color.NRGBA{0xc0, 0xca, 0xf5, 255}
	frame := image.Rect(4, 4, w-4, h-4)
	inner := image.Rect(12, 36, w-12, h-12)
	for y := range h {
		for x := range w {
			fx, fy := float64(x)+0.5, float64(y)+0.5
			switch {
			case inRounded(fx, fy, inner, 6):
				m.SetNRGBA(x, y, screen)
			case fy < 32 && inRounded(fx, fy, frame, 14):
				m.SetNRGBA(x, y, bar)
			case inRounded(fx, fy, frame, 14):
				m.SetNRGBA(x, y, mix(blue, purple, (fx+fy)/float64(w+h)))
			}
		}
	}
	dot := func(cx, cy, r float64, c color.NRGBA) {
		for y := range h {
			for x := range w {
				if math.Hypot(float64(x)+0.5-cx, float64(y)+0.5-cy) <= r {
					m.SetNRGBA(x, y, c)
				}
			}
		}
	}
	dot(24, 18, 6, color.NRGBA{0xf7, 0x76, 0x8e, 255})
	dot(44, 18, 6, color.NRGBA{0xe0, 0xaf, 0x68, 255})
	dot(64, 18, 6, color.NRGBA{0x9e, 0xce, 0x6a, 255})
	// Prompt chevron ">" drawn as a thick polyline, then an underscore cursor.
	for t := 0.0; t <= 1; t += 0.002 {
		for _, seg := range [][4]float64{{40, 64, 76, 94}, {76, 94, 40, 124}} {
			x := seg[0] + (seg[2]-seg[0])*t
			y := seg[1] + (seg[3]-seg[1])*t
			dot(x, y, 6, mix(blue, purple, t))
		}
	}
	for y := 118; y < 128; y++ {
		for x := 96; x < 140; x++ {
			m.SetNRGBA(x, y, fg)
		}
	}
	f, err := os.Create("examples/assets/terminal.png")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := png.Encode(f, m); err != nil {
		panic(err)
	}
}
