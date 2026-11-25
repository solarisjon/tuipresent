// Package docs embeds the agent skill and the reference material it points
// at, so the tuipresent binary can install them (tuipresent -install-skill)
// without a checkout of the repository.
package docs

import (
	_ "embed"
	"strings"
)

//go:embed agent-skill/SKILL.md
var skillMD string

//go:embed AGENT_GUIDE.md
var guideMD string

//go:embed agent/LANGUAGE.md
var languageMD string

//go:embed agent/EXAMPLES.md
var examplesMD string

//go:embed agent/frontmatter.schema.json
var schemaJSON string

// SkillName is the skill's name and the name of its install directory.
const SkillName = "tuipresent-deck"

// File is one file of the installed skill: Path is slash-separated and
// relative to the skill directory.
type File struct {
	Path string
	Data []byte
}

// pathMap maps each repo-relative path the skill uses to its place in the
// installed skill directory.
var pathMap = []struct{ repo, installed string }{
	{"docs/AGENT_GUIDE.md", "references/GUIDE.md"},
	{"docs/agent/LANGUAGE.md", "references/LANGUAGE.md"},
	{"docs/agent/EXAMPLES.md", "references/EXAMPLES.md"},
	{"docs/agent/frontmatter.schema.json", "references/frontmatter.schema.json"},
}

// RepoPath is the repository path of the doc installed at installed, "" if none.
func RepoPath(installed string) string {
	for _, m := range pathMap {
		if m.installed == installed {
			return m.repo
		}
	}
	return ""
}

// RewriteSkill replaces every repo-relative doc path in SKILL.md text with
// the path of the installed copy under references/. It is deterministic and
// idempotent.
func RewriteSkill(s string) string {
	for _, m := range pathMap {
		s = strings.ReplaceAll(s, m.repo, m.installed)
	}
	return s
}

// SkillMD is the installed SKILL.md.
func SkillMD() string { return RewriteSkill(skillMD) }

// SkillFiles is every file of the installed skill, in write order.
func SkillFiles() []File {
	return []File{
		{"SKILL.md", []byte(SkillMD())},
		{"references/GUIDE.md", []byte(guideMD)},
		{"references/LANGUAGE.md", []byte(languageMD)},
		{"references/EXAMPLES.md", []byte(examplesMD)},
		{"references/frontmatter.schema.json", []byte(schemaJSON)},
	}
}
