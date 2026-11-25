package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/solarisjon/tuipresent/internal/deck"
	"github.com/solarisjon/tuipresent/internal/notes"
	"github.com/solarisjon/tuipresent/internal/render"
)

func remoteModel(t *testing.T) *model {
	t.Helper()
	path := filepath.Join(t.TempDir(), "deck.md")
	writeFile(t, path, threeSlides, time.Unix(1_700_000_000, 0))
	d, err := deck.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	m := &model{path: path, deck: d, r: render.New(d), noFade: true}
	m.statePath = filepath.Join(t.TempDir(), "state.json")
	m.remote = newRemote(m.statePath, time.Now())
	return m
}

func sendCmds(t *testing.T, m *model, cmds ...notes.Command) {
	t.Helper()
	if err := notes.WriteCommands(m.remote.path, cmds); err != nil {
		t.Fatal(err)
	}
}

func apply(m *model, now time.Time) {
	for _, c := range m.takeRemote(now) {
		m.applyRemote(c)
	}
}

func TestRemoteApplies(t *testing.T) {
	m := remoteModel(t)
	now := time.Now()
	seq := now.UnixNano() + 1
	pid := os.Getpid()
	sendCmds(t, m,
		notes.Command{Seq: seq, Target: pid, Op: notes.OpNext},     // step 1 of slide 0
		notes.Command{Seq: seq + 1, Target: pid, Op: notes.OpNext}, // slide 1
	)
	apply(m, now)
	if m.slide != 1 || m.step != 0 {
		t.Fatalf("slide %d step %d", m.slide, m.step)
	}
	if m.remote.applied != seq+1 {
		t.Errorf("applied %d", m.remote.applied)
	}

	// Replaying the same file does nothing.
	apply(m, now)
	if m.slide != 1 {
		t.Errorf("replayed: slide %d", m.slide)
	}

	sendCmds(t, m, notes.Command{Seq: seq + 2, Target: pid, Op: notes.OpLast})
	apply(m, now)
	if m.slide != 2 {
		t.Errorf("last: slide %d", m.slide)
	}
	sendCmds(t, m, notes.Command{Seq: seq + 3, Target: pid, Op: notes.OpGoto, Slide: 2})
	apply(m, now)
	if m.slide != 1 {
		t.Errorf("goto 2: slide %d", m.slide)
	}
	sendCmds(t, m, notes.Command{Seq: seq + 4, Target: pid, Op: notes.OpGoto, Slide: 99})
	apply(m, now)
	if m.slide != 2 {
		t.Errorf("goto 99 clamps: slide %d", m.slide)
	}
	sendCmds(t, m, notes.Command{Seq: seq + 5, Target: pid, Op: notes.OpFirst}, notes.Command{Seq: seq + 6, Target: pid, Op: notes.OpPrev})
	apply(m, now)
	if m.slide != 0 || m.step != 0 {
		t.Errorf("first then prev at start: %d/%d", m.slide, m.step)
	}
}

func TestRemoteIgnores(t *testing.T) {
	m := remoteModel(t)
	now := time.Now()
	pid := os.Getpid()
	sendCmds(t, m,
		notes.Command{Seq: 5, Target: pid, Op: notes.OpLast},                                // from before this presenter started
		notes.Command{Seq: now.UnixNano() + 1, Target: pid + 1, Op: notes.OpLast},           // another presenter
		notes.Command{Seq: now.Add(-time.Minute).UnixNano(), Target: pid, Op: notes.OpLast}, // stale (but also pre-start)
	)
	apply(m, now)
	if m.slide != 0 {
		t.Errorf("ignored commands moved to slide %d", m.slide)
	}

	// Too old: newer than the presenter's start but past the max age.
	later := now.Add(remoteMaxAge + time.Second)
	sendCmds(t, m, notes.Command{Seq: now.UnixNano() + 10, Target: pid, Op: notes.OpLast})
	apply(m, later)
	if m.slide != 0 {
		t.Errorf("expired command applied: slide %d", m.slide)
	}
	if m.remote.applied != now.UnixNano()+10 && m.remote.applied < now.UnixNano() {
		t.Errorf("expired command not marked applied: %d", m.remote.applied)
	}
}

func TestNewRemoteClearsLeftovers(t *testing.T) {
	dir := t.TempDir()
	sp := filepath.Join(dir, "state.json")
	if err := notes.WriteCommands(notes.CommandPath(sp), []notes.Command{{Seq: 1, Op: notes.OpNext}}); err != nil {
		t.Fatal(err)
	}
	r := newRemote(sp, time.Now())
	if _, err := os.Stat(r.path); !os.IsNotExist(err) {
		t.Error("leftover command file not removed")
	}
}

func TestNotesSendAndAck(t *testing.T) {
	d, err := deck.Load(writeDeck(t))
	if err != nil {
		t.Fatal(err)
	}
	nm := &notesModel{statePath: filepath.Join(t.TempDir(), "state.json"), deck: d}
	cp := notes.CommandPath(nm.statePath)

	// No presenter: nothing is written, the view stays read-only.
	nm.send(notes.Command{Op: notes.OpNext})
	if _, err := os.Stat(cp); !os.IsNotExist(err) || len(nm.pending) != 0 {
		t.Fatal("command sent with no presenter")
	}

	nm.running, nm.state = true, notes.State{PID: os.Getpid()}
	nm.send(notes.Command{Op: notes.OpNext})
	nm.send(notes.Command{Op: notes.OpNext})
	got := notes.ReadCommands(cp)
	if len(got) != 2 || got[0].Seq >= got[1].Seq || got[0].Target != os.Getpid() {
		t.Fatalf("written %v", got)
	}

	// Ack of the first keeps the second.
	nm.state.Ack = got[0].Seq
	nm.syncCommands()
	if rest := notes.ReadCommands(cp); len(rest) != 1 || rest[0].Seq != got[1].Seq {
		t.Fatalf("after partial ack: %v", rest)
	}
	// Full ack removes the file.
	nm.state.Ack = got[1].Seq
	nm.syncCommands()
	if _, err := os.Stat(cp); !os.IsNotExist(err) {
		t.Error("command file left after full ack")
	}

	// Presenter vanishes with a command outstanding: it is dropped and cleaned.
	nm.send(notes.Command{Op: notes.OpLast})
	nm.running = false
	nm.syncCommands()
	if _, err := os.Stat(cp); !os.IsNotExist(err) || len(nm.pending) != 0 {
		t.Error("pending command survived the presenter")
	}
}

func writeDeck(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "d.md")
	writeFile(t, p, threeSlides, time.Unix(1_700_000_000, 0))
	return p
}
