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
| `-fade` | `true` | Fade in content revealed by `<!-- pause -->` (truecolor, about 280 ms). `-fade=false` makes reveals instant and overrides the deck. |
| `-dump` | | Print every frame (each slide at each pause step) to stdout and exit. Doesn't need a TTY. |
| `-check` | | Report slides that overflow a `-w` x `-h` screen and exit 1 if any do (exit 0 and print `N slides fit WxH` otherwise). Doesn't need a TTY. |
| `-exec` | | With `-check` or `-dump`: run every `+exec` block first so its output is measured and shown. This executes the deck's code, so only use it on decks you trust. |
| `-theme NAME` | | Render in this theme instead of the front matter's (a built-in name or a `.yaml` theme file). Keeps the deck's `background` setting and still applies after a reload. See [Themes](#themes). |
| `-themes` | | Print the built-in theme names, each with a one-line color swatch, and exit. |
| `-background MODE` | | `paint` draws every cell on the theme's background; `terminal` uses the terminal's own. Overrides front matter. |
| `-images MODE` | `auto` | How to draw images: `kitty` (graphics protocol), `blocks` (truecolor half-blocks), `off` (text placeholder) or `auto` (kitty in kitty/ghostty outside tmux, otherwise blocks). `-dump` and `-check` always use blocks unless `off`. |
| `-w W` | `100` | Frame width for `-dump` and `-check` |
| `-h H` | `30` | Frame height for `-dump` and `-check` |
| `-watch` | `true` | Reload the deck automatically when the file changes on disk (polled every 400 ms, so editors that save by rename work too). Keeps the current slide and step. If the new file fails to parse, the old deck stays up and the error shows in the status line. Disable with `-watch=false`. |

### Keys

| Key | Action |
|-----|--------|
| `→` `l` `space` `n` `j` `↓` `enter` `pgdown` | Next step or slide |
| `←` `h` `p` `k` `↑` `backspace` `pgup` | Previous step or slide |
| `g` `home` | First slide |
| `G` `end` | Last slide |
| `e` | Run the visible `+exec` code blocks on this slide |
| `r` | Reload the deck from disk (happens automatically on save unless `-watch=false`) |
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
  name: tokyonight-storm   # see Themes below; `theme: dracula` also works
  background: paint        # optional: paint the theme background into every cell
transition: wipe           # tuipresent extension
fade: false                # tuipresent extension: instant pause reveals (fade is on by default)
---
```

### Themes

Twenty built-in themes, using each scheme's official palette and a matching syntax-highlighting style. Names are exact (case-insensitive), as in presenterm:

| Theme | Kind | Code style (chroma) |
|-------|------|---------------------|
| `dark` (default) | dark | onedark |
| `light` | light | github |
| `tokyonight-storm` | dark | tokyonight-storm |
| `tokyonight-night` | dark | tokyonight-night |
| `tokyonight-moon` | dark | tokyonight-moon |
| `tokyonight-day` | light | tokyonight-day |
| `catppuccin-mocha` | dark | catppuccin-mocha |
| `catppuccin-macchiato` | dark | catppuccin-macchiato |
| `catppuccin-frappe` | dark | catppuccin-frappe |
| `catppuccin-latte` | light | catppuccin-latte |
| `gruvbox-dark` | dark | gruvbox |
| `gruvbox-light` | light | gruvbox-light |
| `dracula` | dark | dracula |
| `nord` | dark | nord |
| `rose-pine` | dark | rose-pine |
| `rose-pine-dawn` | light | rose-pine-dawn |
| `solarized-dark` | dark | solarized-dark |
| `solarized-light` | light | solarized-light |
| `everforest-dark` | dark | evergarden (chroma has no everforest; this is its closest relative) |
| `kanagawa` | dark | kanagawa (built into tuipresent; chroma has none) |

A deck that already works in presenterm keeps working: presenterm's built-in names render with tuipresent's palette, and presenterm-only settings such as an `override:` block are ignored with a warning (presenterm theme files aren't supported).

Run `tuipresent -themes` to print them all with a color swatch, and `tuipresent -theme NAME deck.md` to preview any deck in another theme without editing it (`-theme` also survives live reload). `examples/themes.md` is a gallery deck made for this: `./bin/tuipresent -theme nord examples/themes.md`.

Older spellings still work: `tokyo-night` and `tokyonight` mean `tokyonight-storm`, `catppuccin` means `catppuccin-mocha`, `gruvbox` means `gruvbox-dark`, `solarized` means `solarized-dark`, `everforest` means `everforest-dark`, and `terminal-dark`/`terminal-light` mean `dark`/`light`. Names are never matched by substring. An unknown name renders in `dark` and warns: on the status line while presenting, and on stderr with `-check` and `-dump`.

A theme colors the gradient slide titles and rules, headings, body text, bullets, links, inline code, code blocks (drawn on a panel in the code background), quotes, table borders, the footer and the progress bar.

#### Front-matter forms

```yaml
theme: nord                      # a built-in name

theme:
  name: nord
  background: paint              # paint the theme background (default: terminal)
  accent: "#88c0d0"              # override any palette key inline

theme:
  path: themes/brand.yaml        # a custom theme file, relative to the deck
  background: paint
```

#### Background painting

By default slides are drawn on the terminal's own background, so a theme looks as intended only when the terminal's background roughly matches it (a dark theme in a dark terminal). With `background: paint`, or `-background paint` on the command line, every cell of the frame is drawn on the theme's background and in its foreground, so the deck looks the same in any terminal: transitions, the pause fade (which then fades up from the painted background), the status line and kitty images included. Use it for light themes, for decks shared with others, and whenever you don't know the presenting terminal's colors.

#### Custom themes

A theme file is YAML: an optional `extends:` naming a built-in (default `dark`), then any palette keys to override. Quote colors: in YAML a bare `#` starts a comment. Colors are `#rrggbb` or `#rgb`; anything else, an unreadable file or an unknown `extends` is an error that names the key. Unknown keys are ignored with a warning. The file reloads while you present, like the deck.

```yaml
# themes/brand.yaml
extends: nord
accent: "#ff6b35"
accent2: "#f7c548"
code_background: "#262b36"
code_style: monokai
```

| Key | Colors | Default |
|-----|--------|---------|
| `background` | Slide background: painted with `background: paint`, and what pause reveals fade up from | |
| `foreground` | Body text | |
| `dim` | Footer, cover byline, missing-image boxes | |
| `accent` | Gradient start (titles, rules), progress bar | |
| `accent2` | Gradient end | |
| `heading` | `#` headings | `accent` |
| `subheading` | `##` to `######` headings | `accent2` |
| `link` | Links | `accent` |
| `bullet` | List bullets and numbers | `accent` |
| `code` | Inline code text | `accent2` |
| `code_background` | Inline code and the panel behind code blocks | `background` mixed 8% toward `foreground` |
| `quote` | Blockquote text | `dim` |
| `border` | Table borders, horizontal rules | `dim` |
| `code_style` | Syntax highlighting: any [chroma style](https://xyproto.github.io/splash/docs/) name, e.g. `monokai` | the base theme's |

The same keys work inline in front matter on top of `name:` or `path:`. `examples/brand.md` uses `examples/themes/brand.yaml`. `-theme` also accepts a theme file path (`-theme my.yaml`).

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
| `<!-- fade: off -->` | tuipresent | Make this slide's pause reveals instant (`on` re-enables it when the front matter turned it off) |
| front matter `fade:` | tuipresent | `false` makes pause reveals instant deck-wide. By default, content a pause reveals fades in from the background while what was already shown stays put. Stepping back is instant. |

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

### Images

Put `![alt](path)` on a line of its own. The path is relative to the deck file; PNG, JPEG and GIF are supported. The image is scaled to fit the width and the rows the slide leaves free, keeping its aspect ratio, and it works inside columns. `![image:width:50%](path)` caps it at half the available width.

In kitty and ghostty, images are drawn in full resolution with the [kitty graphics protocol](https://sw.kovidgoyal.net/kitty/graphics-protocol/) (Unicode placeholders, so transitions still work). Elsewhere, including inside tmux, they're drawn with truecolor half-block characters. Use `-images` to force a mode. A missing or unreadable file shows as a dim `[image: alt]` box.

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

[`examples/`](examples/README.md) has ready-to-run decks: a self-demonstrating feature tour, an incident postmortem, a keynote-style pitch, a technical talk with live `+exec` demos, a theme gallery and a deck in a custom theme. Start with `./bin/tuipresent examples/tour.md`, then try `./bin/tuipresent -theme catppuccin-latte examples/themes.md`.

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

`internal/docscheck` fails `go test` when any deck in `examples/` overflows 100x30 or 80x24, or when README.md, docs/AGENT_GUIDE.md or the agent skill omits a directive, transition kind, CLI flag or built-in theme name that the code supports. Two hooks run it automatically:

- **Git pre-commit** (`.githooks/pre-commit`): gofmt, `go vet`, `go test`. Enable once per clone: `ln -s ../../.githooks/pre-commit .git/hooks/pre-commit`.
- **Claude Code Stop/SubagentStop** (`.claude/settings.json` -> `.claude/hooks/check-drift.sh`): the same checks when an agent finishes a turn; failures are fed back so the agent fixes the drift. Skipped when no `.go`/`.md` file changed since the last pass.
