# Reference

Every command-line flag, key, front-matter key, theme, directive and layout number in tuipresent, in tables. This page is the complete reference and is derived from the source code; `go test ./internal/docscheck` fails if it misses a flag, directive, transition or theme, or if its layout numbers drift. For explanations and examples see [Authoring](authoring.md) and [Presenting](presenting.md).

## Contents

- [Command line](#command-line)
- [Warnings and errors](#warnings-and-errors)
- [Key bindings](#key-bindings)
- [File format](#file-format)
- [Front matter](#front-matter)
- [Themes](#themes)
- [Directives](#directives)
- [Slide titles](#slide-titles)
- [Speaker notes](#speaker-notes)
- [Images](#images)
- [Executable code blocks](#executable-code-blocks)
- [Transitions](#transitions)
- [Pause fade](#pause-fade)
- [Layout](#layout)
- [Timings](#timings)

## Command line

```sh
tuipresent [flags] deck.md
tuipresent -themes
```

Exactly one deck file is required (except with `-themes`). Boolean flags take `-flag` or `-flag=false`.

| Flag | Default | Description |
|------|---------|-------------|
| `-slide N` | `1` | Slide to start on, 1-based. The generated cover counts as slide 1. Out-of-range values are clamped. |
| `-transition KIND` | | Default transition: `slide`, `wipe`, `cover`, `dissolve`, `none`. Overrides front matter `transition`; per-slide directives still win. An unknown kind acts as `slide` and warns. Still applies after a reload. |
| `-theme NAME` | | Render in this theme instead of the front matter's: a built-in name (or alias) or a `.yaml`/`.yml` theme file. Replaces the deck's theme name, path and inline overrides but keeps its `background` setting. Still applies after a reload. |
| `-background MODE` | | `paint` draws every cell on the theme's background; `terminal` uses the terminal's own. Overrides front matter. Any other value is a usage error. |
| `-themes` | | Print the built-in theme names, each marked `light` or `dark` with a one-line color swatch, and exit. |
| `-fade` | `true` | Fade in content revealed by a pause. `-fade=false` makes every reveal instant, overriding the deck. |
| `-watch` | `true` | Reload the deck (and its custom theme file) when the file changes. `-watch=false` disables it (`r` still reloads). |
| `-notes` | `false` | Show the speaker-notes view instead of the slides. It follows a presenter of the same deck in another terminal. |
| `-images MODE` | `auto` | `auto`: kitty graphics in kitty/ghostty outside tmux/screen, half-blocks elsewhere. `kitty`: force kitty graphics. `blocks`: force truecolor half-blocks. `off`: draw a `[image: alt]` box. `-dump` and `-check` always use half-blocks unless `off`. |
| `-check` | `false` | Report slides that overflow a `-w` x `-h` screen, then exit. No TTY needed. |
| `-dump` | `false` | Print every slide at every pause step to stdout, then exit. No TTY needed. |
| `-w W` | `100` | Screen width for `-check` and `-dump` |
| `-h H` | `30` | Screen height for `-check` and `-dump` |
| `-exec` | `false` | With `-check` or `-dump` only: run every `+exec` block first, so its output is measured and shown. Executes the deck's code. |

`-check` output and exit status:

```text
deck.md: 12 slides fit 100x30                         exit 0
deck.md: slide 4 overflows 100x30 by 3 rows, 0 cols   exit 1 (one line per slide)
```

`-dump` prints a header before each frame, then exactly `-h` lines of exactly `-w` cells (with ANSI colors):

```text
=== slide 2 step 1/3 ===
```

Exit statuses: `0` success; `1` when the deck can't be read or parsed, its custom theme is invalid, or `-check` found an overflow; `2` for a usage error (no deck, more than one, unknown `-images` mode, bad `-background`).

## Warnings and errors

Problems tuipresent can recover from are **warnings**: the deck still loads and renders, with a fallback. With `-check` and `-dump` each one is printed to stderr as `deck.md: warning: ...` (the exit status doesn't change); while presenting they show on the status line (the bottom row) for 6 seconds, at start and after every reload.

| Warning | Fallback |
|---------|----------|
| Unknown theme name (with a "did you mean" suggestion) | `dark` |
| Unknown key, `extends`, or a nested (non-scalar) value in the front-matter theme mapping; unknown or nested key in a theme file | Ignored |
| Unknown `code_style` | The base theme's style |
| Theme `background` other than `terminal` or `paint` | `terminal` |
| Theme with both `name` and `path` | The path |
| Unknown transition kind: front matter, `-transition`, or a `transition:` directive (with its line number) | `slide` |
| Multi-line `speaker_note` with no closing `-->` (with its line number) | The note ends before the next `<!-- end_slide -->`, or at the end of the file |

**Errors** stop the deck from loading: invalid front-matter YAML, a deck with no slides, and an invalid custom theme (unreadable theme file, a color that isn't `#rgb`/`#rrggbb`, an empty color, or `extends` naming an unknown theme). At startup they exit 1; during a reload the old deck stays up and the error shows on the status line until the next good save.

Everything else, such as a misspelled directive or a missing image, is silent. Read `-dump` to catch it.

## Key bindings

Presenter (the default view):

| Key | Action |
|-----|--------|
| `→` `l` `space` `n` `j` `↓` `enter` `pgdown` | Next step; on the last step, next slide |
| `←` `h` `p` `k` `↑` `backspace` `pgup` | Previous step; on the first step, previous slide at its last step |
| `g` `home` | First slide, first step |
| `G` `end` | Last slide, first step |
| `e` | Run the `+exec` blocks revealed so far on this slide |
| `r` | Reload the deck from disk (clears exec output) |
| `q` `esc` `ctrl+c` | Quit |

Notes view (`-notes`):

| Key | Action |
|-----|--------|
| `q` `esc` `ctrl+c` | Quit |

## File format

- One markdown file. Windows line endings are accepted.
- Optional YAML front matter: the file must start with a `---` line, and the block ends at the next `---` line. An empty block (`---` then `---`) is allowed.
- Slides are separated by `<!-- end_slide -->` on its own line. `---` is not a separator.
- A section with no title, no visible content and no speaker notes is dropped. An empty column layout is not visible content. A section with only speaker notes is kept as a blank slide.
- A deck with no slides at all fails to load.

## Front matter

| Key | Type | Effect |
|-----|------|--------|
| `title` | string | Generates a cover slide (title in spaced capitals, centered) and shows in the footer |
| `sub_title` | string | Cover subtitle, italic |
| `author` | string | Cover byline and footer, as `title  ·  author` |
| `date` | string | Cover byline, after the author: `author  ·  date` |
| `theme` | string or mapping | Color theme, see [Themes](#themes) |
| `transition` | string | Default transition kind. Unset means `slide`. |
| `fade` | string or bool | `false`, `off`, `no` or `none` (any case) disables the pause fade deck-wide. Anything else, or unset, leaves it on. |

Other keys are ignored. A JSON Schema for editors is in [`docs/agent/frontmatter.schema.json`](../agent/frontmatter.schema.json).

## Themes

Twenty built-in themes, each using its scheme's official palette and a matching syntax-highlighting (chroma) style. Run `tuipresent -themes` to see them with swatches, and `tuipresent -theme NAME deck.md` to preview a deck in another theme. Layout is identical in every theme.

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

**Names.** Matching ignores case and surrounding spaces but is otherwise exact; there is no substring matching. These aliases are also accepted:

| Alias | Theme |
|-------|-------|
| `tokyo-night`, `tokyonight`, `tokyo-night-storm` | `tokyonight-storm` |
| `tokyo-night-night` | `tokyonight-night` |
| `tokyo-night-moon` | `tokyonight-moon` |
| `tokyo-night-day`, `tokyo-night-light` | `tokyonight-day` |
| `catppuccin` | `catppuccin-mocha` |
| `gruvbox` | `gruvbox-dark` |
| `rosepine`, `rose-pine-main` | `rose-pine` |
| `rosepine-dawn` | `rose-pine-dawn` |
| `solarized` | `solarized-dark` |
| `everforest` | `everforest-dark` |
| `kanagawa-wave` | `kanagawa` |
| `terminal-dark`, `default` | `dark` |
| `terminal-light` | `light` |

Any other name renders in `dark` with a warning. presenterm's built-in names render with tuipresent's palettes; presenterm theme files and settings such as an `override:` block aren't supported and are ignored with a warning.

**Front-matter forms.**

```yaml
theme: nord                      # a built-in name

theme:
  name: nord
  background: paint              # terminal (default) or paint
  accent: "#88c0d0"              # override palette keys inline
  code_style: monokai

theme:
  path: themes/brand.yaml        # a custom theme file, relative to the deck
  background: paint
```

| Theme mapping key | Value |
|-------------------|-------|
| `name` | A built-in theme name or alias |
| `path` | A custom theme file, relative to the deck file (or absolute). Wins over `name`, with a warning if both are set. |
| `background` | `terminal` (default) or `paint`. In the front matter this is the painting mode, not a color; set the background color in a theme file. |
| palette keys | Any key in the palette table below except `background`, as a quoted `"#rrggbb"` or `"#rgb"` color, applied on top of `name` or `path` |
| `code_style` | A chroma style name |

`extends` is only valid in a theme file; in the front matter it is ignored with a warning (use `name:`).

**Palette keys.** A derived key left unset follows its default, so overriding `accent` also recolors headings, links and bullets that use it.

| Key | Colors | Default |
|-----|--------|---------|
| `background` | Slide background: painted with `background: paint`, and what pause reveals fade up from | |
| `foreground` | Body text | |
| `dim` | Footer, cover byline, missing-image boxes | |
| `accent` | Gradient start (titles, rules), progress bar, status line | |
| `accent2` | Gradient end | |
| `heading` | `#` headings | `accent` |
| `subheading` | `##` to `######` headings | `accent2` |
| `link` | Links | `accent` |
| `bullet` | List bullets and numbers | `accent` |
| `code` | Inline code text | `accent2` |
| `code_background` | Inline code and the panel behind code blocks | `background` mixed 8% toward `foreground` |
| `quote` | Blockquote text and bar | `dim` |
| `border` | Table borders, horizontal rules | `dim` |

`code_style` (not a color) selects the syntax-highlighting style: any [chroma style](https://xyproto.github.io/splash/docs/) name, such as `monokai`. Default: the base theme's.

**Custom theme files.** YAML with an optional `extends:` naming a built-in (default `dark`), then any palette keys and `code_style`. The theme's name is the file name. The file reloads while you present, like the deck. [`examples/themes/brand.yaml`](../../examples/themes/brand.yaml) is a complete example, used by `examples/brand.md`.

```yaml
# themes/brand.yaml
extends: nord
accent: "#ff6b35"
accent2: "#f7c548"
code_background: "#262b36"
code_style: monokai
```

Quote colors: in YAML a bare `#` starts a comment, which leaves the key empty (an error).

**Background painting.** By default slides are drawn on the terminal's own background, so a theme looks right only when the terminal's background roughly matches it. With `background: paint` (or `-background paint`), every cell of the frame is drawn on the theme's `background` and `foreground`, re-applied after every color reset, so the deck looks the same in any terminal: transitions, the pause fade (which then fades up from the painted background), the status line and kitty images included. Geometry is unchanged. Use it for light themes, shared decks, and unknown presenting terminals.

**What a theme colors.** The gradient slide titles and rules, headings, body text, bullets, links, inline code, code blocks (on a `code_background` panel two cells wider than the longest line), quotes, table borders, the footer, the progress bar and the status line.

## Directives

A directive is an HTML comment alone on a line: `<!-- name -->` or `<!-- name: argument -->`. Names are lowercase letters and underscores. Unknown names are ignored silently. Directives inside fenced code blocks are not directives.

| Directive | Argument | Effect |
|-----------|----------|--------|
| `<!-- end_slide -->` | | Ends the current slide |
| `<!-- pause -->` | | Everything after it appears one step later. One counter per slide, shared by all columns. |
| `<!-- column_layout: [2, 1] -->` | relative widths | Starts a column layout. Non-positive or non-numeric entries are dropped; none left means one column. |
| `<!-- column: N -->` | 0-based index | Sends following content to column N. Clamped to the valid range. Ignored outside a layout or if N isn't a number. |
| `<!-- reset_layout -->` | | Ends the column layout; following content is full width |
| `<!-- jump_to_middle -->` | | Centers the slide vertically and horizontally |
| `<!-- transition: KIND -->` | transition kind | Transition used when entering this slide, and in reverse when going back from it. Unknown kinds act as `slide` and warn. |
| `<!-- fade: off -->` | `off`/`false`/`no`/`none`, or `on` | Disables (or re-enables) the pause fade for this slide, overriding front matter |
| `<!-- speaker_note: text -->` | markdown | Adds a speaker note, shown only in the `-notes` view |

`transition` and `fade` (and the matching front-matter keys) are tuipresent extensions; the other directives come from presenterm. Content placed between `column_layout` and the first `column:` goes full width above the columns. Unsupported presenterm directives (`incremental_lists`, `new_lines`, `alignment`, `font_size`, `include`, `skip_slide`, `no_footer`, ...) are ignored.

## Slide titles

A non-blank line followed by a line of `=` characters (setext H1), appearing before any other content on the slide, is the slide title. Directives and speaker notes may precede it. It renders as a gradient title with a rule, taking 4 rows, and never wraps. Its text is literal (no inline markdown). `# ATX` headings are body headings, not titles.

## Speaker notes

```markdown
<!-- speaker_note: Mention the 40% number first. -->

<!-- speaker_note:
- Pause for questions here
- Then demo the check flag
-->
```

| Rule | Detail |
|------|--------|
| Display | Only in the notes view (`tuipresent -notes deck.md`), never on the slide; notes take no rows |
| Format | Markdown. The multi-line body is dedented. Several notes on a slide are shown in order. |
| Multi-line form | `<!-- speaker_note:` on its own line (no `-->` on it), then note lines, then a line containing `-->` |
| Missing `-->` | The note ends before the next `<!-- end_slide -->` (which still splits) or at the end of the file, with a warning. Any other directive before that, such as `<!-- pause -->`, closes the note and is swallowed. |
| Paragraphs | A note does not separate paragraphs: put blank lines around it |
| Notes-only slide | Kept as a blank slide, for talking over |
| Notes view | Current slide's notes, `slide N/M`, the reveal step, the next slide's title and the time since the presenter started. It follows the presenter through a state file in the system temp directory, keyed by the deck's absolute path. |

## Images

```markdown
![alt text](path/to/image.png)
![alt text](<path with spaces.png> "optional title")
![image:width:50%](path/to/image.png)
```

| Rule | Detail |
|------|--------|
| Placement | The image must be alone on its line (leading whitespace allowed) |
| Path | Relative to the deck file's directory, or absolute. URLs aren't fetched. |
| Formats | PNG, JPEG, GIF (first frame) |
| Width option | Alt text exactly `image:width:N%` caps the image at N% of the available width (N over 100 counts as 100). The alt text then becomes the file name. |
| Sizing | Scaled to fit the width and the rows left free on the fully revealed slide, shared among stacked images, at least 3 rows; aspect ratio kept (a cell is treated as 1x2 pixels); at most 297 cells per side |
| Columns | Supported. An image in a shorter column may use the rows its taller neighbor leaves free. |
| Spacing | One blank row before an image (if anything precedes it) and before markdown that follows it |
| Drawing | kitty graphics protocol (Unicode placeholders) in kitty and ghostty outside tmux/screen; truecolor half-blocks elsewhere and in `-dump`/`-check`. Same geometry either way. See `-images`. |
| Missing or undecodable file, or `-images off` | A dim 3-row box reading `[image: alt]`, never an error |

## Executable code blocks

A fenced code block whose info string contains the word `+exec` is runnable: ```` ```bash +exec ````.

| Language (first word of the info string) | Command |
|------------------------------------------|---------|
| `python`, `python3`, `py` | `python3 -c CODE` |
| `sh` | `sh -c CODE` |
| `bash`, or none | `bash -c CODE` |
| anything else | not run; the output is `[unsupported +exec language "X" (supported: bash, sh, python, python3, py)]` |

| Behavior | Detail |
|----------|--------|
| Trigger | `e` in the presenter, or `-exec` with `-check`/`-dump`. Never automatic. |
| Scope of `e` | Blocks on the current slide at or before the current step, including inside columns, run concurrently. The status line shows `running…` until output arrives. |
| Working directory | The deck file's directory (like image paths), with tuipresent's environment |
| Timeout | 15 seconds per block, then the process is killed. A background process that keeps the output open can delay the result by at most another 0.5 s. |
| Output | Combined stdout and stderr, shown below the code in a `text` block. A failure appends `[exit status N]` (or `[signal: killed]` on timeout). |
| Lifetime | `e` again re-runs and replaces the output; a reload clears it |

## Transitions

| Kind | Forward | Backward |
|------|---------|----------|
| `slide` | New slide pushes in from the right | Enters from the left |
| `wipe` | New slide revealed left to right | Revealed right to left |
| `cover` | New slide drops in from the top | Rises from the bottom |
| `dissolve` | Rows switch to the new slide in random order | Same |
| `none` | Instant | Instant |

A slide's **effective transition** is its own `transition:` directive, else `-transition`, else front matter `transition`, else `slide`.

- Moving forward to a slide plays that slide's effective transition.
- Moving back from slide B to slide A plays **B's** effective transition in reverse: it undoes the transition that brought you to B.
- Jumping with `g`/`G` follows the same rules (forward or back by direction). Jumping to the current slide is instant.

Kinds are case-insensitive. An unknown kind acts as `slide` and warns. Transitions never run between pause steps; those fade.

## Pause fade

Stepping forward through a `<!-- pause -->` fades the newly revealed cells in from the background (the terminal's, queried at startup, falling back to the theme's; with `background: paint` always the theme's). What was already visible stays put. Stepping back is instant.

Turning it off, broadest first:

1. `-fade=false`: every reveal is instant; the deck can't re-enable it.
2. `<!-- fade: off -->` / `<!-- fade: on -->` on a slide: wins over the front matter for that slide.
3. Front matter `fade: false` (also `off`, `no`, `none`): the whole deck.

`-dump` and `-check` print static frames, so the fade never affects them.

## Layout

Formulas for a `w` x `h` terminal:

| Quantity | Formula |
|----------|---------|
| Content width `cw` | `min(w - 8, 110)`; on screens narrower than 28 cells, `min(w - 2, 20)` (never less than 1) |
| Left margin | `(w - cw) / 2` |
| Footer | 2 rows (progress bar, then ` title  ·  author` and `N / total`) when `h >= 6`, otherwise none |
| Top margin | 1 row, none on `jump_to_middle` slides |
| Title | 4 rows (blank, title, rule, blank) |
| Body rows | titled `h - 7`, untitled `h - 3`, titled `jump_to_middle` `h - 6` (for `h >= 6`) |
| Prose wrap width | `cw - 2` |
| Longest code line keeping its margin | `cw - 4` (wider lines wrap at `cw - 2`) |
| Column gap | 4 cells |
| Column width | `floor((cw - 4 * (n - 1)) * weight / total weight)`; per column, prose wraps at width - 2 and code keeps its margin up to width - 4 |

Numbers at the two standard sizes:

| | 100x30 | 80x24 |
|---|---|---|
| Content width | 92 | 72 |
| Left margin | 4 | 4 |
| Body rows, titled slide | 23 | 17 |
| Body rows, untitled slide | 27 | 21 |
| Body rows, titled `jump_to_middle` slide | 24 | 18 |
| Prose wrap width | 90 | 70 |
| Longest code line | 88 | 68 |
| `[1, 1]` column widths | 44 + 44 | 34 + 34 |
| `[2, 1]` column widths | 58 + 29 | 45 + 22 |
| `[3, 2]` column widths | 52 + 35 | 40 + 27 |
| `[1, 1, 1]` column widths | 28 x 3 | 21 x 3 |

Content that doesn't fit is clipped at the bottom and right edges. `-check` measures the fully revealed slide, so it catches this; `+exec` output is only measured with `-exec`.

Columns always take the height of their fully revealed content, and centered slides are positioned by their fully revealed size, so nothing moves as pauses reveal content.

## Timings

| What | Value |
|------|-------|
| Slide transition | 380 ms, ease-out cubic |
| Pause fade-in | 280 ms |
| Animation frame rate | 60 fps |
| Deck file polling (`-watch`) | every 400 ms |
| Notes view polling | every 200 ms |
| Warning on the status line | 6 s |
| `+exec` timeout | 15 s |
