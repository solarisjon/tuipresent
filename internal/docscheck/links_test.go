package docscheck

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"tuipresent/internal/deck"
)

var (
	linkRe       = regexp.MustCompile(`\]\(\s*<?([^)\s>]+)>?(?:\s+"[^"]*")?\s*\)`)
	inlineCodeRe = regexp.MustCompile("``[^`]+``|`[^`]+`")
)

// linkedDocs are the markdown files whose relative links must resolve.
func linkedDocs(t *testing.T) []string {
	t.Helper()
	out := []string{"README.md", "llms.txt", "examples/README.md"}
	err := filepath.WalkDir(filepath.Join(root, "docs"), func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(p, ".md") {
			rel, _ := filepath.Rel(root, p)
			out = append(out, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// TestRelativeLinksResolve fails on a relative link (outside code) in the
// docs that points at a file that doesn't exist. Anchors aren't checked.
func TestRelativeLinksResolve(t *testing.T) {
	n := 0
	for _, doc := range linkedDocs(t) {
		for _, target := range relativeLinks(read(t, doc)) {
			n++
			path := target
			if i := strings.IndexByte(path, '#'); i >= 0 {
				path = path[:i]
			}
			if path == "" {
				continue // same-page anchor
			}
			full := filepath.Join(root, filepath.Dir(doc), filepath.FromSlash(path))
			if _, err := os.Stat(full); err != nil {
				t.Errorf("%s: link %q: no such file", doc, target)
			}
		}
	}
	if n == 0 {
		t.Fatal("no links found")
	}
}

// relativeLinks returns the targets of markdown links and images outside
// fenced code and inline code, skipping URLs.
func relativeLinks(md string) []string {
	var out []string
	fence := ""
	for _, l := range strings.Split(md, "\n") {
		if fence == "" {
			if m, _, ok := deck.FenceOpen(l); ok {
				fence = m
				continue
			}
		} else {
			if deck.ClosesFence(fence, l) {
				fence = ""
			}
			continue
		}
		l = inlineCodeRe.ReplaceAllString(l, "")
		for _, m := range linkRe.FindAllStringSubmatch(l, -1) {
			target := m[1]
			if strings.Contains(target, "://") || strings.HasPrefix(target, "mailto:") {
				continue
			}
			out = append(out, target)
		}
	}
	return out
}
