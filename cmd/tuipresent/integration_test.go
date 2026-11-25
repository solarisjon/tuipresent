package main

import (
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"tuipresent/internal/deck"
	"tuipresent/internal/img"
	"tuipresent/internal/notes"
	"tuipresent/internal/render"
)

func TestHandleSignals(t *testing.T) {
	run := func(sig os.Signal) (cleaned bool, exit int) {
		ch := make(chan os.Signal, 1)
		done := make(chan struct{})
		finished := make(chan struct{})
		exit = -1
		go func() {
			handleSignals(ch, done, func() { cleaned = true }, func(c int) { exit = c })
			close(finished)
		}()
		ch <- sig
		<-finished
		return
	}
	if cleaned, exit := run(syscall.SIGHUP); !cleaned || exit != 129 {
		t.Errorf("SIGHUP: cleaned=%v exit=%d, want cleanup and exit 129 (a swallowed SIGHUP must not leave the presenter running)", cleaned, exit)
	}
	if cleaned, exit := run(syscall.SIGTERM); !cleaned || exit != -1 {
		t.Errorf("SIGTERM: cleaned=%v exit=%d, want cleanup and no direct exit", cleaned, exit)
	}
	// Stopping the watcher ends it without cleanup.
	ch, done, finished := make(chan os.Signal), make(chan struct{}), make(chan struct{})
	called := false
	go func() {
		handleSignals(ch, done, func() { called = true }, func(int) { called = true })
		close(finished)
	}()
	close(done)
	<-finished
	if called {
		t.Error("stopped watcher still ran cleanup")
	}
}

const monkeyDeck = `---
title: Monkey
author: A
progress: on
---
Intro [link](https://example.com)
===
<!-- title_size: 2 -->
<!-- subtitle: sub -->

* one
<!-- pause -->
* two [x]{.curly}

<!-- end_slide -->

Code
===
<!-- column_layout: [1, 1] -->
<!-- column: 0 -->
` + "```go {1|2}\na := 1\nb := 2\n```" + `
<!-- column: 1 -->
> [!NOTE]
> a callout with a [link](https://example.org)
<!-- pause -->
after

<!-- end_slide -->

Pic
===
![logo](assets/terminal.png)
<!-- transition: dissolve -->

<!-- end_slide -->

Last
===
<!-- jump_to_middle -->
<!-- title_size: 3 -->
bye
`

// TestMonkey drives the presenter with random keys, mouse, resizes, reloads
// and timer messages with every terminal capability switched on, checking the
// model's invariants and that View never panics or yields a frame of the
// wrong size.
func TestMonkey(t *testing.T) {
	defer func(i render.ImageConfig, h, c, ts bool) {
		render.Images, render.Hyperlinks, render.Curly, render.TextSizing = i, h, c, ts
	}(render.Images, render.Hyperlinks, render.Curly, render.TextSizing)
	render.Hyperlinks, render.Curly, render.TextSizing = true, true, true
	render.Images = render.ImageConfig{Mode: img.Sixel, Overlay: img.NewOverlay(img.Sixel)}

	dir := t.TempDir()
	path := filepath.Join(dir, "deck.md")
	if err := os.WriteFile(path, []byte(monkeyDeck), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(mustAbs(t, "../../examples/assets"), filepath.Join(dir, "assets")); err != nil {
		t.Fatal(err)
	}
	short := strings.Replace(monkeyDeck, "Last\n===", "Last\n===\n<!-- end_slide -->\nX\n===\nx", 1)
	shorter := "Only\n===\none\n"

	keys := []string{"right", "left", "space", "o", "esc", "enter", "g", "G", "1", "2", "3", "0", "backspace", "j", "k", "up", "down", "pgup", "pgdown", "r", "x"}
	for seed := range uint64(12) {
		rng := rand.New(rand.NewPCG(seed, 99))
		d, err := deck.Load(path)
		if err != nil {
			t.Fatal(err)
		}
		m := newModel(path, d, deckFlags{})
		m.progress, m.mouse, m.watch = true, true, true
		m.statePath = filepath.Join(dir, "state.json")
		m.remote = newRemote(m.statePath, time.Now())
		m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
		for i := range 400 {
			var msg tea.Msg
			switch n := rng.IntN(14); {
			case n < 5:
				m.key(keys[rng.IntN(len(keys))])
			case n == 5:
				msg = tea.MouseClickMsg{Button: tea.MouseLeft, X: rng.IntN(120), Y: rng.IntN(40)}
			case n == 6:
				msg = tea.MouseWheelMsg{Button: []tea.MouseButton{tea.MouseWheelUp, tea.MouseWheelDown}[rng.IntN(2)]}
			case n == 7:
				sizes := [][2]int{{100, 30}, {80, 24}, {60, 20}, {20, 5}, {1, 1}, {200, 60}, {7, 0}, {0, 0}, {120, 3}}
				s := sizes[rng.IntN(len(sizes))]
				msg = tea.WindowSizeMsg{Width: s[0], Height: s[1]}
			case n == 8:
				msg = tickMsg(time.Now())
			case n == 9:
				msg = overlayMsg{m.ovGen}
			case n == 10:
				msg = bigTitleMsg{m.big.gen}
			case n == 11:
				src := []string{monkeyDeck, short, shorter}[rng.IntN(3)]
				os.WriteFile(path, []byte(src), 0o644)
				msg = watchMsg{}
				m.stamp = nil
			case n == 12:
				m.anim = nil
				msg = remoteMsg{}
				cmds := []notes.Command{{Seq: time.Now().UnixNano(), Target: os.Getpid(), Op: notes.OpNext},
					{Seq: time.Now().UnixNano() + 1, Target: os.Getpid(), Op: notes.OpGoto, Slide: rng.IntN(9) - 2},
					{Seq: time.Now().UnixNano() + 2, Target: os.Getpid(), Op: notes.OpPrev}}
				notes.WriteCommands(m.remote.path, cmds[:1+rng.IntN(3)])
			default:
				msg = execMsg{id: 1, out: "out"}
			}
			if msg != nil {
				m.Update(msg)
			}
			// Invariants.
			if m.slide < 0 || m.slide >= len(m.deck.Slides) || m.step < 0 || m.step >= m.deck.Slides[m.slide].Steps {
				t.Fatalf("seed %d op %d: position %d/%d out of range", seed, i, m.slide, m.step)
			}
			if m.overview && (m.sel < 0 || m.sel >= len(m.deck.Slides)) {
				t.Fatalf("seed %d op %d: overview selection %d of %d", seed, i, m.sel, len(m.deck.Slides))
			}
			m.View()
			if m.w > 0 && m.h > 0 && !m.overview {
				for j, l := range strings.Split(m.screen(), "\n") {
					if w := ansi.StringWidth(l); w != m.w {
						t.Fatalf("seed %d op %d: screen line %d is %d wide, want %d", seed, i, j, w, m.w)
					}
				}
			}
		}
	}
}

func mustAbs(t *testing.T, p string) string {
	t.Helper()
	a, err := filepath.Abs(p)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

// vhs runs in the output directory, so a relative theme file given with
// -theme must reach the recorded command line as an absolute path.
func TestVHSTapeThemeFileIsAbsolute(t *testing.T) {
	_, d, r := loadExport(t)
	for mode, wantAbs := range map[string]bool{"vhs": true, "tape": false} {
		o := exportOpts{mode: mode, w: 80, h: 24, deckPath: "talk.md", flags: deckFlags{theme: "mine.yaml"}}
		tape, err := buildTape(o, d, r, "talk.md", "talk.gif")
		if err != nil {
			t.Fatal(err)
		}
		abs := mustAbs(t, "mine.yaml")
		if got := strings.Contains(tape, abs); got != wantAbs {
			t.Errorf("-export %s: absolute theme path present = %v, want %v:\n%s", mode, got, wantAbs, tape)
		}
	}
}
