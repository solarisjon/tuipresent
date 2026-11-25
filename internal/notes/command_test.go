package notes

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestCommandRoundTrip(t *testing.T) {
	path := CommandPath(filepath.Join(t.TempDir(), "tuipresent-ab.json"))
	if filepath.Base(path) != "tuipresent-ab.cmd.json" {
		t.Errorf("CommandPath = %s", path)
	}
	if got := ReadCommands(path); got != nil {
		t.Errorf("missing file: %v", got)
	}
	want := []Command{{Seq: 1, Target: 7, Op: OpNext}, {Seq: 2, Target: 7, Op: OpGoto, Slide: 4}}
	if err := WriteCommands(path, want); err != nil {
		t.Fatal(err)
	}
	if got := ReadCommands(path); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if m, _ := filepath.Glob(path + ".*"); len(m) != 0 {
		t.Errorf("temp files left: %v", m)
	}
}

func TestAckRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := Write(path, State{PID: 1, Ack: 42}); err != nil {
		t.Fatal(err)
	}
	if b := ReadCommands(path); b != nil {
		t.Errorf("state file is not a command list: %v", b)
	}
}

func TestPending(t *testing.T) {
	cmds := []Command{
		{Seq: 5, Target: 9, Op: OpNext},
		{Seq: 3, Target: 9, Op: OpPrev},  // out of order
		{Seq: 2, Target: 9, Op: OpFirst}, // already applied
		{Seq: 4, Target: 8, Op: OpLast},  // other presenter
		{Seq: 5, Target: 9, Op: OpLast},  // duplicate Seq
	}
	got := Pending(cmds, 9, 2)
	want := []Command{{Seq: 3, Target: 9, Op: OpPrev}, {Seq: 5, Target: 9, Op: OpNext}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if got := Pending(cmds, 9, 5); len(got) != 0 {
		t.Errorf("all applied, got %v", got)
	}
}

func TestReadCommandsCorrupt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "c.cmd.json")
	if err := writeAtomic(path, []byte("{not json")); err != nil {
		t.Fatal(err)
	}
	if got := ReadCommands(path); got != nil {
		t.Errorf("got %v", got)
	}
}

func TestKeys(t *testing.T) {
	var k Keys
	cases := []struct {
		key  string
		want Command
		ok   bool
		quit bool
	}{
		{"right", Command{Op: OpNext}, true, false},
		{"space", Command{Op: OpNext}, true, false},
		{"left", Command{Op: OpPrev}, true, false},
		{"backspace", Command{Op: OpPrev}, true, false},
		{"g", Command{Op: OpFirst}, true, false},
		{"G", Command{Op: OpLast}, true, false},
		{"q", Command{}, false, true},
		{"esc", Command{}, false, true},
		{"x", Command{}, false, false},
	}
	for _, c := range cases {
		got, ok, quit := k.Handle(c.key)
		if got != c.want || ok != c.ok || quit != c.quit {
			t.Errorf("%q: got %v %v %v", c.key, got, ok, quit)
		}
	}
}

func TestKeysNumber(t *testing.T) {
	var k Keys
	for _, d := range []string{"0", "1", "2"} { // leading 0 ignored
		if _, ok, _ := k.Handle(d); ok {
			t.Fatal("digit sent a command")
		}
	}
	if k.Typing() != "12" {
		t.Fatalf("typing %q", k.Typing())
	}
	k.Handle("backspace")
	if k.Typing() != "1" {
		t.Fatalf("after backspace %q", k.Typing())
	}
	k.Handle("3")
	c, ok, _ := k.Handle("enter")
	if !ok || c.Op != OpGoto || c.Slide != 13 || k.Typing() != "" {
		t.Errorf("enter: %v %v %q", c, ok, k.Typing())
	}
	k.Handle("5")
	if _, ok, quit := k.Handle("esc"); ok || quit || k.Typing() != "" {
		t.Error("esc while typing should only cancel")
	}
	k.Handle("5")
	if c, ok, _ := k.Handle("g"); !ok || c.Op != OpFirst || k.Typing() != "" {
		t.Errorf("other key cancels number and acts: %v %v", c, ok)
	}
	for range 9 {
		k.Handle("9")
	}
	if len(k.Typing()) != maxNumberDigits {
		t.Errorf("unbounded number %q", k.Typing())
	}
}
