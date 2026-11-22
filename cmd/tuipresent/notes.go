package main

import (
	"os"
	"time"

	tea "charm.land/bubbletea/v2"

	"tuipresent/internal/deck"
	"tuipresent/internal/notes"
	"tuipresent/internal/render"
)

const notesPoll = 200 * time.Millisecond

// publish writes the presenter's position for `tuipresent -notes` when it changed.
func (m *model) publish() {
	if m.statePath == "" {
		return
	}
	s := notes.State{PID: os.Getpid(), Slide: m.slide, Step: m.step, Started: m.started}
	if m.published != nil && *m.published == s {
		return
	}
	if notes.Write(m.statePath, s) == nil {
		m.published = &s
	}
}

type pollMsg struct{}

// notesModel follows a presenter of the same deck via its state file.
type notesModel struct {
	path      string
	statePath string
	deck      *deck.Deck
	flags     deckFlags
	r         *render.Renderer
	mtime     time.Time
	w, h      int
	state     notes.State
	running   bool
}

func runNotes(path string, d *deck.Deck, flags deckFlags) error {
	m := &notesModel{path: path, statePath: notes.StatePath(path), deck: d, flags: flags, r: render.New(d)}
	if fi, err := os.Stat(path); err == nil {
		m.mtime = fi.ModTime()
	}
	m.state, m.running = notes.Read(m.statePath)
	_, err := tea.NewProgram(m).Run()
	return err
}

func (m *notesModel) Init() tea.Cmd { return poll() }

func poll() tea.Cmd {
	return tea.Tick(notesPoll, func(time.Time) tea.Msg { return pollMsg{} })
}

func (m *notesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
	case pollMsg:
		m.state, m.running = notes.Read(m.statePath)
		m.reloadIfChanged()
		return m, poll()
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		}
	}
	return m, nil
}

// reloadIfChanged keeps notes current when the deck is edited during a talk.
func (m *notesModel) reloadIfChanged() {
	fi, err := os.Stat(m.path)
	if err != nil || fi.ModTime().Equal(m.mtime) {
		return
	}
	if d, err := deck.Load(m.path); err == nil {
		m.flags.apply(d)
		m.deck, m.r, m.mtime = d, render.New(d), fi.ModTime()
	}
}

func (m *notesModel) View() tea.View {
	v := tea.NewView(notes.Frame(m.deck, m.r, m.path, m.state, m.running, time.Now(), m.w, m.h))
	v.AltScreen = true
	return v
}
