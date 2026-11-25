package main

import (
	"strings"
	"testing"

	"tuipresent/internal/deck"
)

func env(kv ...string) func(string) string {
	m := map[string]string{}
	for i := 0; i+1 < len(kv); i += 2 {
		m[kv[i]] = kv[i+1]
	}
	return func(k string) string { return m[k] }
}

func TestLinksEnabled(t *testing.T) {
	for _, tc := range []struct {
		mode string
		env  func(string) string
		tty  bool
		want bool
	}{
		{"auto", env("TERM", "xterm-ghostty"), true, true},
		{"", env("TERM", "xterm-256color"), true, true},
		{"auto", env("TERM", "xterm-256color"), false, false},
		{"auto", env("TERM", "dumb"), true, false},
		{"auto", env("TERM", "xterm", "TERM_PROGRAM", "Apple_Terminal"), true, false},
		{"on", env(), false, true},
		{"off", env("TERM", "xterm"), true, false},
	} {
		got, err := linksEnabled(tc.mode, tc.env, tc.tty)
		if err != nil || got != tc.want {
			t.Errorf("linksEnabled(%q, tty=%v) = %v, %v; want %v", tc.mode, tc.tty, got, err, tc.want)
		}
	}
	if _, err := linksEnabled("maybe", env(), true); err == nil {
		t.Error("bad mode accepted")
	}
}

func TestProgressSeq(t *testing.T) {
	for _, tc := range []struct {
		slide, n int
		want     string
	}{
		{0, 4, "\x1b]9;4;1;25\x1b\\"},
		{3, 4, "\x1b]9;4;1;100\x1b\\"},
		{0, 1, "\x1b]9;4;1;100\x1b\\"},
		{0, 0, progressClear},
	} {
		if got := progressSeq(tc.slide, tc.n); got != tc.want {
			t.Errorf("progressSeq(%d, %d) = %q, want %q", tc.slide, tc.n, got, tc.want)
		}
	}
}

func progressModel(t *testing.T, front string) *model {
	t.Helper()
	d, err := deck.Parse(front + "One\n===\n\n<!-- end_slide -->\n\nTwo\n===\n")
	if err != nil {
		t.Fatal(err)
	}
	m := newModel("x.md", d, deckFlags{})
	m.progress = true
	m.w, m.h = 80, 24
	return m
}

func TestWithProgressOnlyOnChange(t *testing.T) {
	m := progressModel(t, "")
	if m.withProgress(nil) == nil {
		t.Fatal("first frame did not report progress")
	}
	if m.withProgress(nil) != nil {
		t.Error("progress repeated without a slide change")
	}
	m.slide = 1
	if m.withProgress(nil) == nil {
		t.Error("slide change did not report progress")
	}
}

func TestProgressOptOut(t *testing.T) {
	for _, front := range []string{"---\nprogress: off\n---\n", "---\nprogress: false\n---\n"} {
		m := progressModel(t, front)
		if m.withProgress(nil) != nil {
			t.Errorf("front matter %q still reports progress", strings.TrimSpace(front))
		}
	}
	m := progressModel(t, "")
	m.progress = false
	if m.withProgress(nil) != nil {
		t.Error("-progress=false still reports progress")
	}
	// Turning it off after it was shown (a reload) clears the bar.
	m = progressModel(t, "")
	m.withProgress(nil)
	m.deck.Meta.Progress = "off"
	if m.withProgress(nil) == nil {
		t.Error("switching progress off did not clear the bar")
	}
	if m.withProgress(nil) != nil {
		t.Error("cleared twice")
	}
}
