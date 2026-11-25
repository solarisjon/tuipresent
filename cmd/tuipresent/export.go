package main

// Sharing and export: record a deck with Charm's VHS (-export tape|vhs) or
// render one image per slide with Charm's Freeze (-export freeze). Both tools
// are only shelled out to, never linked, and -dry-run needs neither.

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/x/ansi"

	"tuipresent/internal/deck"
	"tuipresent/internal/render"
)

const (
	vhsHint = "vhs not found on PATH. Install it (it also needs ttyd and ffmpeg):\n" +
		"  brew install vhs\n" +
		"  go install github.com/charmbracelet/vhs@latest\n" +
		"See https://github.com/charmbracelet/vhs#installation. Or run with -dry-run, or use -export tape and run vhs elsewhere."
	freezeHint = "freeze not found on PATH. Install it:\n" +
		"  brew install charmbracelet/tap/freeze\n" +
		"  go install github.com/charmbracelet/freeze@latest\n" +
		"See https://github.com/charmbracelet/freeze#installation. Or run with -dry-run to see what would run."

	// VHS geometry: a cell is vhsFontSize*6/10 wide and 1.2*vhsFontSize
	// tall, so the pixel size follows from the -w x -h screen.
	vhsFontSize = 16
	vhsPadding  = 20
)

// exportOpts is one -export run. lookPath and run are replaceable for tests.
type exportOpts struct {
	mode     string // tape | vhs | freeze
	out      string
	format   string
	dryRun   bool
	exec     bool          // tape/vhs: press e on slides with exec blocks
	dwell    time.Duration // per step; 0 = by amount of text
	w, h     int
	deckPath string // as given on the command line
	flags    deckFlags
	stdout   io.Writer
	stderr   io.Writer
	lookPath func(string) (string, error)
	run      func(dir, name string, args ...string) error
}

func (o *exportOpts) defaults() {
	if o.lookPath == nil {
		o.lookPath = exec.LookPath
	}
	if o.run == nil {
		o.run = func(dir, name string, args ...string) error {
			c := exec.Command(name, args...)
			c.Dir, c.Stdout, c.Stderr = dir, o.stderr, o.stderr
			return c.Run()
		}
	}
}

// exitError carries a process exit status for runExport's caller.
type exitError struct {
	code int
	msg  string
}

func (e *exitError) Error() string { return e.msg }

func usagef(format string, a ...any) error { return &exitError{2, fmt.Sprintf(format, a...)} }

// runExport performs the export and returns an *exitError (or nil).
func runExport(o exportOpts, d *deck.Deck, r *render.Renderer) error {
	o.defaults()
	base := slug(strings.TrimSuffix(filepath.Base(o.deckPath), filepath.Ext(o.deckPath)), "deck")
	switch o.mode {
	case "tape", "vhs":
		if o.format == "" {
			o.format = "gif"
		}
		switch o.format {
		case "gif", "mp4", "webm":
		default:
			return usagef("-format %q: want gif, mp4 or webm with -export %s", o.format, o.mode)
		}
		return exportVHS(o, d, r, base)
	case "freeze":
		if o.format == "" {
			o.format = "png"
		}
		switch o.format {
		case "png", "svg":
		default:
			return usagef("-format %q: want png or svg with -export freeze (freeze has no PDF output; see docs/guide/sharing.md to combine the PNGs)", o.format)
		}
		return exportFreeze(o, d, r, base)
	}
	return usagef("-export %q: want tape, vhs or freeze", o.mode)
}

func exportVHS(o exportOpts, d *deck.Deck, r *render.Renderer, base string) error {
	deckArg := o.deckPath
	var outDir, tapePath string
	switch {
	case o.mode == "vhs":
		// vhs runs inside the out dir, so the deck must be an absolute path.
		abs, err := filepath.Abs(o.deckPath)
		if err != nil {
			return &exitError{1, err.Error()}
		}
		deckArg = abs
		outDir = o.out
		if outDir == "" {
			outDir = base + "-recording"
		}
		tapePath = filepath.Join(outDir, base+".tape")
	case o.out != "":
		if !strings.EqualFold(filepath.Ext(o.out), ".tape") {
			return usagef("-out %q: a tape file must end in .tape (refusing to write anything else)", o.out)
		}
		tapePath = o.out
	}
	tape, err := buildTape(o, d, r, deckArg, base+"."+o.format)
	if err != nil {
		return &exitError{2, err.Error()}
	}
	if o.dryRun || tapePath == "" {
		fmt.Fprint(o.stdout, tape)
		if o.dryRun && o.mode == "vhs" {
			fmt.Fprintf(o.stderr, "dry run: would write %s and run: vhs %s.tape (in %s)\n", tapePath, base, outDir)
		}
		return nil
	}
	if o.mode == "vhs" {
		if _, err := o.lookPath("vhs"); err != nil {
			return &exitError{1, vhsHint}
		}
		if err := prepareDir(outDir); err != nil {
			return &exitError{1, err.Error()}
		}
	}
	if err := writeGuarded(filepath.Dir(tapePath), filepath.Base(tapePath), []byte(tape)); err != nil {
		return &exitError{1, err.Error()}
	}
	fmt.Fprintf(o.stderr, "wrote %s\n", tapePath)
	if o.mode == "tape" {
		fmt.Fprintf(o.stderr, "record it with: vhs %s\n", shellQuote(tapePath))
		return nil
	}
	if err := o.run(outDir, "vhs", base+".tape"); err != nil {
		return &exitError{1, "vhs failed: " + err.Error()}
	}
	fmt.Fprintf(o.stderr, "recorded %s\n", filepath.Join(outDir, base+"."+o.format))
	return nil
}

// buildTape generates the VHS tape: set up size, theme and font, start the
// presenter out of sight, then step through every slide and pause and quit.
func buildTape(o exportOpts, d *deck.Deck, r *render.Renderer, deckArg, output string) (string, error) {
	if strings.ContainsAny(deckArg+output, "\n\r") {
		return "", errors.New("deck path contains a newline")
	}
	cmd := []string{"tuipresent", "-watch=false", "-progress=false", "-mouse=false", "-images=blocks"}
	if o.flags.theme != "" {
		cmd = append(cmd, "-theme", o.flags.theme)
	}
	if o.flags.background != "" {
		cmd = append(cmd, "-background", o.flags.background)
	}
	if o.flags.transition != "" {
		cmd = append(cmd, "-transition", o.flags.transition)
	}
	if strings.HasPrefix(deckArg, "-") {
		deckArg = "./" + deckArg // not a flag
	}
	cmd = append(cmd, deckArg)
	typed, err := vhsString(quoteArgs(cmd))
	if err != nil {
		return "", err
	}
	pal := r.Theme().Palette
	var b strings.Builder
	fmt.Fprintf(&b, "# Generated by tuipresent -export %s for %s. Edit freely, then run: vhs <this file>\n", o.mode, filepath.Base(deckArg))
	fmt.Fprintf(&b, "# Screen %dx%d cells; expects tuipresent on PATH.\n", o.w, o.h)
	fmt.Fprintf(&b, "Output %s\n\n", output)
	fmt.Fprintf(&b, "Set FontSize %d\n", vhsFontSize)
	fmt.Fprintf(&b, "Set LineHeight 1.2\n")
	fmt.Fprintf(&b, "Set Padding %d\n", vhsPadding)
	fmt.Fprintf(&b, "Set Width %d\n", o.w*vhsFontSize*6/10+2*vhsPadding)
	fmt.Fprintf(&b, "Set Height %d\n", o.h*vhsFontSize*12/10+2*vhsPadding)
	fmt.Fprintf(&b, "Set Theme { \"name\": %q, \"background\": %q, \"foreground\": %q, \"cursor\": %q }\n\n",
		"tuipresent-"+r.Theme().Name, pal.Background, pal.Foreground, pal.Accent)
	fmt.Fprintf(&b, "Hide\nType %s\nEnter\nSleep 1s\nShow\n", typed)
	for i, s := range d.Slides {
		fmt.Fprintf(&b, "\n# slide %d", i+1)
		if s.Title != "" {
			fmt.Fprintf(&b, ": %s", strings.Join(strings.Fields(s.Title), " "))
		}
		b.WriteString("\n")
		per := o.dwell
		if per <= 0 {
			per = autoDwell(r.Frame(i, s.Steps-1, o.w, o.h), s.Steps)
		}
		for st := 0; st < s.Steps; st++ {
			fmt.Fprintf(&b, "Sleep %dms\n", per.Milliseconds())
			if st == s.Steps-1 && o.exec {
				var blocks []deck.Block
				collectExec(s.Blocks, s.Steps, &blocks)
				if len(blocks) > 0 {
					b.WriteString("Type \"e\"\nSleep 2s\n")
				}
			}
			if i < len(d.Slides)-1 || st < s.Steps-1 {
				b.WriteString("Right\n")
			}
		}
	}
	b.WriteString("\nType \"q\"\nSleep 500ms\n")
	return b.String(), nil
}

// autoDwell is the time to rest on each step of a slide whose fully revealed
// frame is frame: 2 s plus 100 ms per word, at most 10 s, shared between the
// steps but never under 1 s so a transition can finish.
func autoDwell(frame string, steps int) time.Duration {
	words := len(strings.Fields(ansi.Strip(frame)))
	total := min(2000+100*words, 10000)
	return time.Duration(max(total/max(steps, 1), 1000)/100*100) * time.Millisecond
}

func exportFreeze(o exportOpts, d *deck.Deck, r *render.Renderer, base string) error {
	outDir := o.out
	if outDir == "" {
		outDir = base + "-slides"
	}
	files, err := slideFiles(d, o.format)
	if err != nil {
		return &exitError{1, err.Error()}
	}
	for _, f := range files {
		for _, n := range []string{f.ansi, f.img} {
			if err := checkTarget(outDir, n); err != nil {
				return &exitError{1, err.Error()}
			}
		}
	}
	bg := r.Theme().Palette.Background
	args := func(f slideFile) []string {
		return []string{"--language", "ansi", "--background", bg, "--padding", "20",
			"--font.size", "14", "--line-height", "1.2", "--output", f.img, f.ansi}
	}
	if o.dryRun {
		fmt.Fprintf(o.stdout, "# dry run: nothing written. Would create %s and run, inside it:\n", outDir)
		for _, f := range files {
			fmt.Fprintf(o.stdout, "freeze %s\n", quoteArgs(args(f)))
		}
		if _, err := o.lookPath("freeze"); err != nil {
			fmt.Fprintf(o.stderr, "note: freeze is not installed; a real run would stop here.\n%s\n", freezeHint)
		}
		return nil
	}
	if _, err := o.lookPath("freeze"); err != nil {
		return &exitError{1, freezeHint}
	}
	if err := prepareDir(outDir); err != nil {
		return &exitError{1, err.Error()}
	}
	for i, f := range files {
		frame := r.Frame(i, d.Slides[i].Steps-1, o.w, o.h)
		if err := writeGuarded(outDir, f.ansi, []byte(frame+"\n")); err != nil {
			return &exitError{1, err.Error()}
		}
		if err := o.run(outDir, "freeze", args(f)...); err != nil {
			return &exitError{1, fmt.Sprintf("freeze failed on %s: %v", f.ansi, err)}
		}
	}
	fmt.Fprintf(o.stderr, "wrote %d %s images (and their .ansi sources) to %s\n", len(files), o.format, outDir)
	return nil
}

type slideFile struct{ ansi, img string }

// slideFiles names the .ansi source and image of every slide:
// slide-NN-title-slug.ext, with NN zero-padded to the deck's size.
func slideFiles(d *deck.Deck, format string) ([]slideFile, error) {
	width := max(2, len(fmt.Sprint(len(d.Slides))))
	var out []slideFile
	for i, s := range d.Slides {
		name := fmt.Sprintf("slide-%0*d", width, i+1)
		if sl := slug(s.Title, ""); sl != "" {
			name += "-" + sl
		}
		if err := validName(name + "." + format); err != nil {
			return nil, err
		}
		out = append(out, slideFile{name + ".ansi", name + "." + format})
	}
	return out, nil
}

var (
	nonSlugRe = regexp.MustCompile(`[^a-z0-9]+`)
	safeArgRe = regexp.MustCompile(`^[A-Za-z0-9_@%+=:,./-]+$`)
)

// slug lowercases s to [a-z0-9-], at most 40 characters, or returns def.
func slug(s, def string) string {
	s = strings.Trim(nonSlugRe.ReplaceAllString(strings.ToLower(s), "-"), "-")
	if len(s) > 40 {
		s = strings.Trim(s[:40], "-")
	}
	if s == "" {
		return def
	}
	return s
}

// validName rejects anything that is not a plain file name.
func validName(name string) error {
	if name == "" || strings.HasPrefix(name, ".") ||
		strings.ContainsAny(name, "/\\\x00") || filepath.Base(name) != name {
		return fmt.Errorf("refusing unsafe file name %q", name)
	}
	return nil
}

// checkTarget refuses a name that would land outside dir, or on a symlink or
// directory inside it.
func checkTarget(dir, name string) error {
	if err := validName(name); err != nil {
		return err
	}
	p := filepath.Join(dir, name)
	if rel, err := filepath.Rel(dir, p); err != nil || rel != name {
		return fmt.Errorf("refusing to write %q outside %s", name, dir)
	}
	if fi, err := os.Lstat(p); err == nil && !fi.Mode().IsRegular() {
		return fmt.Errorf("refusing to overwrite %s: not a regular file", p)
	}
	return nil
}

// prepareDir creates dir, refusing an existing non-directory.
func prepareDir(dir string) error {
	if fi, err := os.Stat(dir); err == nil && !fi.IsDir() {
		return fmt.Errorf("-out %s exists and is not a directory", dir)
	}
	return os.MkdirAll(dir, 0o755)
}

func writeGuarded(dir, name string, data []byte) error {
	if err := checkTarget(dir, name); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, name), data, 0o644)
}

// shellQuote quotes s for a POSIX shell only when it has to be.
func shellQuote(s string) string {
	if safeArgRe.MatchString(s) {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func quoteArgs(args []string) string {
	q := make([]string, len(args))
	for i, a := range args {
		q[i] = shellQuote(a)
	}
	return strings.Join(q, " ")
}

// vhsString is s as a VHS string literal, using a quote character s lacks.
func vhsString(s string) (string, error) {
	for _, q := range []string{`"`, `'`, "`"} {
		if !strings.Contains(s, q) {
			return q + s + q, nil
		}
	}
	return "", errors.New("deck path contains \", ' and ` and cannot be typed by VHS")
}
