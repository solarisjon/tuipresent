package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"tuipresent/internal/deck"
	"tuipresent/internal/img"
	"tuipresent/internal/render"
	"tuipresent/internal/transition"
)

const (
	transitionDuration = 380 * time.Millisecond
	frameInterval      = time.Second / 60
	execTimeout        = 15 * time.Second
)

func main() {
	dump := flag.Bool("dump", false, "print every frame to stdout and exit")
	runExec := flag.Bool("exec", false, "with -check or -dump: run +exec blocks first so their output is measured and shown")
	check := flag.Bool("check", false, "report slides that overflow a -w x -h screen and exit non-zero if any do")
	dw := flag.Int("w", 100, "width for -dump")
	dh := flag.Int("h", 30, "height for -dump")
	start := flag.Int("slide", 1, "slide to start on (1-based)")
	trans := flag.String("transition", "", "default transition: slide|wipe|cover|dissolve|none")
	images := flag.String("images", "auto", "image drawing: auto|kitty|blocks|off (-dump and -check always use blocks unless off)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: tuipresent [flags] deck.md\n")
		flag.PrintDefaults()
	}
	flag.Parse()
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
	if *trans != "" {
		d.Meta.Transition = *trans
	}

	r := render.New(d)
	if *runExec && (*check || *dump) {
		var blocks []deck.Block
		for _, s := range d.Slides {
			collectExec(s.Blocks, 1<<30, &blocks)
		}
		for _, b := range blocks {
			r.SetExecOutput(b.ExecID, runBlock(b))
		}
	}

	if *check {
		bad := 0
		for i := range d.Slides {
			rows, cols := r.Overflow(i, *dw, *dh)
			if rows > 0 || cols > 0 {
				bad++
				fmt.Printf("%s: slide %d overflows %dx%d by %d rows, %d cols\n", path, i+1, *dw, *dh, rows, cols)
			}
		}
		if bad > 0 {
			os.Exit(1)
		}
		fmt.Printf("%s: %d slides fit %dx%d\n", path, len(d.Slides), *dw, *dh)
		return
	}

	if *dump {
		for i, s := range d.Slides {
			for st := range s.Steps {
				fmt.Printf("=== slide %d step %d/%d ===\n%s\n", i+1, st+1, s.Steps, r.Frame(i, st, *dw, *dh))
			}
		}
		return
	}

	if imgMode == img.Auto {
		imgMode = img.Detect(os.Getenv)
	}
	if imgMode == img.Kitty {
		render.Images = render.ImageConfig{Mode: img.Kitty, Kitty: img.NewTransmitter()}
	}

	m := &model{path: path, deck: d, r: render.New(d), flagTransition: *trans}
	m.slide = min(max(*start-1, 0), len(d.Slides)-1)
	_, err = tea.NewProgram(m).Run()
	if k := render.Images.Kitty; k != nil {
		fmt.Print(k.Cleanup())
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
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
	start    time.Time
}

type model struct {
	path        string
	deck        *deck.Deck
	r           *render.Renderer
	w, h        int
	slide, step int
	anim        *anim
	status      string
	// flagTransition is reapplied on reload so -transition beats front matter.
	flagTransition string
}

func (m *model) Init() tea.Cmd { return nil }

func (m *model) Update(msg tea.Msg) (_ tea.Model, cmd tea.Cmd) {
	defer func() { cmd = m.withImages(cmd) }()
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		m.anim = nil
	case tickMsg:
		if m.anim == nil {
			return m, nil
		}
		if time.Since(m.anim.start) >= transitionDuration {
			m.anim = nil
			return m, nil
		}
		return m, tick()
	case execMsg:
		m.r.SetExecOutput(msg.id, msg.out)
		m.status = ""
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
		return nil
	}
	if m.slide == len(m.deck.Slides)-1 {
		return nil
	}
	return m.go_(m.slide+1, 0, false)
}

func (m *model) back() tea.Cmd {
	if m.step > 0 {
		m.step--
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
		return nil
	}
	return m.go_(slide, step, slide < m.slide)
}

func (m *model) go_(slide, step int, reverse bool) tea.Cmd {
	kind := transition.Parse(m.deck.Slides[slide].Transition)
	if m.deck.Slides[slide].Transition == "" {
		kind = transition.Parse(m.deck.Meta.Transition)
	}
	if reverse && m.deck.Slides[m.slide].Transition != "" {
		kind = transition.Parse(m.deck.Slides[m.slide].Transition)
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

func (m *model) current() string {
	if m.anim != nil {
		t := float64(time.Since(m.anim.start)) / float64(transitionDuration)
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
		cmds = append(cmds, execCmd(b))
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

func execCmd(b deck.Block) tea.Cmd {
	return func() tea.Msg { return execMsg{id: b.ExecID, out: runBlock(b)} }
}

func runBlock(b deck.Block) string {
	ctx, cancel := context.WithTimeout(context.Background(), execTimeout)
	defer cancel()
	var c *exec.Cmd
	switch b.Lang {
	case "python", "python3", "py":
		c = exec.CommandContext(ctx, "python3", "-c", b.Text)
	case "sh":
		c = exec.CommandContext(ctx, "sh", "-c", b.Text)
	default:
		c = exec.CommandContext(ctx, "bash", "-c", b.Text)
	}
	out, err := c.CombinedOutput()
	s := string(out)
	if err != nil {
		s += "\n[" + err.Error() + "]"
	}
	return s
}

func (m *model) reload() tea.Cmd {
	d, err := deck.Load(m.path)
	if err != nil {
		m.status = err.Error()
		return nil
	}
	if m.flagTransition != "" {
		d.Meta.Transition = m.flagTransition
	}
	m.deck, m.r, m.anim = d, render.New(d), nil
	m.slide = min(m.slide, len(d.Slides)-1)
	m.step = min(m.step, d.Slides[m.slide].Steps-1)
	m.status = "reloaded"
	return tea.Tick(1200*time.Millisecond, func(time.Time) tea.Msg { return execMsg{id: -1} })
}

func (m *model) View() tea.View {
	var v tea.View
	if m.w == 0 {
		v = tea.NewView("")
	} else {
		frame := m.current()
		if m.status != "" && m.anim == nil {
			lines := strings.Split(frame, "\n")
			lines[len(lines)-1] = render.Fit(" "+m.status, m.w)
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
