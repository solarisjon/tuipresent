package notes

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
)

// Operation names a navigation command sent from the notes view.
type Operation string

const (
	OpNext  Operation = "next"  // next step, else next slide
	OpPrev  Operation = "prev"  // previous step, else previous slide
	OpFirst Operation = "first" // first slide
	OpLast  Operation = "last"  // last slide
	OpGoto  Operation = "goto"  // slide N (1-based), clamped by the presenter
)

// Command asks the presenter to navigate. Seq increases with every command a
// notes view issues; the presenter applies each Seq at most once and reports
// the highest it applied as State.Ack, which is how the notes view knows to
// stop resending it. Target is the presenter's PID, so a command written for
// one presenter is never applied by another.
type Command struct {
	Seq    int64     `json:"seq"`
	Target int       `json:"target"`
	Op     Operation `json:"op"`
	Slide  int       `json:"slide,omitempty"` // OpGoto only, 1-based
}

// CommandPath returns the command file that goes with a state file.
func CommandPath(statePath string) string {
	return strings.TrimSuffix(statePath, ".json") + ".cmd.json"
}

// WriteCommands atomically replaces the command file with cmds.
func WriteCommands(path string, cmds []Command) error {
	b, err := json.Marshal(cmds)
	if err != nil {
		return err
	}
	return writeAtomic(path, b)
}

// ReadCommands returns the commands in the file at path; a missing or corrupt
// file yields none.
func ReadCommands(path string) []Command {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var cmds []Command
	if json.Unmarshal(b, &cmds) != nil {
		return nil
	}
	return cmds
}

// Pending returns the commands addressed to pid with Seq greater than applied,
// in Seq order. Stale, duplicate and foreign commands are dropped.
func Pending(cmds []Command, pid int, applied int64) []Command {
	var out []Command
	for _, c := range cmds {
		if c.Target != pid || c.Seq <= applied {
			continue
		}
		i := len(out)
		for i > 0 && out[i-1].Seq > c.Seq {
			i--
		}
		if i > 0 && out[i-1].Seq == c.Seq {
			continue
		}
		out = append(out, Command{})
		copy(out[i+1:], out[i:])
		out[i] = c
	}
	return out
}

// maxNumberDigits bounds a typed slide number.
const maxNumberDigits = 5

// Keys turns notes-view key presses into commands, with the same bindings as
// the presenter. It also tracks a slide number being typed.
type Keys struct {
	num string
}

// Typing returns the slide number typed so far, or "".
func (k *Keys) Typing() string { return k.num }

// Handle processes key. It returns the command to send (ok; only Op and Slide
// are set) and whether the notes view should quit.
func (k *Keys) Handle(key string) (c Command, ok, quit bool) {
	if len(key) == 1 && key[0] >= '0' && key[0] <= '9' {
		if (key != "0" || k.num != "") && len(k.num) < maxNumberDigits {
			k.num += key
		}
		return Command{}, false, false
	}
	if k.num != "" {
		num := k.num
		k.num = ""
		switch key {
		case "enter":
			n, _ := strconv.Atoi(num)
			return Command{Op: OpGoto, Slide: max(n, 1)}, true, false
		case "esc":
			return Command{}, false, false
		case "backspace":
			k.num = num[:len(num)-1]
			return Command{}, false, false
		}
	}
	switch key {
	case "q", "ctrl+c", "esc":
		return Command{}, false, true
	case "right", "l", "space", "n", "pgdown", "enter", "j", "down":
		return Command{Op: OpNext}, true, false
	case "left", "h", "p", "pgup", "backspace", "k", "up":
		return Command{Op: OpPrev}, true, false
	case "g", "home":
		return Command{Op: OpFirst}, true, false
	case "G", "end":
		return Command{Op: OpLast}, true, false
	}
	return Command{}, false, false
}
