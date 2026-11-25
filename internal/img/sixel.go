package img

import (
	"image"
	"sort"
	"strconv"
	"strings"
)

// MaxSixelColors caps the sixel palette. Smaller palettes encode and draw
// faster; with error diffusion 128 colours look fine for slides.
const MaxSixelColors = 128

type rgb [3]uint8

// EncodeSixel returns src as a complete sixel DCS sequence (ESC P ... ESC \)
// with at most MaxSixelColors colours, error-diffused. Pixels with alpha
// below 128 are left undrawn, so the terminal background shows through.
func EncodeSixel(src image.Image) string {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return ""
	}
	// Flatten to NRGBA once so the hot loops avoid interface calls.
	px := Scale(src, w, h)
	pal := medianCut(px, MaxSixelColors)
	idx := quantize(px, pal)

	var sb strings.Builder
	// P2=1: unset pixels stay transparent. Raster attributes: 1:1 aspect, w x h.
	sb.WriteString("\x1bP0;1;0q\"1;1;" + strconv.Itoa(w) + ";" + strconv.Itoa(h))
	for i, c := range pal {
		sb.WriteString("#" + strconv.Itoa(i) + ";2;" + strconv.Itoa(pct(c[0])) + ";" + strconv.Itoa(pct(c[1])) + ";" + strconv.Itoa(pct(c[2])))
	}
	masks := make([][]byte, len(pal))
	used := make([]bool, len(pal))
	for y0 := 0; y0 < h; y0 += 6 {
		if y0 > 0 {
			sb.WriteByte('-')
		}
		clear(used)
		for k := 0; k < 6 && y0+k < h; k++ {
			for x := range w {
				c := idx[(y0+k)*w+x]
				if c < 0 {
					continue
				}
				if masks[c] == nil {
					masks[c] = make([]byte, w)
				}
				if !used[c] {
					used[c] = true
					clear(masks[c])
				}
				masks[c][x] |= 1 << k
			}
		}
		first := true
		for c := range pal {
			if !used[c] {
				continue
			}
			if !first {
				sb.WriteByte('$')
			}
			first = false
			sb.WriteString("#" + strconv.Itoa(c))
			writeRuns(&sb, masks[c])
		}
	}
	sb.WriteString("\x1b\\")
	return sb.String()
}

func pct(v uint8) int { return (int(v)*100 + 127) / 255 }

// writeRuns writes one colour's sixel characters, run-length encoded.
func writeRuns(sb *strings.Builder, m []byte) {
	// Trailing empty columns need no drawing.
	for len(m) > 0 && m[len(m)-1] == 0 {
		m = m[:len(m)-1]
	}
	for i := 0; i < len(m); {
		j := i
		for j < len(m) && m[j] == m[i] {
			j++
		}
		ch, n := '?'+rune(m[i]), j-i
		if n > 3 {
			sb.WriteString("!" + strconv.Itoa(n))
			sb.WriteRune(ch)
		} else {
			for range n {
				sb.WriteRune(ch)
			}
		}
		i = j
	}
}

// medianCut picks up to n colours for the opaque pixels of px.
func medianCut(px *image.NRGBA, n int) []rgb {
	var all []rgb
	// Sample so huge images don't make palette selection the bottleneck.
	step := max(len(px.Pix)/4/40000, 1)
	for i := 0; i < len(px.Pix); i += 4 * step {
		if px.Pix[i+3] >= 128 {
			all = append(all, rgb{px.Pix[i], px.Pix[i+1], px.Pix[i+2]})
		}
	}
	if len(all) == 0 {
		return []rgb{{0, 0, 0}}
	}
	boxes := [][]rgb{all}
	for len(boxes) < n {
		// Split the box with the widest channel range.
		best, bestCh, bestRange := -1, 0, 0
		for i, bx := range boxes {
			if len(bx) < 2 {
				continue
			}
			for ch := range 3 {
				lo, hi := uint8(255), uint8(0)
				for _, c := range bx {
					lo, hi = min(lo, c[ch]), max(hi, c[ch])
				}
				if r := int(hi - lo); r > bestRange {
					best, bestCh, bestRange = i, ch, r
				}
			}
		}
		if best < 0 {
			break
		}
		bx := boxes[best]
		sort.Slice(bx, func(i, j int) bool { return bx[i][bestCh] < bx[j][bestCh] })
		mid := len(bx) / 2
		boxes[best] = bx[:mid]
		boxes = append(boxes, bx[mid:])
	}
	pal := make([]rgb, len(boxes))
	for i, bx := range boxes {
		var s [3]int
		for _, c := range bx {
			for ch := range 3 {
				s[ch] += int(c[ch])
			}
		}
		for ch := range 3 {
			pal[i][ch] = uint8((s[ch] + len(bx)/2) / len(bx))
		}
	}
	return pal
}

// quantize maps px to palette indexes (-1 for transparent) with
// Floyd-Steinberg error diffusion.
func quantize(px *image.NRGBA, pal []rgb) []int16 {
	w, h := px.Rect.Dx(), px.Rect.Dy()
	out := make([]int16, w*h)
	cache := map[uint16]int16{}
	nearest := func(r, g, b int) int16 {
		key := uint16(r>>3)<<10 | uint16(g>>3)<<5 | uint16(b>>3)
		if v, ok := cache[key]; ok {
			return v
		}
		best, bd := 0, 1<<30
		for i, c := range pal {
			dr, dg, db := r-int(c[0]), g-int(c[1]), b-int(c[2])
			if d := dr*dr + dg*dg + db*db; d < bd {
				best, bd = i, d
			}
		}
		cache[key] = int16(best)
		return int16(best)
	}
	// cur and next hold the diffused error (in 1/16ths) for this and the next
	// row, offset by one so x-1 is addressable.
	cur, next := make([][3]int, w+2), make([][3]int, w+2)
	for y := range h {
		for x := range w {
			p := px.Pix[y*px.Stride+x*4:]
			if p[3] < 128 {
				out[y*w+x] = -1
				continue
			}
			v := [3]int{int(p[0]) + cur[x+1][0]/16, int(p[1]) + cur[x+1][1]/16, int(p[2]) + cur[x+1][2]/16}
			for ch := range v {
				v[ch] = min(max(v[ch], 0), 255)
			}
			i := nearest(v[0], v[1], v[2])
			out[y*w+x] = i
			for ch := range v {
				e := v[ch] - int(pal[i][ch])
				cur[x+2][ch] += e * 7
				next[x][ch] += e * 3
				next[x+1][ch] += e * 5
				next[x+2][ch] += e
			}
		}
		cur, next = next, cur
		clear(next)
	}
	return out
}
