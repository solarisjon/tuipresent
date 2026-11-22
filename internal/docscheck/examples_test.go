package docscheck

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"

	"tuipresent/internal/deck"
	"tuipresent/internal/render"
	"tuipresent/internal/report"
)

var update = flag.Bool("update", false, "rewrite the generated parts of docs/agent/EXAMPLES.md from real output")

const (
	examplesDoc = "docs/agent/EXAMPLES.md"
	exW, exH    = 80, 24
	checkCmd    = "$ tuipresent -check -w 80 -h 24 deck.md"
	fullFrames  = "<!-- frames: full -->" // keep the footer rows in this case's frames
)

var (
	caseHeadRe = regexp.MustCompile(`^## (\d+)\. (.+)$`)
	halfRe     = regexp.MustCompile(`^( *)([▀▄ ]*[▀▄][▀▄ ]*)$`)
)

// exampleCase locates one case of EXAMPLES.md: the deck source and the line
// ranges (exclusive of the fences) of its generated -check and -dump blocks.
type exampleCase struct {
	num, head      int // case number, index of its heading line
	title          string
	src            string
	full           bool
	check, frames  [2]int
	checkBlockSeen bool
}

// TestExamples keeps docs/agent/EXAMPLES.md true: each case's -check and
// -dump blocks must be what the renderer produces for its deck. Run
// `go test ./internal/docscheck -run Examples -update` to regenerate them
// (and the index) after changing the parser or renderer, then review the diff.
// The prose under each case is hand-written; reread it when its output changes.
func TestExamples(t *testing.T) {
	lines := strings.Split(read(t, examplesDoc), "\n")
	cases := parseExamples(t, lines)
	if len(cases) == 0 {
		t.Fatal("no cases found")
	}
	photo := filepath.Join(root, "examples", "assets", "mt-evans-goat.jpg")
	type repl struct {
		at   [2]int
		with []string
	}
	var repls []repl
	for i, c := range cases {
		if c.num != i+1 {
			t.Errorf("case %q is numbered %d, want %d", c.title, c.num, i+1)
		}
		check, frames := runExample(t, c.src, c.full, photo)
		repls = append(repls, repl{c.check, check}, repl{c.frames, frames})
		if *update {
			continue
		}
		if got := lines[c.check[0]:c.check[1]]; !equal(got, check) {
			t.Errorf("case %d (%s): -check block differs\n EXAMPLES.md:\n%s\n renderer:\n%s", c.num, c.title, strings.Join(got, "\n"), strings.Join(check, "\n"))
		}
		if got := lines[c.frames[0]:c.frames[1]]; !equal(got, frames) {
			t.Errorf("case %d (%s): -dump block differs; run go test ./internal/docscheck -run Examples -update and review the diff", c.num, c.title)
		}
	}

	// The index lists every case in order.
	start, end := indexRange(t, lines)
	var index []string
	for i, c := range cases {
		index = append(index, fmt.Sprintf("%d. [%s](#case-%d)", i+1, c.title, i+1))
	}
	if !*update {
		if !equal(lines[start:end], index) {
			t.Errorf("the index differs from the case headings; run with -update")
		}
		return
	}
	repls = append([]repl{{[2]int{start, end}, index}}, repls...)
	var out []string
	pos := 0
	for _, r := range repls {
		out = append(out, lines[pos:r.at[0]]...)
		out = append(out, r.with...)
		pos = r.at[1]
	}
	out = append(out, lines[pos:]...)
	if err := os.WriteFile(filepath.Join(root, examplesDoc), []byte(strings.Join(out, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Log("rewrote " + examplesDoc + "; review the diff and the prose under changed cases")
}

func equal(a, b []string) bool {
	return strings.Join(a, "\n") == strings.Join(b, "\n")
}

func parseExamples(t *testing.T, lines []string) []exampleCase {
	t.Helper()
	var cases []exampleCase
	var cur *exampleCase
	for i := 0; i < len(lines); i++ {
		l := lines[i]
		if m := caseHeadRe.FindStringSubmatch(l); m != nil {
			var n int
			fmt.Sscan(m[1], &n)
			cases = append(cases, exampleCase{num: n, head: i, title: m[2]})
			cur = &cases[len(cases)-1]
			continue
		}
		if cur == nil {
			continue
		}
		switch {
		case l == fullFrames:
			cur.full = true
		case l == "````markdown" && cur.src == "":
			j := i + 1
			for j < len(lines) && lines[j] != "````" {
				j++
			}
			cur.src = strings.Join(lines[i+1:j], "\n") + "\n"
			i = j
		case l == "```text" && cur.src != "":
			j := i + 1
			for j < len(lines) && lines[j] != "```" {
				j++
			}
			if !cur.checkBlockSeen {
				cur.check, cur.checkBlockSeen = [2]int{i + 1, j}, true
			} else if cur.frames == [2]int{} {
				cur.frames = [2]int{i + 1, j}
			}
			i = j
		}
	}
	for _, c := range cases {
		if c.src == "" || c.frames == [2]int{} {
			t.Fatalf("case %d (%s): want a ````markdown deck block, then a ```text -check block and a ```text -dump block", c.num, c.title)
		}
	}
	return cases
}

func indexRange(t *testing.T, lines []string) (int, int) {
	t.Helper()
	for i, l := range lines {
		if l == "## Index" {
			start := i + 2
			end := start
			for end < len(lines) && lines[end] != "" {
				end++
			}
			return start, end
		}
	}
	t.Fatal("no ## Index section")
	return 0, 0
}

// runExample renders src as deck.md in a scratch directory (with
// examples/assets/mt-evans-goat.jpg as photo.jpg) and returns its -check
// block (warnings, then the report, then the exit status) and its -dump block.
func runExample(t *testing.T, src string, full bool, photo string) (check, frames []string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "deck.md")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(src, "photo.jpg") {
		b, err := os.ReadFile(photo)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "photo.jpg"), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	check = []string{checkCmd}
	d, err := deck.Load(path)
	if err != nil {
		return append(check, err.Error()+"   (exit 1)"), nil
	}
	r := render.New(d)
	var sb strings.Builder
	report.PrintWarnings(&sb, "deck.md", report.Warnings(d, r, ""))
	exit := 0
	if report.Check(&sb, "deck.md", d, r, exW, exH) > 0 {
		exit = 1
	}
	check = append(check, strings.Split(strings.TrimRight(sb.String(), "\n"), "\n")...)
	check[len(check)-1] += fmt.Sprintf("   (exit %d)", exit)

	for i, s := range d.Slides {
		for st := range s.Steps {
			frames = append(frames, fmt.Sprintf("=== slide %d step %d/%d ===", i+1, st+1, s.Steps))
			frames = append(frames, excerpt(r.Frame(i, st, exW, exH), full)...)
		}
	}
	return check, frames
}

// excerpt is a frame as EXAMPLES.md shows it: ANSI stripped, trailing spaces
// removed and, unless full, the footer rows and trailing blank rows dropped
// and half-block image rows abbreviated.
func excerpt(frame string, full bool) []string {
	lines := strings.Split(ansi.Strip(frame), "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	if full {
		return lines
	}
	lines = lines[:exH-2]
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	for i, l := range lines {
		if m := halfRe.FindStringSubmatch(l); m != nil {
			lines[i] = fmt.Sprintf("%s<%d half-block cells>", m[1], utf8.RuneCountInString(m[2]))
		}
	}
	return lines
}
