# Reference

Every command-line flag, key, front-matter key, directive and layout number in tuipresent, in tables. This page is derived from the source code; for explanations and examples see [Authoring](authoring.md) and [Presenting](presenting.md).

## Contents

- [Command line](#command-line)
- [Key bindings](#key-bindings)
- [File format](#file-format)
- [Front matter](#front-matter)
- [Themes](#themes)
- [Directives](#directives)
- [Slide titles](#slide-titles)
- [Images](#images)
- [Executable code blocks](#executable-code-blocks)
- [Transitions](#transitions)
- [Layout](#layout)
- [Timings](#timings)

## Command line

```sh
tuipresent [flags] deck.md
```

Exactly one deck file is required. Boolean flags take `-flag` or `-flag=false`.

| Flag | Default | Description |
|------|---------|-------------|
| `-slide N` | `1` | Slide to start on, 1-based. The generated cover counts as slide 1. Out-of-range values are clamped. |
| `-transition KIND` | | Default transition: `slide`, `wipe`, `cover`, `dissolve`, `none`. Overrides front matter `transition`; per-slide directives still win. |
| `-fade` | `true` | Fade in content revealed by a pause. `-fade=false` makes every reveal instant, overriding the deck. |
| `-watch` | `true` | Reload the deck when the file changes. `-watch=false` disables it (`r` still reloads). |
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

Other exit statuses: `2` for a usage error (no deck, more than one, unknown `-images` mode), `1` when the deck can't be read or parsed.

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
- Optional YAML front matter: the file must start with a `---` line, and the block ends at the next `---` line.
- Slides are separated by `<!-- end_slide -->` on its own line. `---` is not a separator.
- Sections with no content and no title are dropped.
- A deck with no slides at all fails to load.

## Front matter

| Key | Type | Effect |
|-----|------|--------|
| `title` | string | Generates a cover slide (title in spaced capitals, centered) and shows in the footer |
| `sub_title` | string | Cover subtitle, italic |
| `author` | string | Cover byline and footer |
| `date` | string | Cover byline, after the author |
| `theme` | string or `{name: string}` | Color theme, see [Themes](#themes). `theme: x` and `theme: {name: x}` are equivalent. |
| `transition` | string | Default transition kind. Unset means `slide`. |
| `fade` | string or bool | `false`, `off`, `no` or `none` (any case) disables the pause fade deck-wide. Anything else, or unset, leaves it on. |

Other keys are ignored.

## Themes

<!-- TODO(themes): fill from theme system -->

A deck picks its color theme with the `theme` key in front matter. See the theme list in [README.md](../../README.md) for the available themes.

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
| `<!-- transition: KIND -->` | transition kind | Transition used when entering this slide (and when leaving it backwards) |
| `<!-- fade: off -->` | `off`/`false`/`no`/`none`, or `on` | Disables (or re-enables) the pause fade for this slide, overriding front matter |
| `<!-- speaker_note: text -->` | markdown | Adds a speaker note, shown only in the `-notes` view |

`speaker_note` also has a multi-line form; the body is dedented and treated as markdown:

```markdown
<!-- speaker_note:
- first point
- second point
-->
```

Content placed between `column_layout` and the first `column:` goes full width above the columns.

## Slide titles

A non-blank line followed by a line of `=` characters (setext H1), appearing before any other content on the slide, is the slide title. Directives may precede it. It renders as a gradient title with a rule, taking 4 rows. `# ATX` headings are body headings, not titles.

## Images

```markdown
![alt text](path/to/image.png)
![alt text](<path with spaces.png> "optional title")
![image:width:50%](path/to/image.png)
```

| Rule | Detail |
|------|--------|
| Placement | The image must be alone on its line (leading whitespace allowed) |
| Path | Relative to the deck file's directory, or absolute |
| Formats | PNG, JPEG, GIF (first frame) |
| Width option | Alt text exactly `image:width:N%` caps the image at N% of the available width (N over 100 counts as 100). The alt text then becomes the file name. |
| Sizing | Scaled to fit the width and the rows left free on the fully revealed slide, shared among stacked images, at least 3 rows; aspect ratio kept (a cell is treated as 1x2 pixels); at most 297 cells per side |
| Columns | Supported. An image in a shorter column may use the rows its taller neighbor leaves free. |
| Spacing | One blank row before an image (if anything precedes it) and before markdown that follows it |
| Missing or undecodable file, or `-images off` | A dim 3-row box reading `[image: alt]` |

## Executable code blocks

A fenced code block whose info string contains the word `+exec` is runnable: ```` ```bash +exec ````.

| Language (first word of the info string) | Command |
|------------------------------------------|---------|
| `python`, `python3`, `py` | `python3 -c CODE` |
| `sh` | `sh -c CODE` |
| anything else, or none | `bash -c CODE` |

| Behavior | Detail |
|----------|--------|
| Trigger | `e` in the presenter, or `-exec` with `-check`/`-dump`. Never automatic. |
| Scope of `e` | Blocks on the current slide at or before the current step, including inside columns |
| Timeout | 15 seconds per block, then the process is killed |
| Output | Combined stdout and stderr, shown below the code in a `text` block. A failure appends `[exit status N]` (or `[signal: killed]` on timeout). |
| Working directory | The directory `tuipresent` was started from |
| Lifetime | Cleared on reload |

## Transitions

| Kind | Forward | Backward |
|------|---------|----------|
| `slide` | New slide pushes in from the right | Enters from the left |
| `wipe` | New slide revealed left to right | Revealed right to left |
| `cover` | New slide drops in from the top | Rises from the bottom |
| `dissolve` | Rows switch to the new slide in random order | Same |
| `none` | Instant | Instant |

Which kind runs, in order of precedence:

1. Going backwards: the `transition:` directive of the slide being left, if it has one.
2. The `transition:` directive of the slide being entered.
3. `-transition` on the command line.
4. Front matter `transition`.
5. `slide`.

Unrecognized kinds behave as `slide`. Jumping with `g`/`G` uses the same rules; jumping to the current slide is instant.

## Layout

Formulas for a `w` x `h` terminal:

| Quantity | Formula |
|----------|---------|
| Content width `cw` | `min(w - 8, 110)`; if that's under 20, `w - 2` |
| Left margin | `(w - cw) / 2` |
| Footer | 2 rows (progress bar, then title · author and `N / total`) when `h >= 6`, otherwise none |
| Top margin | 1 row, none on `jump_to_middle` slides |
| Title | 4 rows (blank, title, rule, blank) |
| Prose wrap width | `cw - 2` |
| Longest code line keeping its margin | `cw - 4` (wider lines wrap at `cw - 2`) |
| Column gap | 4 cells |
| Column width | `floor((cw - 4 * (n - 1)) * weight / total weight)` |

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
| `+exec` timeout | 15 s |
