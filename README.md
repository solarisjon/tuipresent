# tuipresent

A terminal presentation tool. It renders a markdown deck as full-screen slides with animated transitions between them, built on Charm's Bubble Tea, Lip Gloss and Glamour. Its directive syntax is compatible with [presenterm](https://github.com/mfontanini/presenterm), so most existing presenterm decks work as-is.

## Build

Requires Go 1.27+.

```sh
go build -o bin/tuipresent ./cmd/tuipresent
# or
go install ./cmd/tuipresent
```

## Usage

```sh
tuipresent [flags] deck.md
```

| Flag | Default | Description |
|------|---------|-------------|
| `-slide N` | `1` | Slide to start on (1-based) |
| `-transition KIND` | | Default transition: `slide`, `wipe`, `cover`, `dissolve`, `none`. Overrides front matter. |
| `-notes` | | Show the speaker notes view instead of the slides. It follows a presenter of the same deck running in another terminal (see [Speaker notes](#speaker-notes)). |
| `-dump` | | Print every frame (each slide at each pause step) to stdout and exit. Doesn't need a TTY. |
| `-check` | | Report slides that overflow a `-w` x `-h` screen and exit 1 if any do (exit 0 and print `N slides fit WxH` otherwise). Doesn't need a TTY. |
| `-exec` | | With `-check` or `-dump`: run every `+exec` block first so its output is measured and shown. This executes the deck's code, so only use it on decks you trust. |
| `-w W` | `100` | Frame width for `-dump` and `-check` |
| `-h H` | `30` | Frame height for `-dump` and `-check` |

### Keys

| Key | Action |
|-----|--------|
| `→` `l` `space` `n` `j` `↓` `enter` `pgdown` | Next step or slide |
| `←` `h` `p` `k` `↑` `backspace` `pgup` | Previous step or slide |
| `g` `home` | First slide |
| `G` `end` | Last slide |
| `e` | Run the visible `+exec` code blocks on this slide |
| `r` | Reload the deck from disk |
| `q` `esc` `ctrl+c` | Quit |

## Deck format

A deck is a single markdown file. Slides are separated by `<!-- end_slide -->`. Directives inside code fences are ignored.

### Front matter

Optional YAML at the top of the file. When `title` is set, a cover slide is generated from it.

```yaml
---
title: My Talk
sub_title: A subtitle
author: Jane Doe
date: 2026-12-08
theme:
  name: tokyonight-storm   # dark (default), tokyonight-storm, dracula, light (or *-day); `theme: dracula` also works
transition: wipe           # tuipresent extension
---
```

### Slide titles

A setext H1 (a line underlined with `===`) that comes before any other content on a slide becomes the slide title. It's drawn with a gradient and a rule under it.

### Directives

| Directive | Source | Effect |
|-----------|--------|--------|
| `<!-- end_slide -->` | presenterm | End the current slide |
| `<!-- pause -->` | presenterm | Reveal the content after this on the next key press |
| `<!-- column_layout: [2, 1] -->` | presenterm | Start a column layout with these relative widths |
| `<!-- column: N -->` | presenterm | Send the content that follows to column N (0-based) |
| `<!-- reset_layout -->` | presenterm | End the column layout and go back to full width |
| `<!-- jump_to_middle -->` | presenterm | Center the slide vertically and horizontally |
| `<!-- transition: KIND -->` | tuipresent | Transition used when entering this slide (`slide`, `wipe`, `cover`, `dissolve`, `none`) |
| `<!-- speaker_note: text -->` | presenterm | Speaker note for this slide, shown only in the `-notes` view (multi-line form under [Speaker notes](#speaker-notes)) |
| front matter `transition:` | tuipresent | Default transition for the whole deck (`slide` if unset) |

Directives tuipresent doesn't recognize are ignored.

### Speaker notes

Notes never appear on the slide and take no space on it. A slide can have several; they're shown in order. Note text is markdown. Use either form:

```markdown
<!-- speaker_note: Mention the 40% number first. -->

<!-- speaker_note:
- Pause for questions here
- Then demo the check flag
-->
```

To present with notes, split your terminal (kitty: `ctrl+shift+enter` in the splits layout; ghostty: `cmd+d` on macOS, `ctrl+shift+o` on Linux) and run the notes view in the pane only you can see:

```sh
tuipresent deck.md          # presenter, in the window the audience sees
tuipresent -notes deck.md   # notes, in the other pane
```

The notes view shows the current slide's notes, `slide N/M` and the reveal step, the next slide's title and the time since the presenter started. It follows the presenter through a small state file in the system temp directory, keyed by the deck's absolute path, so no network is involved and either one can start first. When the presenter quits, the notes view goes back to "waiting". Press `q` to quit it.

### Executable code

Add `+exec` to a fenced code block's info string to make it runnable. Press `e` and the output appears under the code. Nothing runs unless you press `e` or pass `-exec` to `-check`/`-dump`. `bash` is the default interpreter. `sh` and `python3` (also `python`, `py`) are supported too. Each run times out after 15 seconds.

````markdown
```bash +exec
uname -a
```
````

## Example

````markdown
---
title: Hello
author: Me
transition: dissolve
---

First slide
===

* point one
<!-- pause -->
* point two

<!-- end_slide -->

Two columns
===

<!-- transition: wipe -->
<!-- column_layout: [1, 1] -->
<!-- column: 0 -->

Left side.

<!-- column: 1 -->

```bash +exec
date
```

<!-- reset_layout -->

Full-width footer text.
````

See `cpe-staff-meeting-demo.md` for a longer example.

## Examples

[`examples/`](examples/README.md) has four ready-to-run decks: a self-demonstrating feature tour, an incident postmortem, a keynote-style pitch and a technical talk with live `+exec` demos. Start with `./bin/tuipresent examples/tour.md`.

## For AI agents

If you're asking a coding agent such as Claude Code to write a deck, point it at [`docs/AGENT_GUIDE.md`](docs/AGENT_GUIDE.md). The guide covers:

- the authoring workflow,
- the exact syntax the parser accepts,
- the layout budget (23 body rows at 100x30, 17 at 80x24),
- silent-failure gotchas,
- the required `-check` / `-dump` verification step.

[`docs/agent-skill/SKILL.md`](docs/agent-skill/SKILL.md) is a condensed version packaged as a Claude Code skill (`tuipresent-deck`). It triggers when you ask for a presentation, slides or a deck. To install it, copy the directory into your personal skills or into a project's skills:

```sh
mkdir -p ~/.claude/skills/tuipresent-deck
cp docs/agent-skill/SKILL.md ~/.claude/skills/tuipresent-deck/
# or, for one project only:
mkdir -p .claude/skills/tuipresent-deck
cp docs/agent-skill/SKILL.md .claude/skills/tuipresent-deck/
```

## Keeping docs and examples current

`internal/docscheck` fails `go test` when any deck in `examples/` overflows 100x30 or 80x24, or when README.md, docs/AGENT_GUIDE.md or the agent skill omits a directive, transition kind or CLI flag that the code supports. Two hooks run it automatically:

- **Git pre-commit** (`.githooks/pre-commit`): gofmt, `go vet`, `go test`. Enable once per clone: `ln -s ../../.githooks/pre-commit .git/hooks/pre-commit`.
- **Claude Code Stop/SubagentStop** (`.claude/settings.json` -> `.claude/hooks/check-drift.sh`): the same checks when an agent finishes a turn; failures are fed back so the agent fixes the drift. Skipped when no `.go`/`.md` file changed since the last pass.

## Roadmap

- Inline images via the kitty graphics protocol, with a text fallback for other terminals
- Per-element fade-in on pause, using truecolor interpolation
- Live reload when the deck file changes
- More themes
