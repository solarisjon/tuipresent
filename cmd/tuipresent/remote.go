package main

import (
	"os"
	"strconv"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/solarisjon/tuipresent/internal/notes"
)

// Navigation from the notes view. The notes view writes commands (see
// notes.Command) to a file next to the state file; the presenter polls it and
// applies each command through its normal navigation, then publishes the
// highest Seq it applied as State.Ack so the notes view stops resending it.
// The presenter never writes the command file, so the two cannot feed back.

const (
	remotePoll   = 100 * time.Millisecond
	remoteMaxAge = 5 * time.Second // older commands are dropped unapplied
)

type remoteMsg struct{}

// remote is the presenter's side of notes-view control.
type remote struct {
	path    string
	applied int64 // highest Seq applied; starts at the presenter's start time
}

func newRemote(statePath string, now time.Time) remote {
	r := remote{path: notes.CommandPath(statePath), applied: now.UnixNano()}
	os.Remove(r.path) // leftovers from an earlier presenter
	return r
}

func remoteTick() tea.Cmd {
	return tea.Tick(remotePoll, func(time.Time) tea.Msg { return remoteMsg{} })
}

// pollRemote applies pending notes-view commands and schedules the next poll.
func (m *model) pollRemote() tea.Cmd {
	cmds := []tea.Cmd{remoteTick()}
	for _, c := range m.takeRemote(time.Now()) {
		cmds = append(cmds, m.applyRemote(c))
	}
	return tea.Batch(cmds...)
}

// takeRemote returns the fresh, unapplied commands for this presenter and
// marks them (and any stale ones) applied.
func (m *model) takeRemote(now time.Time) []notes.Command {
	if m.remote.path == "" {
		return nil
	}
	pending := notes.Pending(notes.ReadCommands(m.remote.path), os.Getpid(), m.remote.applied)
	var fresh []notes.Command
	for _, c := range pending {
		m.remote.applied = c.Seq
		if now.Sub(time.Unix(0, c.Seq)) <= remoteMaxAge {
			fresh = append(fresh, c)
		}
	}
	return fresh
}

// send queues c for the running presenter. With no presenter it does nothing,
// so the notes view stays read-only.
func (m *notesModel) send(c notes.Command) {
	if !m.running {
		return
	}
	c.Seq = max(time.Now().UnixNano(), m.lastSeq+1)
	m.lastSeq, c.Target = c.Seq, m.state.PID
	m.pending = append(m.pending, c)
	m.writePending()
}

// syncCommands drops commands the presenter acknowledged, or that were
// addressed to a presenter that is gone, and removes an orphaned file.
func (m *notesModel) syncCommands() {
	if len(m.pending) == 0 {
		return
	}
	var keep []notes.Command
	if m.running {
		keep = notes.Pending(m.pending, m.state.PID, m.state.Ack)
	}
	if len(keep) == len(m.pending) {
		return
	}
	m.pending = keep
	m.writePending()
}

func (m *notesModel) writePending() {
	path := notes.CommandPath(m.statePath)
	if len(m.pending) == 0 {
		os.Remove(path)
		return
	}
	notes.WriteCommands(path, m.pending)
}

// status is shown on the notes view's "Next:" line.
func (m *notesModel) status() string {
	switch {
	case m.keys.Typing() != "":
		return "go to slide " + m.keys.Typing() + "▏ of " + strconv.Itoa(len(m.deck.Slides)) + " (enter jumps)"
	case m.running:
		return "keys drive the presenter"
	}
	return ""
}

// applyRemote performs c exactly as the equivalent key press would.
func (m *model) applyRemote(c notes.Command) tea.Cmd {
	m.overview, m.num = false, ""
	last := len(m.deck.Slides) - 1
	switch c.Op {
	case notes.OpNext:
		return m.forward()
	case notes.OpPrev:
		return m.back()
	case notes.OpFirst:
		return m.jump(0, 0)
	case notes.OpLast:
		return m.jump(last, 0)
	case notes.OpGoto:
		return m.jump(min(max(c.Slide, 1), last+1)-1, 0)
	}
	return nil
}
