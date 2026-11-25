package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/solarisjon/tuipresent/docs"
)

func skillOptsFor(t *testing.T, target string) (skillOpts, *bytes.Buffer) {
	t.Helper()
	var out bytes.Buffer
	return skillOpts{target: target, cwd: t.TempDir(), home: t.TempDir(), stdout: &out}, &out
}

func TestInstallSkillTargets(t *testing.T) {
	for _, target := range []string{"project", "user", ""} {
		o, out := skillOptsFor(t, target)
		if err := installSkill(o); err != nil {
			t.Fatalf("%q: %v", target, err)
		}
		base := o.cwd
		if target == "user" {
			base = o.home
		}
		dir := filepath.Join(base, ".claude", "skills", "tuipresent-deck")
		for _, f := range docs.SkillFiles() {
			got, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(f.Path)))
			if err != nil || !bytes.Equal(got, f.Data) {
				t.Errorf("%q: %s: err %v, content differs", target, f.Path, err)
			}
			if !strings.Contains(out.String(), "wrote "+filepath.Join(dir, filepath.FromSlash(f.Path))) {
				t.Errorf("%q: output does not list %s", target, f.Path)
			}
		}
		if !strings.Contains(out.String(), "next:") {
			t.Errorf("%q: no next step printed", target)
		}
	}
}

func TestInstallSkillPath(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "my", "skill")
	o, _ := skillOptsFor(t, dir)
	if err := installSkill(o); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "SKILL.md")); err != nil {
		t.Error(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "references", "LANGUAGE.md")); err != nil {
		t.Error(err)
	}
}

func TestInstallSkillNoOverwrite(t *testing.T) {
	dir := t.TempDir()
	o, _ := skillOptsFor(t, dir)
	mine := filepath.Join(dir, "references", "GUIDE.md")
	if err := os.MkdirAll(filepath.Dir(mine), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mine, []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := installSkill(o)
	if err == nil || !strings.Contains(err.Error(), "-force") {
		t.Fatalf("want a -force error, got %v", err)
	}
	if b, _ := os.ReadFile(mine); string(b) != "mine" {
		t.Error("file was overwritten")
	}
	if _, err := os.Stat(filepath.Join(dir, "SKILL.md")); err == nil {
		t.Error("partial install: SKILL.md written although another file blocked it")
	}
	o.force = true
	if err := installSkill(o); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(mine); string(b) == "mine" {
		t.Error("-force did not overwrite")
	}
}

func TestInstallSkillRefusesSymlinks(t *testing.T) {
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret")
	if err := os.WriteFile(secret, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	cases := map[string]func(dir string){
		"target dir":     func(dir string) { os.Symlink(outside, dir) },
		"references dir": func(dir string) { os.MkdirAll(dir, 0o755); os.Symlink(outside, filepath.Join(dir, "references")) },
		"file":           func(dir string) { os.MkdirAll(dir, 0o755); os.Symlink(secret, filepath.Join(dir, "SKILL.md")) },
	}
	for name, setup := range cases {
		dir := filepath.Join(t.TempDir(), "skill")
		setup(dir)
		o, _ := skillOptsFor(t, dir)
		o.force = true // even -force must not follow a symlink
		if err := installSkill(o); err == nil {
			t.Errorf("%s: symlink was accepted", name)
		}
		if b, _ := os.ReadFile(secret); string(b) != "keep" {
			t.Errorf("%s: wrote through the symlink", name)
		}
		if entries, _ := os.ReadDir(outside); len(entries) != 1 {
			t.Errorf("%s: created files outside the target", name)
		}
	}
}

func TestRunSkillCmd(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := runSkillCmd(false, true, "project", false, &out, &errOut); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if out.String() != docs.SkillMD() || !strings.HasPrefix(out.String(), "---\nname: tuipresent-deck\n") {
		t.Error("-print-skill output is not the installed SKILL.md")
	}
	if strings.Contains(out.String(), "docs/agent") {
		t.Error("printed skill has repo-relative doc paths")
	}
	out.Reset()
	if code := runSkillCmd(true, true, "project", false, &out, &errOut); code != 2 {
		t.Errorf("both flags: exit %d, want 2", code)
	}
	dir := t.TempDir()
	if code := runSkillCmd(true, false, dir, false, &out, &errOut); code != 0 {
		t.Errorf("install: exit %d: %s", code, errOut.String())
	}
	if code := runSkillCmd(true, false, dir, false, &out, &errOut); code != 1 {
		t.Errorf("second install: exit %d, want 1", code)
	}
}
