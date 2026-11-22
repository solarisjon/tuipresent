package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"tuipresent/internal/deck"
	"tuipresent/internal/render"
)

func writeFile(t *testing.T, path, content string, mtime time.Time) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatal(err)
	}
}

func TestFileChanged(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "deck.md")
	t0 := time.Unix(1_700_000_000, 0)
	writeFile(t, path, "one", t0)
	prev := stat(path)

	if fileChanged(prev, stat(path)) {
		t.Error("unchanged file reported as changed")
	}
	if fileChanged(prev, stat(filepath.Join(dir, "missing.md"))) {
		t.Error("missing file reported as changed")
	}
	if !fileChanged(nil, prev) {
		t.Error("file appearing after being unreadable not reported")
	}

	writeFile(t, path, "two", t0.Add(time.Second))
	if !fileChanged(prev, stat(path)) {
		t.Error("mtime change not detected")
	}

	writeFile(t, path, "three!", t0)
	if !fileChanged(prev, stat(path)) {
		t.Error("size change not detected")
	}

	// Editor-style save: write a temp file and rename it over the original,
	// with identical size and mtime. Only the file identity differs.
	writeFile(t, path, "one", t0)
	prev = stat(path)
	tmp := filepath.Join(dir, "deck.md.tmp")
	writeFile(t, tmp, "uno", t0)
	if err := os.Rename(tmp, path); err != nil {
		t.Fatal(err)
	}
	if !fileChanged(prev, stat(path)) {
		t.Error("rename/replace not detected")
	}
}

const threeSlides = "A\n===\n\none\n<!-- pause -->\nmore\n<!-- end_slide -->\nB\n===\n\ntwo\n<!-- end_slide -->\nC\n===\n\nthree\n<!-- pause -->\nmore\n"

func newWatchModel(t *testing.T, path string, flags deckFlags) *model {
	t.Helper()
	d, err := deck.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	flags.apply(d)
	return &model{path: path, deck: d, r: render.New(d), flags: flags, watch: true, stamp: stat(path), themePath: themeFile(d), themeStamp: stat(themeFile(d))}
}

func TestPollWatchReloads(t *testing.T) {
	path := filepath.Join(t.TempDir(), "deck.md")
	t0 := time.Unix(1_700_000_000, 0)
	writeFile(t, path, threeSlides, t0)
	m := newWatchModel(t, path, deckFlags{transition: "wipe"})
	m.slide, m.step = 2, 1
	m.r.SetExecOutput(0, "old output")

	if m.pollWatch() == nil {
		t.Fatal("pollWatch must always schedule the next poll")
	}
	if m.status != "" {
		t.Fatalf("unchanged file triggered reload: status %q", m.status)
	}

	// Shrink to two slides; the current slide and step are clamped.
	writeFile(t, path, "---\ntransition: none\n---\nA\n===\n\none\n<!-- end_slide -->\nB\n===\n\ntwo\n", t0.Add(time.Second))
	m.pollWatch()
	if m.status != "reloaded" || len(m.deck.Slides) != 2 || m.slide != 1 || m.step != 0 {
		t.Fatalf("after reload: status %q, %d slides, slide %d step %d", m.status, len(m.deck.Slides), m.slide, m.step)
	}
	if m.deck.Meta.Transition != "wipe" {
		t.Errorf("-transition flag lost on reload: %q", m.deck.Meta.Transition)
	}

	// A parse error keeps the old deck and shows the error until a good save.
	writeFile(t, path, "---\ntitle: [unclosed\n---\nX\n===\n", t0.Add(2*time.Second))
	m.pollWatch()
	if m.status == "reloaded" || m.status == "" || len(m.deck.Slides) != 2 {
		t.Fatalf("bad file: status %q, %d slides", m.status, len(m.deck.Slides))
	}
	errStatus := m.status
	m.Update(clearStatusMsg{"reloaded"}) // stale timer from the earlier reload
	m.pollWatch()                        // file unchanged since the failed load
	if m.status != errStatus {
		t.Errorf("error status not kept: %q", m.status)
	}

	writeFile(t, path, threeSlides, t0.Add(3*time.Second))
	m.pollWatch()
	if m.status != "reloaded" || len(m.deck.Slides) != 3 {
		t.Fatalf("recovery: status %q, %d slides", m.status, len(m.deck.Slides))
	}
	if !strings.Contains(m.r.Frame(0, 0, 80, 24), "one") {
		t.Error("reloaded deck not rendered")
	}
}
