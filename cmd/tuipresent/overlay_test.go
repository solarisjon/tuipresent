package main

import (
	"strings"
	"testing"

	"github.com/solarisjon/tuipresent/internal/deck"
	"github.com/solarisjon/tuipresent/internal/img"
	"github.com/solarisjon/tuipresent/internal/render"
)

func overlayModel(t *testing.T) *model {
	t.Helper()
	d, err := deck.Parse("Pic\n===\n\n![logo](assets/terminal.png)\n<!-- end_slide -->\nPlain\n===\n\ntext\n")
	if err != nil {
		t.Fatal(err)
	}
	d.Dir = "../../examples"
	m := newModel("deck.md", d, deckFlags{})
	m.w, m.h = 100, 30
	return m
}

func TestOverlayLifecycle(t *testing.T) {
	old := render.Images
	defer func() { render.Images = old }()
	render.Images = render.ImageConfig{Mode: img.Sixel, Overlay: img.NewOverlay(img.Sixel)}
	m := overlayModel(t)
	m.r = render.New(m.deck) // after the config, so images load in this mode

	// The image slide schedules a paint and shows blank cells, not stand-ins.
	if cmd := m.withOverlay(nil); cmd == nil || m.ovKey == "" || len(m.ovPlaces) != 1 {
		t.Fatalf("no paint scheduled: key %q places %v", m.ovKey, m.ovPlaces)
	}
	if strings.ContainsRune(m.screen(), '\U0010EEEE') {
		t.Error("stand-in cells reach the terminal")
	}
	if m.withOverlay(nil) != nil {
		t.Error("an unchanged screen was rescheduled")
	}
	if m.drawOverlay(m.ovGen-1) != nil {
		t.Error("a stale paint was drawn")
	}
	if m.drawOverlay(m.ovGen) == nil || !m.ovDrawn {
		t.Fatal("paint not issued")
	}
	if m.drawOverlay(m.ovGen) != nil {
		t.Error("painted twice")
	}

	// An animation wipes the images (clear screen) and nothing is repainted
	// until it ends.
	m.anim = &anim{}
	if m.withOverlay(nil) == nil || m.ovDrawn || m.ovKey != "" {
		t.Error("animation did not wipe the images")
	}
	if m.drawOverlay(m.ovGen) != nil {
		t.Error("painted during an animation")
	}
	m.anim = nil
	if m.withOverlay(nil) == nil || m.ovKey == "" {
		t.Error("images not scheduled after the animation")
	}

	// A slide without images leaves nothing scheduled, and wipes the old ones.
	m.ovDrawn = true
	m.slide = 1
	if m.withOverlay(nil) == nil || m.ovDrawn || m.ovKey != "" {
		t.Error("leaving the image slide did not wipe it")
	}
	if m.withOverlay(nil) != nil {
		t.Error("plain slide scheduled work")
	}
}

func TestOverlayOffForOtherModes(t *testing.T) {
	m := overlayModel(t)
	if m.withOverlay(nil) != nil || m.ovKey != "" {
		t.Error("overlay active without a sixel/iterm2 config")
	}
}
