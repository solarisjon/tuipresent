package main

import (
	"testing"

	"github.com/solarisjon/tuipresent/internal/deck"
	"github.com/solarisjon/tuipresent/internal/render"
)

func bigModel(t *testing.T) *model {
	t.Helper()
	d, err := deck.Parse("Big\n===\n<!-- title_size: 2 -->\nbody\n<!-- pause -->\nmore\n<!-- end_slide -->\nSmall\n===\nbody\n")
	if err != nil {
		t.Fatal(err)
	}
	m := newModel("deck.md", d, deckFlags{})
	m.w, m.h = 100, 30
	return m
}

func TestBigTitleOverlayLifecycle(t *testing.T) {
	defer func(v bool) { render.TextSizing = v }(render.TextSizing)
	m := bigModel(t)

	render.TextSizing = false
	if m.withBigTitle(nil, nil) != nil || m.big.pending != "" {
		t.Fatal("overlay scheduled without text sizing")
	}

	render.TextSizing = true
	if m.withBigTitle(nil, nil) == nil || m.big.pending == "" {
		t.Fatal("settled big title should schedule its overlay")
	}
	gen := m.big.gen
	m.withBigTitle(bigTitleMsg{gen - 1}, nil) // a stale tick does nothing
	if m.big.shown != "" {
		t.Fatal("stale tick drew the overlay")
	}
	m.withBigTitle(bigTitleMsg{gen}, nil)
	if m.big.shown == "" || m.big.pending != "" {
		t.Fatalf("overlay not drawn: %+v", m.big)
	}

	// A pause reveal fades the body only: the overlay stays.
	shown := m.big.shown
	m.step = 1
	m.anim = &anim{fade: true}
	m.withBigTitle(nil, nil)
	if m.big.shown != shown {
		t.Error("a reveal fade must keep the overlay")
	}

	// A slide transition blanks it at once; nothing is scheduled mid-animation.
	m.anim = &anim{}
	if m.withBigTitle(nil, nil) == nil || m.big.shown != "" || m.big.pending != "" {
		t.Errorf("transition should erase and not reschedule: %+v", m.big)
	}

	// A slide without a big title has none.
	m.anim, m.slide, m.step = nil, 1, 0
	if m.withBigTitle(nil, nil) != nil || m.big.pending != "" {
		t.Errorf("plain slide scheduled an overlay: %+v", m.big)
	}
}
