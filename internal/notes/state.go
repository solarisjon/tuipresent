// Package notes syncs the presenter's position to a speaker-notes view through
// a small JSON state file in os.TempDir(), keyed by the deck's absolute path.
// The presenter rewrites it atomically on every slide or step change and removes
// it on quit; the notes view polls it. No network or TTY is involved.
package notes

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// State is the presenter's position as seen by the notes view.
type State struct {
	PID     int       `json:"pid"`
	Slide   int       `json:"slide"` // 0-based
	Step    int       `json:"step"`  // 0-based
	Started time.Time `json:"started"`
	Ack     int64     `json:"ack,omitempty"` // highest command Seq applied (see Command)
}

// StatePath returns the state file for deckPath. Relative paths and symlinks
// resolve to the same file, so both processes agree regardless of cwd.
func StatePath(deckPath string) string {
	p, err := filepath.Abs(deckPath)
	if err == nil {
		if r, err := filepath.EvalSymlinks(p); err == nil {
			p = r
		}
	} else {
		p = deckPath
	}
	sum := sha256.Sum256([]byte(p))
	return filepath.Join(os.TempDir(), "tuipresent-"+hex.EncodeToString(sum[:8])+".json")
}

// Write atomically replaces the state file at path.
func Write(path string, s State) error {
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return writeAtomic(path, b)
}

// writeAtomic replaces path with b via a temp file and rename.
func writeAtomic(path string, b []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return nil
}

// Read returns the presenter state at path. ok is false when no presenter is
// running: the file is missing or unreadable, or its process has exited.
func Read(path string) (s State, ok bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		return State{}, false
	}
	if json.Unmarshal(b, &s) != nil || !alive(s.PID) {
		return State{}, false
	}
	return s, true
}

// Remove deletes the state file; the notes view then shows "waiting".
func Remove(path string) error {
	err := os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

// alive reports whether pid is a running process, so a presenter that crashed
// without removing its file is not followed forever.
func alive(pid int) bool {
	if pid <= 0 {
		return false
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = p.Signal(syscall.Signal(0))
	return err == nil || errors.Is(err, syscall.EPERM)
}
