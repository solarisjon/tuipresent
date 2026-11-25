package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/solarisjon/tuipresent/internal/deck"
)

func TestRunBlockSupportedLanguages(t *testing.T) {
	for _, tc := range []struct{ lang, text, want string }{
		{"bash", "echo hi", "hi"},
		{"", "echo hi", "hi"},
		{"sh", "echo hi", "hi"},
		{"python", "print('hi')", "hi"},
		{"python3", "print('hi')", "hi"},
		{"py", "print('hi')", "hi"},
	} {
		got := runBlock(deck.Block{Kind: deck.Exec, Lang: tc.lang, Text: tc.text}, t.TempDir())
		if strings.TrimSpace(got) != tc.want {
			t.Errorf("lang %q: got %q, want %q", tc.lang, got, tc.want)
		}
	}
}

func TestRunBlockUnsupportedLanguageErrors(t *testing.T) {
	for _, lang := range []string{"go", "js", "ruby", "zsh"} {
		dir := t.TempDir()
		// The body would create a file if it ran through bash.
		got := runBlock(deck.Block{Kind: deck.Exec, Lang: lang, Text: "touch ran"}, dir)
		want := `unsupported +exec language "` + lang + `" (supported: bash, sh, python, python3, py)`
		if !strings.Contains(got, want) {
			t.Errorf("lang %q: got %q, want it to contain %q", lang, got, want)
		}
		if matches, _ := filepath.Glob(filepath.Join(dir, "ran")); len(matches) > 0 {
			t.Errorf("lang %q: body ran", lang)
		}
	}
}
