package main

import (
	"context"
	"flag"
	"fmt"
	"image/color"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"tuipresent/internal/deck"
	"tuipresent/internal/img"
	"tuipresent/internal/notes"
	"tuipresent/internal/render"
	"tuipresent/internal/report"
	"tuipresent/internal/transition"
)

const (
	transitionDuration = 380 * time.Millisecond
	fadeDuration       = 280 * time.Millisecond
	frameInterval      = time.Second / 60
)

var (
	execTimeout = 15 * time.Second
	// execWaitDelay bounds how long a run waits for its output pipe after
	// the command exits or times out, so a background child that keeps the
	// pipe open can't hold the run past the timeout.
	execWaitDelay = 500 * time.Millisecond
)

func main() {
	dump := flag.Bool("dump", false, "print every frame to stdout and exit")
	runExec := flag.Bool("exec", false, "with -check or -dump: run +exec blocks first so their output is measured and shown")
	check := flag.Bool("check", false, "report slides that overflow a -w x -h screen and exit non-zero if any do")
	dw := flag.Int("w", 100, "width for -dump")
	dh := flag.Int("h", 30, "height for -dump")
	start := flag.Int("slide", 1, "slide to start on (1-based)")
	trans := flag.String("transition", "", "default transition: slide|wipe|cover|dissolve|none")
	watch := flag.Bool("watch", true, "reload the deck automatically when the file changes")
	notesView := flag.Bool("notes", false, "show speaker notes for a presenter running the same deck in another terminal")
	fade := flag.Bool("fade", true, "fade in content revealed by a pause; -fade=false makes reveals instant")
	images := flag.String("images", "auto", "image drawing: auto|kitty|blocks|off (-dump and -check always use blocks unless off)")
	themeName := flag.String("theme", "", "theme name or theme file (.yaml), overriding front matter; see -themes")
	themeBg := flag.String("background", "", "terminal|paint: paint the theme background into every cell, overriding front matter")
	listThemes := flag.Bool("themes", false, "list the built-in themes with a color swatch and exit")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: tuipresent [flags] deck.md\n       tuipresent -themes\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	if *listThemes {
		printThemes(os.Stdout)
		return
	}
	switch *themeBg {
	case "", "terminal", "paint":
	default:
		fmt.Fprintf(os.Stderr, "-background %q: want terminal or paint\n", *themeBg)
		os.Exit(2)
	}
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}
	imgMode, err := img.ParseMode(*images)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if imgMode == img.Off {
		render.Images.Mode = img.Off
	}
	path := flag.Arg(0)
	d, err := deck.Load(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	flags := deckFlags{transition: *trans, theme: *themeName, background: *themeBg}
	flags.apply(d)

	r := render.New(d)
	if err := r.ThemeErr(); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
		os.Exit(1)
	}

	if *notesView {
		if err := runNotes(path, d, flags); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if *check || *dump {
		report.PrintWarnings(os.Stderr, path, report.Warnings(d, r, flags.transition))
	}
	if *runExec && (*check || *dump) {
		var blocks []deck.Block
		for _, s := range d.Slides {
			collectExec(s.Blocks, 1<<30, &blocks)
		}
		for _, b := range blocks {
			r.SetExecOutput(b.ExecID, runBlock(b, d.Dir))
		}
	}

	if *check {
		if report.Check(os.Stdout, path, d, r, *dw, *dh) > 0 {
			os.Exit(1)
		}
		return
	}

	if *dump {
		report.Dump(os.Stdout, d, r, *dw, *dh)
		return
	}

	if imgMode == img.Auto {
		imgMode = img.Detect(os.Getenv)
	}
	if imgMode == img.Kitty {
		render.Images = render.ImageConfig{Mode: img.Kitty, Kitty: img.NewTransmitter()}
	}

	m := newModel(path, d, flags)
	m.watch, m.noFade = *watch, !*fade
	m.statePath, m.started = notes.StatePath(path), time.Now()
	m.slide = min(max(*start-1, 0), len(d.Slides)-1)
	m.publish()
	_, err = tea.NewProgram(m).Run()
	notes.Remove(m.statePath)
	if k := render.Images.Kitty; k != nil {
		fmt.Print(k.Cleanup())
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// newModel is the presenter for deck d loaded from path, with flags already
// applied. Its status line starts with the deck's warnings, if any.
func newModel(path string, d *deck.Deck, flags deckFlags) *model {
	m := &model{path: path, deck: d, r: render.New(d), flags: flags, stamp: stat(path)}
	m.themePath, m.themeStamp = themeFile(d), stat(themeFile(d))
	m.status = strings.Join(report.Warnings(d, m.r, flags.transition), "; ")
	return m
}

type tickMsg time.Time

type execMsg struct {
	id  int
	out string
}

type anim struct {
	from, to string
	kind     transition.Kind
	reverse  bool
	fade     bool // a pause reveal: to's new cells fade in over from
	start    time.Time
}

func (a *anim) duration() time.Duration {
	if a.fade {
		return fadeDuration
	}
	return transitionDuration
}

type model struct {
	path        string
	deck        *deck.Deck
	r           *render.Renderer
	w, h        int
	slide, step int
	anim        *anim
	status      string
	// flags are reapplied on reload so -transition, -theme and -background beat front matter.
	flags deckFlags
	// watch polls the deck file (and its custom theme file, if any) and
	// reloads on change; the stamps are their last seen state.
	watch      bool
	stamp      os.FileInfo
	themePath  string
	themeStamp os.FileInfo
	// Speaker-notes sync (see notes.go).
	statePath string
	started   time.Time
	published *notes.State
	noFade    bool        // -fade=false
	termBg    color.Color // terminal background (OSC 11), nil until reported
}

func (m *model) Init() tea.Cmd {
	cmds := []tea.Cmd{tea.RequestBackgroundColor, m.clearStatusLater(statusWarnTime)}
	if m.watch {
		cmds = append(cmds, watchTick())
	}
	return tea.Batch(cmds...)
}

const statusWarnTime = 6 * time.Second

// clearStatusLater clears the current status message after d unless it has
// been replaced by then.
func (m *model) clearStatusLater(d time.Duration) tea.Cmd {
	if m.status == "" {
		return nil
	}
	text := m.status
	return tea.Tick(d, func(time.Time) tea.Msg { return clearStatusMsg{text} })
}

func (m *model) Update(msg tea.Msg) (_ tea.Model, cmd tea.Cmd) {
	defer m.publish()
	defer func() { cmd = m.withImages(cmd) }()
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		m.anim = nil
	case tea.BackgroundColorMsg:
		m.termBg = msg.Color
	case tickMsg:
		if m.anim == nil {
			return m, nil
		}
		if time.Since(m.anim.start) >= m.anim.duration() {
			m.anim = nil
			return m, nil
		}
		return m, tick()
	case execMsg:
		m.r.SetExecOutput(msg.id, msg.out)
		m.status = ""
	case watchMsg:
		return m, m.pollWatch()
	case clearStatusMsg:
		if m.status == msg.text {
			m.status = ""
		}
	case tea.KeyPressMsg:
		return m, m.key(msg.String())
	}
	return m, nil
}

// withImages uploads kitty images for every slide at the current size so
// they are in the terminal before their placeholder cells are shown.
func (m *model) withImages(cmd tea.Cmd) tea.Cmd {
	k := render.Images.Kitty
	if k == nil || m.w == 0 {
		return cmd
	}
	for i, s := range m.deck.Slides {
		m.r.Frame(i, s.Steps-1, m.w, m.h)
	}
	if seq := k.Flush(); seq != "" {
		return tea.Batch(tea.Raw(seq), cmd)
	}
	return cmd
}

func (m *model) key(k string) tea.Cmd {
	switch k {
	case "q", "ctrl+c", "esc":
		return tea.Quit
	case "right", "l", "space", "n", "pgdown", "enter", "j", "down":
		return m.forward()
	case "left", "h", "p", "pgup", "backspace", "k", "up":
		return m.back()
	case "g", "home":
		return m.jump(0, 0)
	case "G", "end":
		return m.jump(len(m.deck.Slides)-1, 0)
	case "e":
		return m.runExec()
	case "r":
		return m.reload()
	}
	return nil
}

func (m *model) forward() tea.Cmd {
	s := m.deck.Slides[m.slide]
	if m.step < s.Steps-1 {
		m.step++
		return m.reveal()
	}
	if m.slide == len(m.deck.Slides)-1 {
		return nil
	}
	return m.go_(m.slide+1, 0, false)
}

func (m *model) back() tea.Cmd {
	if m.step > 0 {
		m.step--
		m.anim = nil
		return nil
	}
	if m.slide == 0 {
		return nil
	}
	prev := m.slide - 1
	return m.go_(prev, m.deck.Slides[prev].Steps-1, true)
}

func (m *model) jump(slide, step int) tea.Cmd {
	if slide == m.slide {
		m.step = step
		m.anim = nil
		return nil
	}
	return m.go_(slide, step, slide < m.slide)
}

// transitionInto is slide i's effective transition: its own directive, else
// the deck default (front matter, or -transition), else slide.
func transitionInto(d *deck.Deck, i int) transition.Kind {
	if t := d.Slides[i].Transition; t != "" {
		return transition.Parse(t)
	}
	return transition.Parse(d.Meta.Transition)
}

func (m *model) go_(slide, step int, reverse bool) tea.Cmd {
	// Going back undoes the transition that brought us to the current slide,
	// so it plays the current slide's transition in reverse.
	kind := transitionInto(m.deck, slide)
	if reverse {
		kind = transitionInto(m.deck, m.slide)
	}
	var from string
	if m.w > 0 {
		from = m.current()
	}
	m.slide, m.step = slide, step
	if kind == transition.None || m.w == 0 {
		m.anim = nil
		return nil
	}
	m.anim = &anim{from: from, to: m.r.Frame(m.slide, m.step, m.w, m.h), kind: kind, reverse: reverse, start: time.Now()}
	return tick()
}

// reveal fades in what the step just advanced to added; earlier steps stay put.
func (m *model) reveal() tea.Cmd {
	if m.noFade || m.w == 0 || !m.deck.FadeEnabled(m.slide) {
		m.anim = nil
		return nil
	}
	from := m.r.Frame(m.slide, m.step-1, m.w, m.h)
	m.anim = &anim{from: from, to: m.r.Frame(m.slide, m.step, m.w, m.h), fade: true, start: time.Now()}
	return tick()
}

func (m *model) current() string {
	if m.anim != nil {
		t := float64(time.Since(m.anim.start)) / float64(m.anim.duration())
		if m.anim.fade {
			// A painted slide fades up from its own background; otherwise
			// from the terminal's, or the theme's when that is unknown.
			bg := m.termBg
			if bg == nil || m.r.Paints() {
				bg = m.r.Background()
			}
			return transition.FadeIn(m.anim.from, m.anim.to, transition.EaseOutCubic(min(t, 1)), bg, m.r.Foreground())
		}
		return transition.Composite(m.anim.kind, m.anim.from, m.anim.to, m.w, transition.EaseOutCubic(min(t, 1)), m.anim.reverse)
	}
	return m.r.Frame(m.slide, m.step, m.w, m.h)
}

func (m *model) runExec() tea.Cmd {
	s := m.deck.Slides[m.slide]
	var blocks []deck.Block
	collectExec(s.Blocks, m.step, &blocks)
	if len(blocks) == 0 {
		return nil
	}
	m.status = "running…"
	var cmds []tea.Cmd
	for _, b := range blocks {
		cmds = append(cmds, execCmd(b, m.deck.Dir))
	}
	return tea.Batch(cmds...)
}

func collectExec(blocks []deck.Block, step int, out *[]deck.Block) {
	for _, b := range blocks {
		if b.Step > step {
			continue
		}
		switch b.Kind {
		case deck.Exec:
			*out = append(*out, b)
		case deck.Columns:
			for _, c := range b.Children {
				collectExec(c, step, out)
			}
		}
	}
}

func execCmd(b deck.Block, dir string) tea.Cmd {
	return func() tea.Msg { return execMsg{id: b.ExecID, out: runBlock(b, dir)} }
}

// runBlock runs an exec block in dir (the deck's directory, so relative
// paths in the code mean what they mean for images) and returns its combined
// output, with any error appended.
func runBlock(b deck.Block, dir string) string {
	ctx, cancel := context.WithTimeout(context.Background(), execTimeout)
	defer cancel()
	var c *exec.Cmd
	switch b.Lang {
	case "python", "python3", "py":
		c = exec.CommandContext(ctx, "python3", "-c", b.Text)
	case "sh":
		c = exec.CommandContext(ctx, "sh", "-c", b.Text)
	case "bash", "":
		c = exec.CommandContext(ctx, "bash", "-c", b.Text)
	default:
		return "[unsupported +exec language \"" + b.Lang + "\" (supported: bash, sh, python, python3, py)]"
	}
	c.Dir = dir
	c.WaitDelay = execWaitDelay
	out, err := c.CombinedOutput()
	s := string(out)
	if err != nil {
		s += "\n[" + err.Error() + "]"
	}
	return s
}

func (m *model) reload() tea.Cmd {
	m.stamp = stat(m.path)
	if m.themePath != "" {
		m.themeStamp = stat(m.themePath)
	}
	d, err := deck.Load(m.path)
	if err != nil {
		m.status = err.Error()
		return nil
	}
	m.flags.apply(d)
	r := render.New(d)
	m.themePath = themeFile(d)
	m.themeStamp = stat(m.themePath)
	if err := r.ThemeErr(); err != nil {
		// Like a parse error: keep showing the old deck until the theme is fixed.
		m.status = err.Error()
		return nil
	}
	m.deck, m.r, m.anim = d, r, nil
	m.slide = min(m.slide, len(d.Slides)-1)
	m.step = min(m.step, d.Slides[m.slide].Steps-1)
	if w := report.Warnings(d, r, m.flags.transition); len(w) > 0 {
		m.status = strings.Join(w, "; ")
		return m.clearStatusLater(statusWarnTime)
	}
	m.status = "reloaded"
	return m.clearStatusLater(1200 * time.Millisecond)
}

func (m *model) View() tea.View {
	var v tea.View
	if m.w == 0 {
		v = tea.NewView("")
	} else {
		frame := m.current()
		if m.status != "" && m.anim == nil {
			lines := strings.Split(frame, "\n")
			lines[len(lines)-1] = m.r.StatusLine(m.status, m.w)
			frame = strings.Join(lines, "\n")
		}
		v = tea.NewView(frame)
	}
	v.AltScreen = true
	return v
}

func tick() tea.Cmd {
	return tea.Tick(frameInterval, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// deckFlags are command-line settings that beat the deck's front matter.
// They are applied after every load, so they survive live reload.
type deckFlags struct {
	transition string
	theme      string // built-in name, or a .yaml/.yml theme file
	background string // terminal|paint
}

func (f deckFlags) apply(d *deck.Deck) {
	if f.transition != "" {
		d.Meta.Transition = f.transition
	}
	if f.theme != "" {
		// Previewing another theme replaces the deck's theme and its inline
		// overrides, but keeps its background choice.
		t := deck.Theme{Name: f.theme, Background: d.Meta.Theme.Background}
		if ext := strings.ToLower(filepath.Ext(f.theme)); ext == ".yaml" || ext == ".yml" {
			abs, err := filepath.Abs(f.theme)
			if err != nil {
				abs = f.theme
			}
			t = deck.Theme{Path: abs, Background: t.Background}
		}
		d.Meta.Theme = t
	}
	if f.background != "" {
		d.Meta.Theme.Background = f.background
	}
}

// themeFile is the deck's custom theme file, or "" if it uses a built-in.
func themeFile(d *deck.Deck) string {
	p := d.Meta.Theme.Path
	if p == "" || filepath.IsAbs(p) || d.Dir == "" {
		return p
	}
	return filepath.Join(d.Dir, p)
}

// printThemes lists the built-in themes, one per line with a color swatch.
func printThemes(w io.Writer) {
	names := render.ThemeNames()
	width := 0
	for _, n := range names {
		width = max(width, len(n))
	}
	for _, n := range names {
		t, _ := render.LookupTheme(n)
		kind := "dark "
		if t.IsLight() {
			kind = "light"
		}
		fmt.Fprintf(w, "%-*s  %s  %s\n", width, n, kind, render.Swatch(t))
	}
}
