package deck

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var highlightRe = regexp.MustCompile(`\{([^{}]*)\}`)

// splitHighlight removes the line-highlight spec, `{1-3|5}`, from a fence
// info string. It returns the info without it, the spec (without braces),
// and whether one was found. Only the first spec counts; an opening brace
// with no closing one is left in the info (and found is false).
func splitHighlight(info string) (rest, spec string, found bool) {
	m := highlightRe.FindStringSubmatchIndex(info)
	if m == nil {
		return info, "", false
	}
	spec = info[m[2]:m[3]]
	rest = info[:m[0]] + " " + highlightRe.ReplaceAllString(info[m[1]:], " ")
	return rest, spec, true
}

// ParseHighlights parses a line-highlight spec for a code block of n lines:
// `|`-separated groups, each a comma-separated list of 1-based lines (`5`),
// ranges (`2-4`) or `all`. Each group is one reveal step, shown with only its
// lines undimmed. The result has one []bool of length n per usable group;
// groups that are empty, or select no line of the block, are dropped. Every
// problem is returned as a warning message and the rest of the spec still
// applies: a bad item is skipped, and a range running past the end is clipped.
func ParseHighlights(spec string, n int) (groups [][]bool, warns []string) {
	for gi, raw := range strings.Split(spec, "|") {
		sel := make([]bool, n)
		anySel := false
		items := 0
		for _, item := range strings.Split(raw, ",") {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			items++
			lo, hi, ok := parseLineItem(item)
			if !ok {
				warns = append(warns, fmt.Sprintf("bad highlight %q: expected a line, a range like 2-4, or all", item))
				continue
			}
			if lo == 0 { // all
				lo, hi = 1, n
			}
			if lo > n {
				warns = append(warns, fmt.Sprintf("highlight %q is past the end of the %d-line code block", item, n))
				continue
			}
			if hi > n {
				warns = append(warns, fmt.Sprintf("highlight %q runs past the end of the %d-line code block; clipped to line %d", item, n, n))
				hi = n
			}
			for l := lo; l <= hi; l++ {
				sel[l-1] = true
				anySel = true
			}
		}
		switch {
		case items == 0:
			warns = append(warns, fmt.Sprintf("empty highlight group %d is ignored", gi+1))
		case anySel:
			groups = append(groups, sel)
		}
		// A group whose items were all bad or past the end has warned already.
	}
	return groups, warns
}

// parseLineItem parses `N`, `N-M` or `all`. All returns lo == 0.
func parseLineItem(s string) (lo, hi int, ok bool) {
	if strings.EqualFold(s, "all") {
		return 0, 0, true
	}
	a, b, isRange := strings.Cut(s, "-")
	lo, err := strconv.Atoi(strings.TrimSpace(a))
	if err != nil || lo < 1 {
		return 0, 0, false
	}
	if !isRange {
		return lo, lo, true
	}
	hi, err = strconv.Atoi(strings.TrimSpace(b))
	if err != nil || hi < lo {
		return 0, 0, false
	}
	return lo, hi, true
}
