---
name: tuipresent-deck
description: Write and verify a terminal slide deck for tuipresent, a presenterm-compatible markdown presenter. Use when the user asks to create, make, build or write a presentation, slides, slide deck or talk for the terminal, or mentions tuipresent or presenterm.
---

# tuipresent deck authoring

This skill is self-contained. In the tuipresent repo, `docs/AGENT_GUIDE.md` has the full workflow and design guidance and `docs/agent/LANGUAGE.md` the exact syntax rules; read them when a rule here is unclear, or before using columns, pauses or `+exec` together.

## Workflow

1. **Gather requirements.** Topic and key message, audience, length (about 1 slide per minute), whether live demos are wanted, theme, output path. State any assumptions you make.
2. **Outline.** One line per slide: title plus a single idea.
3. **Write** one markdown file using the rules below.
4. **Verify (mandatory).** Build, run `-check` at both sizes, then read `-dump` at both sizes:
   ```sh
   go build -o bin/tuipresent ./cmd/tuipresent
   ./bin/tuipresent -check -w 100 -h 30 deck.md   # must print "N slides fit", exit 0
   ./bin/tuipresent -check -w 80 -h 24 deck.md
   ./bin/tuipresent -dump -w 100 -h 30 deck.md | perl -pe 's/\e\[[0-9;:]*[A-Za-z]//g'
   ./bin/tuipresent -dump -w 80 -h 24 deck.md | perl -pe 's/\e\[[0-9;:]*[A-Za-z]//g'
   ```
   `-check` reports rows over budget and these column overflows: wide body lines, too-long titles, text clipped in columns, and code lines over the no-wrap limit. It does not count `+exec` output unless you add `-exec`. It prints warnings to stderr as `deck.md: warning: ...` (unknown theme or transition name, unknown theme key, a speaker note missing `-->`) without failing: fix every one.
   For decks with `+exec`, add `-exec` to `-check`/`-dump` to run every block first and measure its output. This executes the deck's code. Use it only on a deck you wrote or reviewed, with blocks that follow the safety rules below, and only with the user's approval. Otherwise budget output rows by hand. Read the dump for readability and reveal order.
5. **Fix and re-verify** until both checks pass and the dumps look right.
6. **Hand off.** Give the path, slide count, `./bin/tuipresent deck.md`, and the keys: `space`/`→` next, `←` back, `o` overview grid, a number + `enter` jumps to a slide, mouse click (left/right third) or wheel navigates (`-mouse=false` disables), `e` run exec, `r` reload, `q` quit. Also list what each `+exec` block runs.

## Syntax

- **Front matter** must start at line 1 with `---`. Keys: `title`, `sub_title`, `author`, `date`, `theme`, `transition`, `fade`, `progress`, `title_style`.
  - `theme: dracula` and `theme:` / `  name: dracula` are equivalent. Names are case-insensitive but otherwise exact (no substring matching): dark themes `dark` (default), `tokyonight-storm`, `tokyonight-night`, `tokyonight-moon`, `catppuccin-mocha`, `catppuccin-macchiato`, `catppuccin-frappe`, `gruvbox-dark`, `dracula`, `nord`, `rose-pine`, `solarized-dark`, `everforest-dark`, `kanagawa`; light themes `light`, `tokyonight-day`, `catppuccin-latte`, `gruvbox-light`, `rose-pine-dawn`, `solarized-light`. Aliases such as `tokyonight`, `catppuccin`, `gruvbox`, `solarized` map to the first variant. An unknown name renders in `dark` with a warning.
  - Mapping keys: `name`; `background: terminal|paint` (the painting mode, not a color: `paint` draws every cell on the theme background; use it for light themes, bright rooms/projectors, or an unknown presenting terminal); palette overrides `foreground`, `dim`, `accent`, `accent2`, `heading`, `subheading`, `link`, `bullet`, `code`, `code_background`, `quote`, `border`, `highlight`, `note`, `tip`, `important`, `warning`, `caution` as quoted colors (`accent: "#88c0d0"`; a bare `#` is a YAML comment); `code_style` (a chroma style); `path: themes/x.yaml` for a theme file relative to the deck. A theme file has `extends: nord` (only valid there) plus the same palette keys, where `background` is a color. Bad colors or an unreadable theme file are load errors.
  - Theme choice: light theme + `background: paint` for bright rooms/projectors; dark themes for dark rooms and screens.
  - Setting `title` adds a cover slide, so total slides = sections + 1.
- **Slides** are separated by `<!-- end_slide -->` on its own line. It is ignored inside code fences. Sections with no title, content or notes are dropped; a section with only speaker notes is kept as a blank slide.
- **Slide title:** a setext H1 (`Title` then a `===` line) before any other content. `# Title` is NOT a title. It renders as a body heading. Titles take inline markdown (`**bold**`, `` `code` ``). `<!-- subtitle: text -->` adds a line (+1 row); `<!-- title_style: curly -->` underlines the title with a curly line where the terminal supports it (else the rule), `frame` boxes it (+1 row); `title_style:` in front matter sets the default.
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
- **Images:** `![alt](path)` alone on its line; path relative to the deck; PNG, JPEG or GIF. Scaled to fit with aspect kept: height = body rows left free on the fully revealed slide (split between stacked images, min 3), width at most the content or column width. `![image:width:50%](path)` caps the width. kitty/ghostty draw real pixels; other terminals, `-dump` and `-check` use truecolor half-blocks with the same geometry. A missing file renders as a dim `[image: alt]` box (3 rows), never an error.
- **CLI flags:** `-check`, `-dump`, `-exec` (with `-check`/`-dump`: run exec blocks first), `-w W`, `-h H` (sizes for `-check`/`-dump`), `-slide N` (start slide, cover counts), `-transition KIND` (overrides the front matter default, also after reload), `-watch` (default on: reload on save, keeping the slide, so the user can keep the deck open while you edit; a parse error keeps the old deck and shows in the status line; `-watch=false` disables), `-notes` (speaker notes view that follows a presenter of the same deck), `-mouse=false` (no mouse reporting; leaves text selection to the terminal), `-fade=false` (instant pause reveals, overrides the deck), `-progress=false` (no deck progress in the terminal tab; front matter `progress: false` does it deck-wide), `-links auto|on|off` (OSC 8 hyperlinks; `-dump`/`-check` never emit them), `-images auto|kitty|blocks|off` (image drawing; `auto` uses kitty graphics in kitty/ghostty outside tmux), `-theme NAME` (preview in another theme or a `.yaml` theme file, overrides front matter, also after reload), `-background terminal|paint`, `-themes` (list themes with swatches).

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

## Pre-handoff checklist

- [ ] Front matter is valid, the theme name is exact (no `-check` warning on stderr), colors are quoted, and the transition values are valid.
- [ ] Every slide starts with a setext title of 60 characters or fewer. All fences are closed.
- [ ] Directives are spelled exactly: `end_slide`, `pause`, `column_layout`, `column`, `reset_layout`, `jump_to_middle`, `transition`, `speaker_note`, `fade`, `title_style`, `subtitle`.
- [ ] No pauses in earlier columns unless intended. Exec blocks are safe and have rows reserved for output.
- [ ] `-check` exits 0 at 100x30 and at 80x24. With `+exec`: `-exec -check` also passes if approved, otherwise output rows were budgeted by hand.
- [ ] Both `-dump` outputs reviewed: no wrapped charts, clipped columns, cut-off titles or missing lines.
- [ ] Handoff includes the path, slide count, run command, keys and exec contents.
