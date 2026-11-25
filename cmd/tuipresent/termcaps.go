package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/term"
)

// stdoutIsTTY reports whether stdout is a terminal.
func stdoutIsTTY() bool { return term.IsTerminal(os.Stdout.Fd()) }

// oscSupported reports whether the terminal can be sent OSC sequences
// (hyperlinks, progress): it must be a terminal, not a dumb one, and not the
// Linux console or macOS Terminal.app, which print or ignore them badly.
func oscSupported(getenv func(string) string, tty bool) bool {
	t := getenv("TERM")
	return tty && t != "" && t != "dumb" && t != "linux" && getenv("TERM_PROGRAM") != "Apple_Terminal"
}

// linksEnabled resolves the -links flag (auto, on or off).
func linksEnabled(mode string, getenv func(string) string, tty bool) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", "auto":
		return oscSupported(getenv, tty), nil
	case "on":
		return true, nil
	case "off":
		return false, nil
	}
	return false, fmt.Errorf("-links %q: want auto, on or off", mode)
}

// progressSeq is the OSC 9;4 sequence that shows the terminal tab's
// progress bar for slide (0-based) of slides: slide 1 of N reports 1/N, the
// last slide 100.
func progressSeq(slide, slides int) string {
	if slides < 1 {
		return progressClear
	}
	pct := min(max((slide+1)*100/slides, 0), 100)
	return fmt.Sprintf("\x1b]9;4;1;%d\x1b\\", pct)
}

// progressClear removes the progress bar.
const progressClear = "\x1b]9;4;0;0\x1b\\"

// withProgress reports the current slide as progress when it changed, and
// clears the bar when a reload switched progress off.
func (m *model) withProgress(cmd tea.Cmd) tea.Cmd {
	var seq string
	switch want := m.progress && m.deck.ProgressEnabled(); {
	case want && m.w > 0 && (!m.progressShown || m.progressSlide != m.slide || m.progressTotal != len(m.deck.Slides)):
		seq = progressSeq(m.slide, len(m.deck.Slides))
		m.progressShown, m.progressSlide, m.progressTotal = true, m.slide, len(m.deck.Slides)
	case !want && m.progressShown:
		seq = progressClear
		m.progressShown = false
	}
	if seq == "" {
		return cmd
	}
	return tea.Batch(tea.Raw(seq), cmd)
}

// cleanupOnSignal runs cleanup when the process is hung up or terminated,
// for the paths where the program does not get to exit normally. SIGHUP is
// swallowed by signal.Notify, so it must also end the process, or a closed
// terminal would leave the presenter running; SIGTERM ends the program
// through bubbletea, which then runs the normal cleanup again (cleanup must
// be idempotent). It returns a function that stops watching.
func cleanupOnSignal(cleanup func()) (stop func()) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGHUP, syscall.SIGTERM)
	done := make(chan struct{})
	go handleSignals(ch, done, cleanup, os.Exit)
	return func() { signal.Stop(ch); close(done) }
}

func handleSignals(ch <-chan os.Signal, done <-chan struct{}, cleanup func(), exit func(int)) {
	select {
	case sig := <-ch:
		cleanup()
		if sig == syscall.SIGHUP {
			exit(129)
		}
	case <-done:
	}
}
