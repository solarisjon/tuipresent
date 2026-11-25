---
name: tuipresent-deck
description: Write, verify and present a terminal slide deck with tuipresent, a presenterm-compatible markdown presenter. Use when the user asks to create, make, build or write a presentation, slides, slide deck, talk or keynote for the terminal, to check or fix a deck's layout, to preview, record or export one, or mentions tuipresent or presenterm.
---

# tuipresent deck authoring

Self-contained: the checkout of the tuipresent repo is not needed. Exact syntax is in `docs/agent/LANGUAGE.md`, real output for each construct in `docs/agent/EXAMPLES.md`, the full workflow and design guidance in `docs/AGENT_GUIDE.md`, the front matter schema in `docs/agent/frontmatter.schema.json`. Read LANGUAGE.md before using columns, pauses and `+exec` together, or when a rule here is unclear.

## Bootstrap (first)

Run `tuipresent -version`. If it is not found, install it, in this order:

1. `go install github.com/solarisjon/tuipresent/cmd/tuipresent@latest` (needs Go; the binary lands in `$(go env GOPATH)/bin`, which must be on `PATH`).
2. A release binary from https://github.com/solarisjon/tuipresent/releases: pick the one for the OS and CPU, `chmod +x`, put it on `PATH`.
3. From a clone: `go install ./cmd/tuipresent`.

Use `tuipresent` from `PATH`, not `./bin/tuipresent`, unless the user gives another path. Ask before installing anything. If no install is possible, say so at handoff and apply the layout budget below conservatively.

## Workflow

1. **Gather requirements.** Ask only what you cannot reasonably assume: topic and key message, audience, length (about 1 slide per minute), live demos, where it is shown (laptop, bright-room projector). Assume and state: 8-12 slides, theme `dark` (light theme + `background: paint` for projectors), no `+exec`, output `<topic-slug>.md` in the current directory. Do not ask about transitions or pauses.
2. **Outline.** One line per slide: title plus a single idea. Show it if the deck is long or the brief vague.
3. **Write** one markdown file using the rules below.
4. **Verify (mandatory).** Run `-check` at both sizes, then read `-dump` at both sizes:
   ```sh
   tuipresent -check -w 100 -h 30 deck.md   # must print "N slides fit", exit 0
   tuipresent -check -w 80 -h 24 deck.md
   tuipresent -dump -w 100 -h 30 deck.md | perl -pe 's/\e\[[0-9;:]*[A-Za-z]//g'
   tuipresent -dump -w 80 -h 24 deck.md | perl -pe 's/\e\[[0-9;:]*[A-Za-z]//g'
   ```
   `-check` reports rows over budget and column overflows (wide body lines, long titles, clipped column text, code lines over the no-wrap limit). Load errors exit 1. It prints `deck.md: warning: ...` to stderr (unknown theme or transition, unknown theme key, a speaker note missing `-->`) without failing: fix every one. It does not count `+exec` output, misspelled directives or readability: read the dump (the perl filter strips ANSI; `-dump` prints every slide at every pause step).
   `-exec` with `-check`/`-dump` runs every `+exec` block first and measures the output. It executes the deck's code: only on a deck you wrote or reviewed, only with the user's approval. Otherwise budget output rows by hand.
5. **Fix and re-verify** until both checks pass and the dumps look right.
6. **Hand off.** Give the path, slide count, the run command, and what each `+exec` block runs. Tell the user to run `tuipresent deck.md` in their own terminal (a presenter needs a real TTY; do not launch it yourself). Live reload is on by default (`-watch`), so they can keep it open while you edit: it reloads on save, keeping the slide and step, and shows a parse error in the status line while keeping the old deck. Keys: `space`/`→` next, `←` back, `o` overview grid, a number + `enter` jumps, mouse click (left/right third) or wheel navigates (`-mouse=false` disables), `e` runs `+exec`, `r` reload, `q` quit.

## Presenting, themes, notes

- Speaker notes: tell the user to run `tuipresent -notes deck.md` in a second terminal, same machine. It shows the notes and next slide, follows the presenter, and its navigation keys also drive the presenter.
- Themes: `tuipresent -themes` lists them with swatches. `-theme NAME` (or a `.yaml` theme file) previews one without editing the deck; `-background terminal|paint` overrides painting. `-slide N` starts on a slide, `-transition KIND` overrides the default.
- Other flags: `-fade=false`, `-progress=false`, `-links auto|on|off`, `-images auto|kitty|sixel|iterm2|blocks|off` (`auto` never draws real pixels inside tmux), `-watch=false`, `-version`.
- Sharing, only when asked: `tuipresent -export tape deck.md` prints a VHS tape; `-export vhs` records it and `-export freeze` makes one PNG/SVG per slide (separate installs of Charm's VHS and Freeze; no PDF, combine PNGs with `magick slides/slide-*.png deck.pdf`). Options: `-out`, `-format`, `-dwell`. Run with `-dry-run` first (needs neither tool). `-exec` with `-export` runs deck code: only with the user's approval.
- Skill setup: `tuipresent -install-skill` writes this skill into `./.claude/skills/tuipresent-deck`; `-skill-dir user|project|PATH` picks the place, `-force` overwrites, `-print-skill` prints SKILL.md for other agent runtimes.

## Syntax

- **Front matter** must start at line 1 with `---`. Keys: `title`, `sub_title`, `author`, `date`, `theme`, `transition`, `fade`, `progress`, `title_style`, `title_size`.
  - `theme: dracula` and `theme:` / `  name: dracula` are equivalent. Names are case-insensitive but otherwise exact (no substring matching): dark themes `dark` (default), `tokyonight-storm`, `tokyonight-night`, `tokyonight-moon`, `catppuccin-mocha`, `catppuccin-macchiato`, `catppuccin-frappe`, `gruvbox-dark`, `dracula`, `nord`, `rose-pine`, `solarized-dark`, `everforest-dark`, `kanagawa`; light themes `light`, `tokyonight-day`, `catppuccin-latte`, `gruvbox-light`, `rose-pine-dawn`, `solarized-light`. Aliases such as `tokyonight`, `catppuccin`, `gruvbox`, `solarized` map to the first variant. An unknown name renders in `dark` with a warning.
  - Mapping keys: `name`; `background: terminal|paint` (the painting mode, not a color: `paint` draws every cell on the theme background; use it for light themes, bright rooms/projectors, or an unknown presenting terminal); palette overrides `foreground`, `dim`, `accent`, `accent2`, `heading`, `subheading`, `link`, `bullet`, `code`, `code_background`, `quote`, `border`, `highlight`, `note`, `tip`, `important`, `warning`, `caution` as quoted colors (`accent: "#88c0d0"`; a bare `#` is a YAML comment); `code_style` (a chroma style); `path: themes/x.yaml` for a theme file relative to the deck. A theme file has `extends: nord` (only valid there) plus the same palette keys, where `background` is a color. Bad colors or an unreadable theme file are load errors.
  - Theme choice: light theme + `background: paint` for bright rooms/projectors; dark themes for dark rooms and screens.
  - Setting `title` adds a cover slide, so total slides = sections + 1.
- **Slides** are separated by `<!-- end_slide -->` on its own line. It is ignored inside code fences. Sections with no title, content or notes are dropped; a section with only speaker notes is kept as a blank slide.
- **Slide title:** a setext H1 (`Title` then a `===` line) before any other content. `# Title` is NOT a title. It renders as a body heading. Titles take inline markdown (`**bold**`, `` `code` ``). `<!-- subtitle: text -->` adds a line (+1 row); `<!-- title_style: curly -->` underlines the title with a curly line where the terminal supports it (else the rule), `frame` boxes it (+1 row); `title_style:` in front matter sets the default. `<!-- title_size: 2 -->` or `3` draws a short title in big block letters (+2 or +4 rows; kitty draws scaled text); a title with unsupported characters or too wide warns and stays normal.
- **Text styles:** `~~strike~~`, `==highlight==`, `++underline++`, `[x]{.faint}`, `[x]{.curly}` (also `.dotted`, `.dashed`, `.double`). **Callouts:** a blockquote starting `> [!NOTE]`, `[!TIP]`, `[!IMPORTANT]`, `[!WARNING]` or `[!CAUTION]` is drawn as a themed box: its lines + 2 rows, 4 cells narrower.
- **Directives** must be lowercase and alone on their line, with blank lines around them (a directive between two lines does not separate paragraphs). Unknown ones are silently dropped.
  - `<!-- pause -->` reveals the rest of the slide on the next key press. New content fades in from the background (280 ms); what was already visible stays put. Stepping back is instant.
  - `<!-- fade: off -->` makes this slide's reveals instant. Front matter `fade: false` does it deck-wide; `<!-- fade: on -->` re-enables one slide.
  - `<!-- column_layout: [2, 1] -->` starts columns. `<!-- column: 0 -->` selects one (0-based). `<!-- reset_layout -->` ends them.
  - `<!-- jump_to_middle -->` centers the slide vertically and horizontally. Plain-prose bodies are centered line by line. Bodies with a list, table, quote or code are centered as one block.
  - `<!-- transition: slide|wipe|cover|dissolve|none -->` sets the transition into this slide; going back from it plays the same transition reversed. The front matter `transition:` sets the deck default, which is `slide` if unset. Unknown values fall back to `slide` with a warning.
  - `<!-- speaker_note: text -->`, or multi-line: `<!-- speaker_note:` / note lines / `-->`. Never shown on the slide and costs 0 rows. Put talking points here, not on the slide. Viewed with `tuipresent -notes deck.md` in a second terminal, whose navigation keys (same as the presenter's, including number then `enter`) also drive the presenter. A missing `-->` ends the note before the next directive line (warned); the directive still applies.
- **Links:** `[text](https://url)` is a clickable hyperlink (OSC 8) showing only `text` where the terminal supports it, else `text` then its dimmed URL; relative and `#anchor` links show text only. Use absolute URLs, and keep link text meaningful: the URL may not be visible.
- **Not supported** (silently dropped): `incremental_lists`, `new_lines`, `alignment`, `font_size`, `include`, `skip_slide`, `no_footer`.
- **`+exec`**: ```` ```bash +exec ````, `sh`, or `python`/`python3`/`py`. Any other language shows an `unsupported +exec language` error as the output.
- **Line highlights**: ```` ```rust {1-3|5|all} ````, also with `+exec` (```` ```bash {1|2} +exec ````). `|` separates groups; each is a reveal step (like a pause) with the other lines dimmed. Items are lines `5`, ranges `2-4`, or `all`, 1-based inside the fence. `{2-4}` alone is a static highlight with no extra step. Bad or out-of-range items warn; fix them.
  - Runs only when the presenter presses `e`, or with `-exec` during `-check`/`-dump`, in the deck file's directory.
  - 15 s timeout. Combined stdout and stderr appears in a text block below the code.
- **Images:** `![alt](path)` alone on its line; path relative to the deck; PNG, JPEG or GIF. Scaled to fit with aspect kept: height = body rows left free on the fully revealed slide (split between stacked images, min 3), width at most the content or column width. `![image:width:50%](path)` caps the width. kitty/ghostty (kitty graphics), iTerm2/WezTerm (iTerm2 inline images) and foot (sixel) draw real pixels; other terminals, tmux, `-dump` and `-check` use truecolor half-blocks with the same geometry in every mode. A missing file renders as a dim `[image: alt]` box (3 rows), never an error.

## Layout budget

| | 100x30 | 80x24 |
|---|---|---|
| Body rows, titled slide (`h - 7`) | 23 | 17 |
| Body rows, untitled slide (`h - 3`) | 27 | 21 |
| Content width, `min(w-8, 110)` | 92 | 72 |
| Prose wrap / code max (`cw-2` / `cw-4`) | 90 / 88 | 70 / 68 |
| `[1, 1]` columns (gap 4) | 44 + 44 | 34 + 34 |
| `[2, 1]` columns | 58 + 29 | 45 + 22 |

Design for 80x24:

- At most 6 bullets of 68 characters or fewer, and about 12 planned body rows per slide.
- Code at most 10 lines of 68 characters or fewer (tabs count as 4 spaces). Titles 60 characters or fewer (they never wrap).
- `[1, 1]` column lines 30 characters or fewer.
- `+exec` slides: code lines + output lines + about 5 must be 17 or fewer.
- Images: one per slide with at most 4 rows of other content, or in one `[1, 1]` column beside short bullets. In `-dump` they appear as `▀▄` rows; a `[image: …]` box means a bad path.

## Gotchas

- Pauses are global per slide. Content in a later column appears only after every pause in earlier columns. Put no pauses inside earlier columns.
- Column height is fixed at the fully revealed height. Overflow is silently truncated at the bottom and clipped at column or screen edges.
- Content after `column_layout` but before the first `column:` renders full width above the columns, with no blank row. An out-of-range `column: N` is clamped to the last column (negative becomes 0).
- Text after `reset_layout` gets 1 blank row above it. Count it in the budget.
- Blank lines and `&nbsp;` cannot add vertical space. Leading and trailing blanks are trimmed.
- An unclosed fence swallows every later `end_slide`.

## Design

- One idea per slide. The title states the claim.
- Charts go in ```` ```text ```` fences using block characters (`█▏▎▍▌▋▊▉`) and space-aligned labels.
- Use `jump_to_middle` statement slides for a thesis or key number, at most 1 in 5 slides.
- Transitions: one deck default (`slide` or `dissolve`). Override only with meaning: `cover` for a new section, `wipe` for a reveal, `none` for an evolving diagram.
- Pauses fade in what they add, so reveal whole units (a bullet, a column, a takeaway) where order matters, not on every slide. Use `<!-- fade: off -->` only for slides clicked through quickly.
- `+exec` must be harmless, offline, idempotent and fast (`echo`, `date`, `seq`, pure `python3`). Never use destructive commands (`rm`, `git push`, `sudo`, installs), network commands (`curl`, `ssh`, `ping`), or anything that prints secrets. The presenter runs it live.

## Troubleshooting

| Symptom | Fix |
|---|---|
| `tuipresent: command not found` | Bootstrap above; check `$(go env GOPATH)/bin` is on `PATH`. |
| `-check`: slide N overflows by R rows | Split the slide, cut bullets, shorten code, or drop a `pause`d block to the next slide. Re-run both sizes. |
| `-check`: overflows by C cols | Shorten the title (60 characters max) or the widest body, column or code line. |
| A directive does nothing | Unknown directives are dropped silently. It must be lowercase, spelled exactly, alone on its line with blank lines around it. |
| `warning: unknown theme` | Use an exact name from `tuipresent -themes`; aliases are only `tokyonight`, `catppuccin`, `gruvbox`, `solarized`. |
| `# Title` shows as a body heading | Titles are setext: `Title` then a `===` line. |
| `[image: alt]` box instead of a picture | Wrong path (relative to the deck) or not PNG, JPEG or GIF. |
| Images are blocks in tmux, over SSH, or in `-dump` | Expected: real pixels need kitty, Ghostty, iTerm2, WezTerm or foot outside tmux. Not a bug. |
| Everything after a fence is missing | An unclosed fence swallows every later `end_slide`. |
| Reload shows an error in the status line | The old deck stays up; fix the file and save again. |

## Pre-handoff checklist

- [ ] Front matter is valid, the theme name is exact (no `-check` warning on stderr), colors are quoted, and the transition values are valid.
- [ ] Every slide starts with a setext title of 60 characters or fewer. All fences are closed.
- [ ] Directives are spelled exactly: `end_slide`, `pause`, `column_layout`, `column`, `reset_layout`, `jump_to_middle`, `transition`, `speaker_note`, `fade`, `title_style`, `title_size`, `subtitle`.
- [ ] No pauses in earlier columns unless intended. Exec blocks are safe and have rows reserved for output.
- [ ] `-check` exits 0 at 100x30 and at 80x24. With `+exec`: `-exec -check` also passes if approved, otherwise output rows were budgeted by hand.
- [ ] Both `-dump` outputs reviewed: no wrapped charts, clipped columns, cut-off titles or missing lines.
- [ ] Handoff includes the path, slide count, run command, keys and exec contents.
