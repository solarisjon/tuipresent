package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/solarisjon/tuipresent/internal/render"
)

func TestThemeFlagBeatsFrontMatterAcrossReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "deck.md")
	t0 := time.Unix(1_700_000_000, 0)
	writeFile(t, path, "---\ntheme:\n  name: dracula\n  background: paint\n  accent: \"#ff0000\"\n---\nA\n===\n\none\n", t0)

	m := newWatchModel(t, path, deckFlags{theme: "nord"})
	th := m.r.Theme()
	nord, _ := render.LookupTheme("nord")
	if th.Name != "nord" || th.Palette.Accent != nord.Palette.Accent {
		t.Fatalf("-theme nord: got %s accent %s (front-matter overrides must not leak)", th.Name, th.Palette.Accent)
	}
	if !th.Paint {
		t.Error("-theme should keep the deck's background: paint")
	}

	writeFile(t, path, "---\ntheme: gruvbox-light\n---\nA\n===\n\ntwo\n", t0.Add(time.Second))
	m.pollWatch()
	if m.status != "reloaded" || m.r.Theme().Name != "nord" {
		t.Fatalf("after reload: status %q, theme %q; -theme must survive reload", m.status, m.r.Theme().Name)
	}

	// Without the flag the front matter decides.
	m = newWatchModel(t, path, deckFlags{})
	if m.r.Theme().Name != "gruvbox-light" {
		t.Errorf("front matter theme = %q", m.r.Theme().Name)
	}

	// -background overrides front matter too.
	m = newWatchModel(t, path, deckFlags{background: "paint"})
	if !m.r.Paints() {
		t.Error("-background paint ignored")
	}
}

func TestThemeFileFlagAndThemeFileReload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "deck.md")
	theme := filepath.Join(dir, "brand.yaml")
	t0 := time.Unix(1_700_000_000, 0)
	writeFile(t, theme, "extends: nord\naccent: \"#ff8800\"\n", t0)
	writeFile(t, path, "---\ntheme:\n  path: brand.yaml\n---\nA\n===\n\none\n", t0)

	m := newWatchModel(t, path, deckFlags{})
	if m.r.Theme().Palette.Accent != "#ff8800" {
		t.Fatalf("theme file not used: %+v", m.r.Theme())
	}

	// Editing only the theme file reloads.
	writeFile(t, theme, "extends: nord\naccent: \"#00ff88\"\n", t0.Add(time.Second))
	m.pollWatch()
	if m.r.Theme().Palette.Accent != "#00ff88" {
		t.Fatalf("theme file edit not picked up: status %q", m.status)
	}

	// A broken theme file keeps the old deck and shows the error.
	writeFile(t, theme, "accent: \"#nothex\"\n", t0.Add(2*time.Second))
	m.pollWatch()
	if !strings.Contains(m.status, "not a hex color") || m.r.Theme().Palette.Accent != "#00ff88" {
		t.Fatalf("bad theme: status %q, accent %s", m.status, m.r.Theme().Palette.Accent)
	}

	// -theme with a file path.
	other := filepath.Join(dir, "other.yml")
	writeFile(t, other, "extends: kanagawa\n", t0)
	m = newWatchModel(t, path, deckFlags{theme: other})
	if m.r.Theme().Name != "other" {
		t.Errorf("-theme file: %q", m.r.Theme().Name)
	}
}

func TestUnknownThemeWarnsInStatus(t *testing.T) {
	path := filepath.Join(t.TempDir(), "deck.md")
	writeFile(t, path, "---\ntheme: tokyo-nite\n---\nA\n===\n\none\n", time.Unix(1_700_000_000, 0))
	m := newWatchModel(t, path, deckFlags{})
	if m.r.Theme().Name != "dark" || len(m.r.ThemeWarnings()) == 0 {
		t.Fatalf("theme %q, warnings %q", m.r.Theme().Name, m.r.ThemeWarnings())
	}
	m.status = strings.Join(m.r.ThemeWarnings(), "; ")
	m.w, m.h = 80, 24
	lines := strings.Split(ansi.Strip(m.View().Content), "\n")
	if !strings.Contains(lines[len(lines)-1], "unknown theme") {
		t.Errorf("status line = %q", lines[len(lines)-1])
	}
}

func TestPrintThemes(t *testing.T) {
	var b bytes.Buffer
	printThemes(&b)
	lines := strings.Split(strings.TrimSpace(ansi.Strip(b.String())), "\n")
	names := render.ThemeNames()
	if len(lines) != len(names) {
		t.Fatalf("%d lines for %d themes", len(lines), len(names))
	}
	for i, n := range names {
		if !strings.HasPrefix(lines[i], n+" ") {
			t.Errorf("line %d = %q, want theme %s", i, lines[i], n)
		}
	}
}
