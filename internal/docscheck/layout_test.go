package docscheck

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"tuipresent/internal/deck"
	"tuipresent/internal/render"
)

var (
	sizeRe    = regexp.MustCompile(`^(\d+)x(\d+)$`)
	weightsRe = regexp.MustCompile(`\[(\d+(?:\s*,\s*\d+)*)\]`)
)

// TestLayoutTables checks every table whose columns are screen sizes
// ("| | 100x30 | 80x24 |") in the docs against the renderer. Every row label
// must be one layoutValue understands, so a new row needs a rule here.
func TestLayoutTables(t *testing.T) {
	found := 0
	for _, doc := range docs {
		lines := strings.Split(read(t, doc), "\n")
		for i := 0; i+1 < len(lines); i++ {
			sizes := tableSizes(lines[i])
			if sep := strings.TrimSpace(lines[i+1]); sizes == nil || !strings.HasPrefix(sep, "|") || !strings.Contains(sep, "---") {
				continue
			}
			found++
			for j := i + 2; j < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[j]), "|"); j++ {
				cells := tableCells(lines[j])
				if len(cells) != len(sizes)+1 {
					t.Errorf("%s:%d: want %d cells", doc, j+1, len(sizes)+1)
					continue
				}
				for k, sz := range sizes {
					want, ok := layoutValue(t, cells[0], sz[0], sz[1])
					if !ok {
						t.Errorf("%s:%d: unknown layout row %q (teach layoutValue in internal/docscheck/layout_test.go)", doc, j+1, cells[0])
						break
					}
					if got := norm(cells[k+1]); !contains(want, got) {
						t.Errorf("%s:%d: %q at %dx%d is %q, renderer gives %q", doc, j+1, cells[0], sz[0], sz[1], cells[k+1], want[0])
					}
				}
			}
		}
	}
	if found < 3 {
		t.Errorf("found %d layout tables, want at least 3 (LANGUAGE.md, reference.md, SKILL.md)", found)
	}
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if norm(v) == s {
			return true
		}
	}
	return false
}

func norm(s string) string {
	return strings.NewReplacer(" ", "", "*", "").Replace(s)
}

func tableCells(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimSuffix(strings.TrimPrefix(line, "|"), "|")
	cells := strings.Split(line, "|")
	for i := range cells {
		cells[i] = strings.TrimSpace(cells[i])
	}
	return cells
}

// tableSizes returns the screen sizes of a header row like "| | 100x30 | 80x24 |".
func tableSizes(line string) [][2]int {
	if !strings.HasPrefix(strings.TrimSpace(line), "|") {
		return nil
	}
	cells := tableCells(line)
	if len(cells) < 2 {
		return nil
	}
	var out [][2]int
	for _, c := range cells[1:] {
		m := sizeRe.FindStringSubmatch(c)
		if m == nil {
			return nil
		}
		w, _ := strconv.Atoi(m[1])
		h, _ := strconv.Atoi(m[2])
		out = append(out, [2]int{w, h})
	}
	return out
}

// layoutValue returns the accepted spellings of a layout table cell.
func layoutValue(t *testing.T, label string, w, h int) ([]string, bool) {
	l := strings.ToLower(label)
	cw := render.ContentWidth(w)
	if got := measuredContentWidth(t, w, h); got != cw {
		t.Fatalf("content width at %d: title overflow says %d, ContentWidth says %d", w, got, cw)
	}
	g := max(cw-2, 10)
	k := g - 2
	one := func(n int) []string { return []string{strconv.Itoa(n)} }
	switch {
	case strings.Contains(l, "body rows") || strings.Contains(l, "image row budget"):
		switch {
		case strings.Contains(l, "untitled"):
			return one(bodyRows(t, w, h, false, false)), true
		case strings.Contains(l, "jump_to_middle") || strings.Contains(l, "centered"):
			return one(bodyRows(t, w, h, true, true)), true
		default:
			return one(bodyRows(t, w, h, true, false)), true
		}
	case weightsRe.MatchString(l):
		var weights []int
		for _, f := range strings.Split(weightsRe.FindStringSubmatch(l)[1], ",") {
			n, _ := strconv.Atoi(strings.TrimSpace(f))
			weights = append(weights, n)
		}
		cols := render.ColumnWidths(weights, cw)
		if strings.Contains(l, "prose wrap") {
			return []string{fmt.Sprintf("%d/%d", cols[0]-2, cols[0]-4)}, true
		}
		var forms []string
		var parts []string
		equal := true
		for _, c := range cols {
			parts = append(parts, strconv.Itoa(c))
			equal = equal && c == cols[0]
		}
		forms = append(forms, strings.Join(parts, "+"))
		if equal {
			forms = append(forms, fmt.Sprintf("%dx%d", cols[0], len(cols)))
		}
		if strings.Contains(l, "code") {
			for i := range forms {
				forms[i] += fmt.Sprintf("(%d)", cols[0]-4)
			}
		}
		return forms, true
	case strings.Contains(l, "prose wrap") && strings.Contains(l, "code"):
		return []string{fmt.Sprintf("%d/%d", g, k)}, true
	case strings.Contains(l, "prose wrap"):
		return one(g), true
	case strings.Contains(l, "code limit") || strings.Contains(l, "code line") || strings.Contains(l, "code max"):
		return one(k), true
	case strings.Contains(l, "content width") || strings.Contains(l, "`cw`") || strings.Contains(l, "title width"):
		if strings.Contains(l, "indent") || strings.Contains(l, "margin") {
			return []string{fmt.Sprintf("%d/%d", cw, (w-cw)/2)}, true
		}
		return one(cw), true
	case strings.Contains(l, "left indent") || strings.Contains(l, "left margin"):
		return one((w - cw) / 2), true
	}
	return nil, false
}

// measuredContentWidth derives the content width from -check: a title of n
// cells overflows by n - cw columns.
func measuredContentWidth(t *testing.T, w, h int) int {
	const n = 300
	d, err := deck.Parse(strings.Repeat("x", n) + "\n===\n")
	if err != nil {
		t.Fatal(err)
	}
	_, cols := render.New(d).Overflow(0, w, h)
	return n - cols
}

// bodyRows derives the body row budget from -check: a code block of n lines
// overflows by n - budget rows.
func bodyRows(t *testing.T, w, h int, titled, centered bool) int {
	const n = 300
	var sb strings.Builder
	if centered {
		sb.WriteString("<!-- jump_to_middle -->\n")
	}
	if titled {
		sb.WriteString("Title\n===\n\n")
	}
	sb.WriteString("```text\n" + strings.Repeat("x\n", n) + "```\n")
	d, err := deck.Parse(sb.String())
	if err != nil {
		t.Fatal(err)
	}
	rows, _ := render.New(d).Overflow(0, w, h)
	return n - rows
}
