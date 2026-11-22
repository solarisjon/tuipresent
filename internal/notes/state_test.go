package notes

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWriteReadRemove(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if _, ok := Read(path); ok {
		t.Fatal("Read of missing file reported a presenter")
	}
	want := State{PID: os.Getpid(), Slide: 3, Step: 1, Started: time.Now().Round(time.Second)}
	if err := Write(path, want); err != nil {
		t.Fatal(err)
	}
	got, ok := Read(path)
	if !ok {
		t.Fatal("Read after Write: not ok")
	}
	if got.Slide != want.Slide || got.Step != want.Step || got.PID != want.PID || !got.Started.Equal(want.Started) {
		t.Errorf("got %+v, want %+v", got, want)
	}

	want.Slide, want.Step = 4, 0
	if err := Write(path, want); err != nil {
		t.Fatal(err)
	}
	if got, _ := Read(path); got.Slide != 4 || got.Step != 0 {
		t.Errorf("overwrite: got %+v", got)
	}
	if m, _ := filepath.Glob(path + ".*"); len(m) != 0 {
		t.Errorf("temp files left behind: %v", m)
	}

	if err := Remove(path); err != nil {
		t.Fatal(err)
	}
	if _, ok := Read(path); ok {
		t.Error("Read after Remove reported a presenter")
	}
	if err := Remove(path); err != nil {
		t.Errorf("Remove of missing file: %v", err)
	}
}

func TestReadIgnoresDeadOrCorrupt(t *testing.T) {
	dir := t.TempDir()
	dead := filepath.Join(dir, "dead.json")
	if err := Write(dead, State{PID: 0, Slide: 2}); err != nil {
		t.Fatal(err)
	}
	if _, ok := Read(dead); ok {
		t.Error("state with no live PID reported as running")
	}
	bad := filepath.Join(dir, "bad.json")
	os.WriteFile(bad, []byte("{not json"), 0o644)
	if _, ok := Read(bad); ok {
		t.Error("corrupt state reported as running")
	}
}

func TestStatePathStable(t *testing.T) {
	dir := t.TempDir()
	deck := filepath.Join(dir, "deck.md")
	os.WriteFile(deck, []byte("x"), 0o644)
	link := filepath.Join(dir, "link.md")
	if err := os.Symlink(deck, link); err != nil {
		t.Skip(err)
	}
	t.Chdir(dir)
	a, b, c := StatePath(deck), StatePath("deck.md"), StatePath(link)
	if a != b || a != c {
		t.Errorf("paths differ: %s %s %s", a, b, c)
	}
	if filepath.Dir(a) != filepath.Clean(os.TempDir()) {
		t.Errorf("state file %s not in %s", a, os.TempDir())
	}
	if StatePath(filepath.Join(dir, "other.md")) == a {
		t.Error("different decks share a state file")
	}
}
