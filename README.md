# tuipresent

tuipresent turns a markdown file into a full-screen slide deck in your terminal. Slides animate in, pauses fade content in, code is syntax-highlighted and can run live on stage, and images draw in full resolution in kitty and ghostty. It is built on Charm's Bubble Tea, Lip Gloss and Glamour, and its directive syntax is compatible with [presenterm](https://github.com/mfontanini/presenterm), so most presenterm decks work as-is. Because `-check` and `-dump` verify a deck's layout without a TTY, it's also a deck format that AI agents can write and check on their own.

## Highlights

- **Plain markdown decks**: slides split on `<!-- end_slide -->`, setext titles drawn with a gradient, columns, centered statement slides.
- **Motion**: five slide transitions (`slide`, `wipe`, `cover`, `dissolve`, `none`) and a per-cell fade for content revealed by `<!-- pause -->`.
- **Themes**: twenty built-in themes (tokyonight, catppuccin, gruvbox, dracula, nord, rose-pine, solarized, kanagawa and more), inline color overrides, custom theme files, and optional background painting so a deck looks the same in any terminal.
- **Images**: PNG, JPEG and GIF via the kitty graphics protocol, with a truecolor half-block fallback everywhere else.
- **Live demos**: `+exec` code blocks run on a key press (bash, sh, Python) and show their output under the code.
- **Speaker notes** in a second terminal that follows the presenter.
- **Live reload**: save the file and the open deck updates in place.
- **Verifiable**: `-check` reports every slide that overflows a given screen size; `-dump` prints every frame as text.

## Install

Requires Go 1.27 or newer.

```sh
go build -o bin/tuipresent ./cmd/tuipresent
# or
go install ./cmd/tuipresent
```

## Quick start

Save this as `hello.md`:

```markdown
---
title: Hello
author: Me
theme: catppuccin-mocha
---

First slide
===

* point one

<!-- pause -->

* point two

<!-- end_slide -->

<!-- jump_to_middle -->

That's it.
===
```

Present it, and check that it fits a small screen:

```sh
tuipresent hello.md                    # space / → next, ← back, q quit
tuipresent -check -w 80 -h 24 hello.md # hello.md: 3 slides fit 80x24
```

Then try the feature tour and the theme gallery:

```sh
tuipresent examples/tour.md
tuipresent -theme catppuccin-latte examples/themes.md
```

## Documentation

The [user guide](docs/guide/README.md) covers everything:

- [Getting started](docs/guide/getting-started.md): build, write a first deck, check it, present it.
- [Authoring](docs/guide/authoring.md): every slide construct with examples, gotchas and design advice.
- [Presenting](docs/guide/presenting.md): keys, a speaker-notes window, live reload, live demos, fitting the projector, troubleshooting.
- [Reference](docs/guide/reference.md): complete tables of flags, keys, warnings, front matter, themes, directives, images, `+exec`, transitions and layout numbers.

For AI agents and tools:

- [docs/agent/](docs/agent/README.md): the normative [language spec](docs/agent/LANGUAGE.md), [verified examples](docs/agent/EXAMPLES.md) with real output, and a [front-matter JSON Schema](docs/agent/frontmatter.schema.json).
- [llms.txt](llms.txt): an index of the docs for LLMs.
- [AGENT_GUIDE.md](docs/AGENT_GUIDE.md): the workflow for writing and verifying a deck, with layout budgets and design guidance.
- [SKILL.md](docs/agent-skill/SKILL.md): a condensed, standalone Claude Code skill.

[`examples/`](examples/README.md) has ready-to-run decks: a feature tour, an incident postmortem, a keynote-style pitch, a technical talk with live demos, a theme gallery and a deck in a custom theme.

## For AI agents

If you're asking a coding agent such as Claude Code to write a deck, point it at [`docs/AGENT_GUIDE.md`](docs/AGENT_GUIDE.md). The guide covers:

- the authoring workflow, from brief to verified deck,
- the layout budget (23 body rows at 100x30, 17 at 80x24) and rules of thumb,
- design guidance for content, themes, transitions and pauses,
- the required `-check` / `-dump` verification step and a pre-handoff checklist.

Exact syntax lives in [`docs/agent/LANGUAGE.md`](docs/agent/LANGUAGE.md).

[`docs/agent-skill/SKILL.md`](docs/agent-skill/SKILL.md) is a condensed version packaged as a Claude Code skill (`tuipresent-deck`). It triggers when you ask for a presentation, slides or a deck. To install it, copy the directory into your personal skills or into a project's skills:

```sh
mkdir -p ~/.claude/skills/tuipresent-deck
cp docs/agent-skill/SKILL.md ~/.claude/skills/tuipresent-deck/
# or, for one project only:
mkdir -p .claude/skills/tuipresent-deck
cp docs/agent-skill/SKILL.md .claude/skills/tuipresent-deck/
```

## Keeping docs and examples current

`internal/docscheck` fails `go test` when the docs or examples drift from the code:

- any deck in `examples/` overflows 100x30 or 80x24;
- the reference, the language spec, the agent guide or the agent skill omits a directive, transition kind, CLI flag or built-in theme name (and the theme gallery deck omits a theme);
- `docs/agent/frontmatter.schema.json` and the front-matter keys, theme keys or transition kinds in the code disagree;
- a relative link in README.md, llms.txt, `docs/` or `examples/README.md` points at a missing file;
- a layout-numbers table in the docs disagrees with the renderer;
- an output block in `docs/agent/EXAMPLES.md` differs from what the renderer produces (regenerate with `go test ./internal/docscheck -run Examples -update`).

Two hooks run it automatically:

- **Git pre-commit** (`.githooks/pre-commit`): gofmt, `go vet`, `go test`. Enable once per clone: `ln -s ../../.githooks/pre-commit .git/hooks/pre-commit`.
- **Claude Code Stop/SubagentStop** (`.claude/settings.json` -> `.claude/hooks/check-drift.sh`): the same checks when an agent finishes a turn; failures are fed back so the agent fixes the drift. Skipped when no `.go`/`.md` file changed since the last pass.

## Roadmap

Titles and text styling
- Big titles: a `title_size` directive using kitty's text-sizing protocol where supported, block letters elsewhere
- Inline markdown (bold, italic, code) in slide titles
- Title styling: curly accent underline, framed title cards, subtitle line
- More text styles from markdown: strikethrough, highlight, faint, underline variants
- Callout boxes for GitHub-style `> [!NOTE]` / `> [!WARNING]` blocks

Terminal capabilities
- Clickable links (OSC 8)
- Sixel and iTerm2 image protocols for WezTerm, iTerm2 and foot
- Deck progress in the terminal tab via the OSC 9;4 progress bar

Presenting
- Slide overview grid and jump-to-slide by number
- Navigate the presenter from the notes view
- Mouse navigation

Sharing and export
- Record decks to GIF/MP4 with Charm's VHS; export PDF/PNG slides with Charm's Freeze

