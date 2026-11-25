package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/solarisjon/tuipresent/docs"
)

// skillOpts are the -install-skill settings.
type skillOpts struct {
	target string // project, user or a directory
	force  bool
	cwd    string // base of "project"
	home   string // base of "user"
	stdout io.Writer
}

// skillDir is the directory the skill is installed into.
func skillDir(o skillOpts) (string, error) {
	switch o.target {
	case "", "project":
		if o.cwd == "" {
			return "", errors.New("-skill-dir project: cannot find the current directory")
		}
		return filepath.Join(o.cwd, ".claude", "skills", docs.SkillName), nil
	case "user":
		if o.home == "" {
			return "", errors.New("-skill-dir user: cannot find the home directory")
		}
		return filepath.Join(o.home, ".claude", "skills", docs.SkillName), nil
	}
	return filepath.Abs(o.target)
}

// installSkill writes the skill's files into the target directory and
// reports them. It writes nothing unless every file can be written: it
// refuses symlinks (the target, its references directory or any file in it)
// and existing files, unless force is set.
func installSkill(o skillOpts) error {
	dir, err := skillDir(o)
	if err != nil {
		return err
	}
	files := docs.SkillFiles()
	if err := checkSkillTarget(dir, files, o.force); err != nil {
		return err
	}
	for _, f := range files {
		dest := filepath.Join(dir, filepath.FromSlash(f.Path))
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		if err := writeNew(dest, f.Data, o.force); err != nil {
			return err
		}
		fmt.Fprintf(o.stdout, "wrote %s\n", dest)
	}
	fmt.Fprintf(o.stdout, "installed the %s skill in %s\n", docs.SkillName, dir)
	switch o.target {
	case "", "project", "user":
		fmt.Fprintln(o.stdout, "next: start `claude` and ask \"make me a 10-slide deck about X\"; it uses `tuipresent` from PATH")
	default:
		fmt.Fprintln(o.stdout, "next: point your agent runtime at SKILL.md in that directory (its references/ hold the exact syntax)")
	}
	return nil
}

// checkSkillTarget fails before anything is written if the target is not
// safe: a symlinked or non-directory skill or references directory, a
// symlink or directory where a file goes, or (without force) any file that
// exists.
func checkSkillTarget(dir string, files []docs.File, force bool) error {
	if err := checkDir(dir); err != nil {
		return err
	}
	if err := checkDir(filepath.Join(dir, "references")); err != nil {
		return err
	}
	var exist []string
	for _, f := range files {
		if strings.Contains(f.Path, "..") || filepath.IsAbs(f.Path) {
			return fmt.Errorf("internal error: unsafe skill path %q", f.Path)
		}
		dest := filepath.Join(dir, filepath.FromSlash(f.Path))
		fi, err := os.Lstat(dest)
		switch {
		case errors.Is(err, os.ErrNotExist):
		case err != nil:
			return err
		case !fi.Mode().IsRegular():
			return fmt.Errorf("%s exists and is not a regular file; not touching it", dest)
		default:
			exist = append(exist, dest)
		}
	}
	if len(exist) > 0 && !force {
		return fmt.Errorf("not overwriting existing files (use -force to replace them):\n  %s", strings.Join(exist, "\n  "))
	}
	return nil
}

// checkDir allows a directory that is missing or a real directory.
func checkDir(p string) error {
	fi, err := os.Lstat(p)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return nil
	case err != nil:
		return err
	case fi.Mode()&os.ModeSymlink != 0:
		return fmt.Errorf("%s is a symlink; refusing to write through it", p)
	case !fi.IsDir():
		return fmt.Errorf("%s exists and is not a directory", p)
	}
	return nil
}

// writeNew creates path with data. An existing regular file is removed first
// only with force; a symlink swapped in meanwhile makes the create fail.
func writeNew(path string, data []byte, force bool) error {
	if force {
		if fi, err := os.Lstat(path); err == nil && fi.Mode().IsRegular() {
			if err := os.Remove(path); err != nil {
				return err
			}
		}
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// runSkillCmd handles -install-skill and -print-skill; it returns the exit status.
func runSkillCmd(install, print bool, target string, force bool, stdout, stderr io.Writer) int {
	if install && print {
		fmt.Fprintln(stderr, "-install-skill and -print-skill cannot be combined")
		return 2
	}
	if print {
		io.WriteString(stdout, docs.SkillMD())
		return 0
	}
	cwd, _ := os.Getwd()
	home, _ := os.UserHomeDir()
	if err := installSkill(skillOpts{target: target, force: force, cwd: cwd, home: home, stdout: stdout}); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
