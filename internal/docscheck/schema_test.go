package docscheck

import (
	"encoding/json"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"

	"tuipresent/internal/deck"
	"tuipresent/internal/render"
	"tuipresent/internal/transition"
)

const schemaDoc = "docs/agent/frontmatter.schema.json"

type schemaNode struct {
	Ref        string                `json:"$ref"`
	Enum       []string              `json:"enum"`
	Examples   []string              `json:"examples"`
	Properties map[string]schemaNode `json:"properties"`
	AnyOf      []schemaNode          `json:"anyOf"`
	Defs       map[string]schemaNode `json:"$defs"`
}

func loadSchema(t *testing.T) schemaNode {
	t.Helper()
	var s schemaNode
	if err := json.Unmarshal([]byte(read(t, schemaDoc)), &s); err != nil {
		t.Fatalf("%s: %v", schemaDoc, err)
	}
	return s
}

func keys(m map[string]schemaNode) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

func sameSet(t *testing.T, what string, got, want []string) {
	t.Helper()
	got, want = slices.Clone(got), slices.Clone(want)
	slices.Sort(got)
	slices.Sort(want)
	want = slices.Compact(want)
	if !slices.Equal(got, want) {
		t.Errorf("%s: schema has %q, code has %q", what, got, want)
	}
}

// TestSchemaMatchesCode keeps frontmatter.schema.json in step with deck.Meta,
// the theme mapping keys and the transition kinds.
func TestSchemaMatchesCode(t *testing.T) {
	s := loadSchema(t)

	var metaKeys []string
	mt := reflect.TypeFor[deck.Meta]()
	for i := range mt.NumField() {
		if tag := strings.Split(mt.Field(i).Tag.Get("yaml"), ",")[0]; tag != "" && tag != "-" {
			metaKeys = append(metaKeys, tag)
		}
	}
	sameSet(t, "front matter keys", keys(s.Properties), metaKeys)

	// Theme mapping keys: those deck.Theme decodes itself, plus the palette
	// keys and code_style that render applies (background is both).
	src := read(t, "internal/deck/deck.go")
	fn := src[strings.Index(src, "func (t *Theme) UnmarshalYAML"):]
	fn = fn[:strings.Index(fn, "\n}\n")]
	var themeKeys []string
	for _, m := range regexp.MustCompile(`case "([a-z_]+)"`).FindAllStringSubmatch(fn, -1) {
		themeKeys = append(themeKeys, m[1])
	}
	if len(themeKeys) == 0 {
		t.Fatal("no theme mapping keys found in deck.go")
	}
	themeKeys = append(themeKeys, render.ThemeKeys()...)
	obj, ok := s.Defs["themeObject"]
	if !ok {
		t.Fatal("schema has no $defs.themeObject")
	}
	sameSet(t, "theme mapping keys", keys(obj.Properties), themeKeys)
	if slices.Contains(keys(obj.Properties), "extends") {
		t.Error("extends is only valid in theme files")
	}
	sameSet(t, "theme background values", obj.Properties["background"].Enum, []string{"terminal", "paint"})

	theme := s.Properties["theme"]
	var refs []string
	for _, a := range theme.AnyOf {
		refs = append(refs, a.Ref)
	}
	sameSet(t, "theme forms", refs, []string{"#/$defs/themeName", "#/$defs/themeObject"})
	sameSet(t, "theme names", s.Defs["themeName"].Examples, render.ThemeNames())

	var kinds []string
	for _, k := range transition.Kinds {
		kinds = append(kinds, string(k))
	}
	sameSet(t, "transition values", s.Properties["transition"].Enum, kinds)
}
