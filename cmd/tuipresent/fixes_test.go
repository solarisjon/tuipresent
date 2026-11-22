package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"tuipresent/internal/deck"
	"tuipresent/internal/transition"
)

func TestBackwardUsesDepartingSlidesEffectiveTransition(t *testing.T) {
	src := "---\ntransition: wipe\n---\n" +
		"A\n<!-- transition: cover -->\n<!-- end_slide -->\n" + // slide 1: own cover
		"B\n<!-- end_slide -->\n" + // slide 2: deck default (wipe)
		"C\n<!-- transition: dissolve -->\n" // slide 3: own dissolve
	path := filepath.Join(t.TempDir(), "deck.md")
	writeFile(t, path, src, time.Unix(1_700_000_000, 0))
	for _, tc := range []struct {
		name     string
		flags    deckFlags
		from, to int
		want     transition.Kind
	}{
		{"forward uses destination's default", deckFlags{}, 0, 1, transition.Wipe},
		{"forward uses destination's directive", deckFlags{}, 1, 2, transition.Dissolve},
		{"back from directive slide", deckFlags{}, 2, 1, transition.Dissolve},
		// B came in with the deck default; going back must not use A's cover.
		{"back from default slide", deckFlags{}, 1, 0, transition.Wipe},
		{"back from default slide with -transition", deckFlags{transition: "none"}, 1, 0, transition.None},
		{"jump back", deckFlags{}, 2, 0, transition.Dissolve},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := newWatchModel(t, path, tc.flags)
			m.w, m.h = 40, 12
			m.slide = tc.from
			m.jump(tc.to, 0)
			var got transition.Kind = transition.None
			if m.anim != nil {
				got = m.anim.kind
				if m.anim.reverse != (tc.to < tc.from) {
					t.Errorf("reverse = %v", m.anim.reverse)
				}
			}
			if got != tc.want {
				t.Errorf("kind = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestExecRunsInDeckDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "data.txt"), []byte("found\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := runBlock(deck.Block{Kind: deck.Exec, Lang: "sh", Text: "cat data.txt"}, dir)
	if strings.TrimSpace(out) != "found" {
		t.Errorf("output = %q, want the deck directory's data.txt", out)
	}
}

func TestExecChildHoldingPipeCannotOutlastTimeout(t *testing.T) {
	old := execTimeout
	execTimeout = 300 * time.Millisecond
	defer func() { execTimeout = old }()
	start := time.Now()
	// The background sleep inherits stdout and would keep the pipe open for 30 s.
	out := runBlock(deck.Block{Kind: deck.Exec, Lang: "sh", Text: "sleep 30 & echo started"}, t.TempDir())
	if d := time.Since(start); d > 5*time.Second {
		t.Fatalf("run took %v, want well under the child's 30 s", d)
	}
	if !strings.Contains(out, "started") {
		t.Errorf("output = %q", out)
	}
}

func TestDeckWarningsInStatus(t *testing.T) {
	path := filepath.Join(t.TempDir(), "deck.md")
	writeFile(t, path, "A\n===\n\n<!-- transition: wpie -->\none\n", time.Unix(1_700_000_000, 0))
	d, err := deck.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	flags := deckFlags{transition: "swoosh"}
	flags.apply(d)
	m := newModel(path, d, flags)
	for _, want := range []string{`-transition: unknown transition "swoosh"`, `line 4: unknown transition "wpie"`} {
		if !strings.Contains(m.status, want) {
			t.Errorf("status %q lacks %q", m.status, want)
		}
	}
	m.w, m.h = 200, 24
	lines := strings.Split(ansi.Strip(m.View().Content), "\n")
	if !strings.Contains(lines[len(lines)-1], "unknown transition") {
		t.Errorf("status line = %q", lines[len(lines)-1])
	}
	// Reload keeps reporting them.
	m.reload()
	if !strings.Contains(m.status, `"wpie"`) {
		t.Errorf("status after reload = %q", m.status)
	}
}
