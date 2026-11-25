package main

import (
	"bytes"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/solarisjon/tuipresent/internal/deck"
	"github.com/solarisjon/tuipresent/internal/render"
)

var updateGolden = flag.Bool("update", false, "rewrite cmd/tuipresent/testdata golden files")

const exportDeck = "---\ntitle: Demo\ntheme: nord\n---\n\nIntro\n===\n\nHello there\n\n<!-- pause -->\n\nSecond step\n\n<!-- end_slide -->\n\n../../Etc/Passwd & Friends!\n===\n\n```sh +exec\necho hi\n```\n\n<!-- end_slide -->\n\nUntitled body only\n"

func loadExport(t *testing.T) (string, *deck.Deck, *render.Renderer) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "My Deck.md")
	if err := os.WriteFile(path, []byte(exportDeck), 0o644); err != nil {
		t.Fatal(err)
	}
	d, err := deck.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	return path, d, render.New(d)
}

func goldenCheck(t *testing.T, name, got string) {
	t.Helper()
	p := filepath.Join("testdata", name)
	if *updateGolden {
		if err := os.WriteFile(p, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(want) != got {
		t.Errorf("%s differs (run go test ./cmd/tuipresent -update to accept):\n%s", name, got)
	}
}

func TestBuildTapeGolden(t *testing.T) {
	_, d, r := loadExport(t)
	o := exportOpts{mode: "tape", w: 100, h: 30, deckPath: "talk.md", dwell: 1500 * time.Millisecond}
	got, err := buildTape(o, d, r, "talk.md", "talk.gif")
	if err != nil {
		t.Fatal(err)
	}
	goldenCheck(t, "export.tape.golden", got)

	o.exec = true
	got, _ = buildTape(o, d, r, "talk.md", "talk.gif")
	if strings.Count(got, "Type \"e\"") != 1 {
		t.Errorf("-exec should press e once, on the exec slide:\n%s", got)
	}
}

func TestTapeAutoDwellAndQuoting(t *testing.T) {
	if got := autoDwell("one two", 1); got != 2200*time.Millisecond {
		t.Errorf("autoDwell short = %v", got)
	}
	if got := autoDwell(strings.Repeat("w ", 500), 1); got != 10*time.Second {
		t.Errorf("autoDwell is capped at 10s, got %v", got)
	}
	if got := autoDwell("a", 5); got != time.Second {
		t.Errorf("autoDwell floors at 1s per step, got %v", got)
	}
	_, d, r := loadExport(t)
	tape, err := buildTape(exportOpts{w: 80, h: 24, mode: "tape"}, d, r, "-odd deck's.md", "x.gif")
	if err != nil || !strings.Contains(tape, `Type "tuipresent -watch=false -progress=false -mouse=false -images=blocks './-odd deck'\''s.md'"`) {
		t.Errorf("quoting: %v\n%s", err, tape)
	}
	if _, err := buildTape(exportOpts{mode: "tape"}, d, r, "a\nb.md", "x.gif"); err == nil {
		t.Error("newline in deck path accepted")
	}
}

func TestSlugAndSlideFiles(t *testing.T) {
	for in, want := range map[string]string{
		"Hello, World!":            "hello-world",
		"../../etc/passwd":         "etc-passwd",
		"日本語":                      "",
		strings.Repeat("abc ", 30): "abc-abc-abc-abc-abc-abc-abc-abc-abc-abc",
		"  --Mixed_Case 42--  ":    "mixed-case-42",
		"`code` & <b>bold</b>":     "code-b-bold-b",
	} {
		if got := slug(in, ""); got != want {
			t.Errorf("slug(%q) = %q, want %q", in, got, want)
		}
	}
	_, d, _ := loadExport(t)
	files, err := slideFiles(d, "png")
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, f := range files {
		names = append(names, f.img)
		if filepath.Base(f.img) != f.img || strings.Contains(f.img, "..") {
			t.Errorf("unsafe name %q", f.img)
		}
	}
	if len(files) != 4 || names[0] != "slide-01.png" || names[2] != "slide-03-etc-passwd-friends.png" || names[3] != "slide-04.png" {
		t.Errorf("names = %v", names)
	}
}

func TestNameValidation(t *testing.T) {
	for _, bad := range []string{"", ".", "..", "../x", "a/b", `a\b`, ".hidden", "a\x00b"} {
		if validName(bad) == nil {
			t.Errorf("validName(%q) accepted", bad)
		}
	}
	dir := t.TempDir()
	if err := checkTarget(dir, "ok.png"); err != nil {
		t.Error(err)
	}
	if err := os.Symlink("/etc/hosts", filepath.Join(dir, "link.png")); err == nil {
		if checkTarget(dir, "link.png") == nil {
			t.Error("symlink target accepted")
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "sub.png"), 0o755); err != nil {
		t.Fatal(err)
	}
	if checkTarget(dir, "sub.png") == nil {
		t.Error("directory target accepted")
	}
	if writeGuarded(dir, "../evil", []byte("x")) == nil {
		t.Error("traversal write accepted")
	}
}

func opts(t *testing.T, mode string) (exportOpts, *bytes.Buffer, *bytes.Buffer) {
	path, _, _ := loadExport(t)
	var so, se bytes.Buffer
	return exportOpts{mode: mode, w: 80, h: 24, deckPath: path, stdout: &so, stderr: &se,
		lookPath: func(string) (string, error) { return "", errors.New("not found") },
		run: func(dir, name string, args ...string) error {
			t.Errorf("tool run unexpectedly: %s %v", name, args)
			return nil
		}}, &so, &se
}

func TestFreezeDryRunTouchesNothing(t *testing.T) {
	_, d, r := loadExport(t)
	o, so, se := opts(t, "freeze")
	o.dryRun = true
	o.out = filepath.Join(t.TempDir(), "out")
	if err := runExport(o, d, r); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(o.out); err == nil {
		t.Error("dry run created the out dir")
	}
	if !strings.Contains(so.String(), "freeze --language ansi --background '#") || strings.Count(so.String(), "\nfreeze ") != 4 ||
		!strings.Contains(so.String(), "slide-03-etc-passwd-friends.png") {
		t.Errorf("stdout:\n%s", so)
	}
	if !strings.Contains(se.String(), "brew install charmbracelet/tap/freeze") {
		t.Errorf("stderr should carry the install hint:\n%s", se)
	}
}

func TestFreezeMissingToolHint(t *testing.T) {
	_, d, r := loadExport(t)
	o, _, _ := opts(t, "freeze")
	o.out = filepath.Join(t.TempDir(), "out")
	err := runExport(o, d, r)
	var ee *exitError
	if !errors.As(err, &ee) || ee.code != 1 || !strings.Contains(ee.msg, "go install github.com/charmbracelet/freeze@latest") {
		t.Fatalf("err = %v", err)
	}
	if _, err := os.Stat(o.out); err == nil {
		t.Error("out dir created before the tool check")
	}
}

func TestFreezeRunsPerSlide(t *testing.T) {
	_, d, r := loadExport(t)
	o, _, _ := opts(t, "freeze")
	o.out = filepath.Join(t.TempDir(), "out")
	o.lookPath = func(string) (string, error) { return "/bin/freeze", nil }
	var calls int
	o.run = func(dir, name string, args ...string) error {
		calls++
		if name != "freeze" || dir != o.out {
			t.Errorf("run(%q, %q)", dir, name)
		}
		return nil
	}
	if err := runExport(o, d, r); err != nil {
		t.Fatal(err)
	}
	if calls != 4 {
		t.Errorf("freeze runs = %d", calls)
	}
	b, err := os.ReadFile(filepath.Join(o.out, "slide-02-intro.ansi"))
	if err != nil || !strings.Contains(string(b), "\x1b[") || !strings.Contains(ansi.Strip(string(b)), "Second step") {
		t.Errorf("ansi source of the revealed slide: %v\n%q", err, b)
	}
}

func TestTapeModeOutput(t *testing.T) {
	_, d, r := loadExport(t)
	o, so, _ := opts(t, "tape")
	if err := runExport(o, d, r); err != nil || !strings.HasPrefix(so.String(), "# Generated by tuipresent -export tape") ||
		!strings.Contains(so.String(), "Output my-deck.gif") {
		t.Errorf("stdout tape: %v\n%s", err, so)
	}
	o.out = filepath.Join(t.TempDir(), "notes.txt")
	var ee *exitError
	if err := runExport(o, d, r); !errors.As(err, &ee) || ee.code != 2 {
		t.Errorf("non-.tape -out: %v", err)
	}
	o.out = filepath.Join(t.TempDir(), "x.tape")
	if err := runExport(o, d, r); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(o.out); !strings.Contains(string(b), "Sleep") {
		t.Error("tape not written")
	}
	o.format = "pdf"
	if err := runExport(o, d, r); !errors.As(err, &ee) || ee.code != 2 {
		t.Errorf("bad format: %v", err)
	}
}

func TestVHSModeDryRunAndMissingTool(t *testing.T) {
	_, d, r := loadExport(t)
	o, so, se := opts(t, "vhs")
	o.out = filepath.Join(t.TempDir(), "rec")
	o.dryRun = true
	if err := runExport(o, d, r); err != nil || !strings.Contains(so.String(), "Output my-deck.gif") ||
		!strings.Contains(se.String(), "vhs my-deck.tape") {
		t.Fatalf("dry run: %v\n%s\n%s", err, so, se)
	}
	if _, err := os.Stat(o.out); err == nil {
		t.Error("dry run created the out dir")
	}
	o.dryRun = false
	var ee *exitError
	if err := runExport(o, d, r); !errors.As(err, &ee) || !strings.Contains(ee.msg, "brew install vhs") {
		t.Errorf("missing vhs: %v", err)
	}
	if err := runExport(exportOpts{mode: "bogus", deckPath: "d.md", stdout: so, stderr: se}, d, r); !errors.As(err, &ee) || ee.code != 2 {
		t.Errorf("bad mode: %v", err)
	}
}
