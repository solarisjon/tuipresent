package main

import (
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
)

// watchInterval is how often the deck file is polled for changes.
const watchInterval = 400 * time.Millisecond

type watchMsg struct{}

// clearStatusMsg clears the status line only if it still shows text, so a
// stale timer never wipes a newer message such as a parse error.
type clearStatusMsg struct{ text string }

func watchTick() tea.Cmd {
	return tea.Tick(watchInterval, func(time.Time) tea.Msg { return watchMsg{} })
}

// stat returns the deck file's info, or nil if it can't be read right now
// (for example mid-way through an editor's rename-and-replace save).
func stat(path string) os.FileInfo {
	fi, err := os.Stat(path)
	if err != nil {
		return nil
	}
	return fi
}

// fileChanged reports whether the deck should be reloaded. A missing file
// (cur == nil) is not a change: keep the old deck and look again later.
// A different file at the same path (rename/replace saves) is a change even
// if its size and mtime happen to match.
func fileChanged(prev, cur os.FileInfo) bool {
	if cur == nil {
		return false
	}
	if prev == nil {
		return true
	}
	return !os.SameFile(prev, cur) || !prev.ModTime().Equal(cur.ModTime()) || prev.Size() != cur.Size()
}

// pollWatch reloads the deck if its file changed and schedules the next poll.
func (m *model) pollWatch() tea.Cmd {
	if !fileChanged(m.stamp, stat(m.path)) {
		return watchTick()
	}
	return tea.Batch(m.reload(), watchTick())
}
