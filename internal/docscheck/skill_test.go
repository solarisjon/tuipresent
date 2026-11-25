package docscheck

import (
	"regexp"
	"strings"
	"testing"

	skilldocs "github.com/solarisjon/tuipresent/docs"
)

var (
	skillFlagRe = regexp.MustCompile("(?:^|[ `(])-([a-z][a-z-]*)")
	skillRefRe  = regexp.MustCompile(`references/[A-Za-z0-9_.-]+`)
	// anything that would point outside the installed skill directory
	skillEscapeRe = regexp.MustCompile(`\.\./|(^|[\s(` + "`" + `])(/|~/)[A-Za-z]|docs/|examples/|internal/`)
)

// TestSkillFlagsExist requires every -flag the skill's text uses to be a
// real CLI flag, so the skill's example commands cannot drift.
func TestSkillFlagsExist(t *testing.T) {
	src := read(t, "cmd/tuipresent/main.go")
	known := map[string]bool{}
	for _, m := range flagRe.FindAllStringSubmatch(src, -1) {
		known[m[1]] = true
	}
	// Flags of other tools in the skill's example commands (perl -pe).
	other := map[string]bool{"pe": true}
	n := 0
	for _, line := range strings.Split(read(t, "docs/agent-skill/SKILL.md"), "\n") {
		for _, m := range skillFlagRe.FindAllStringSubmatch(line, -1) {
			n++
			if !known[m[1]] && !other[m[1]] {
				t.Errorf("SKILL.md uses unknown flag -%s in %q", m[1], strings.TrimSpace(line))
			}
		}
	}
	if n == 0 {
		t.Fatal("no flags found in SKILL.md")
	}
}

// TestSkillRepoPaths requires the repo SKILL.md to name only doc files that
// exist, and the installed one to point only inside the skill directory.
func TestSkillRepoPaths(t *testing.T) {
	repo := read(t, "docs/agent-skill/SKILL.md")
	for _, f := range skilldocs.SkillFiles()[1:] {
		rp := skilldocs.RepoPath(f.Path)
		if !strings.Contains(repo, rp) {
			t.Errorf("SKILL.md does not point at %s", rp)
		}
		read(t, rp) // exists
	}
	if skillRefRe.MatchString(repo) {
		t.Error("repo SKILL.md already uses installed references/ paths; use the docs/ paths and let the installer rewrite them")
	}
}

func TestInstalledSkillStaysInside(t *testing.T) {
	installed := map[string]bool{}
	for _, f := range skilldocs.SkillFiles() {
		installed[f.Path] = true
	}
	text := skilldocs.SkillMD()
	refs := skillRefRe.FindAllString(text, -1)
	if len(refs) < 4 {
		t.Errorf("installed SKILL.md names %d reference files, want all 4: %v", len(refs), refs)
	}
	for _, r := range refs {
		if !installed[r] {
			t.Errorf("installed SKILL.md points at %s, which is not installed", r)
		}
	}
	for _, line := range strings.Split(text, "\n") {
		if skillEscapeRe.MatchString(line) {
			t.Errorf("installed SKILL.md has a path outside the skill dir: %q", strings.TrimSpace(line))
		}
	}
}
