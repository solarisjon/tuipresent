package docs

import (
	"os"
	"strings"
	"testing"
)

// The embedded references are the repo files, byte for byte; SKILL.md is the
// repo file with only its doc paths rewritten.
func TestEmbeddedEqualsDisk(t *testing.T) {
	raw, err := os.ReadFile("agent-skill/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range SkillFiles() {
		want := ""
		if f.Path == "SKILL.md" {
			want = RewriteSkill(string(raw))
		} else {
			repo := strings.TrimPrefix(RepoPath(f.Path), "docs/")
			b, err := os.ReadFile(repo)
			if err != nil {
				t.Fatalf("%s: %v", f.Path, err)
			}
			want = string(b)
		}
		if string(f.Data) != want {
			t.Errorf("%s differs from its source", f.Path)
		}
	}
}

func TestRewriteSkill(t *testing.T) {
	in := "see `docs/AGENT_GUIDE.md`, docs/agent/LANGUAGE.md, docs/agent/EXAMPLES.md and docs/agent/frontmatter.schema.json"
	want := "see `references/GUIDE.md`, references/LANGUAGE.md, references/EXAMPLES.md and references/frontmatter.schema.json"
	if got := RewriteSkill(in); got != want {
		t.Errorf("got %q", got)
	}
	if RewriteSkill(want) != want {
		t.Error("not idempotent")
	}
	if RewriteSkill("nothing here") != "nothing here" {
		t.Error("changed unrelated text")
	}
}
