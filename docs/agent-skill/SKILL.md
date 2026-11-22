---
name: tuipresent-deck
description: Write and verify a terminal slide deck for tuipresent, a presenterm-compatible markdown presenter. Use when the user asks to create, make, build or write a presentation, slides, slide deck or talk for the terminal, or mentions tuipresent or presenterm.
---

# tuipresent deck authoring

Full reference: `docs/AGENT_GUIDE.md` in the tuipresent repo. Read it when a rule here is unclear, or before using columns, pauses or `+exec` together.

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
   `-check` reports rows over budget and these column overflows: wide body lines, too-long titles, text clipped in columns, and code lines over the no-wrap limit. It does not count `+exec` output unless you add `-exec`.
   For decks with `+exec`, add `-exec` to `-check`/`-dump` to run every block first and measure its output. This executes the deck's code. Use it only on a deck you wrote or reviewed, with blocks that follow the safety rules below, and only with the user's approval. Otherwise budget output rows by hand. Read the dump for readability and reveal order.
5. **Fix and re-verify** until both checks pass and the dumps look right.
6. **Hand off.** Give the path, slide count, `./bin/tuipresent deck.md`, and the keys: `space`/`→` next, `←` back, `e` run exec, `r` reload, `q` quit. Also list what each `+exec` block runs.

## Syntax

- **Front matter** must start at line 1 with `---`. Keys: `title`, `sub_title`, `author`, `date`, `theme`, `transition`.
  - `theme: dracula` and `theme:` / `  name: dracula` are equivalent. Use `dark`, `tokyonight-storm`, `dracula` or `light`.
  - Themes match by substring in this order: `light` or `day` gives light (so `tokyonight-day` is light), then `tokyo`, then `dracula`, otherwise `dark`.
  - Setting `title` adds a cover slide, so total slides = sections + 1.
- **Slides** are separated by `<!-- end_slide -->` on its own line. It is ignored inside code fences.
- **Slide title:** a setext H1 (`Title` then a `===` line) before any other content. `# Title` is NOT a title. It renders as a body heading.
- **Directives** must be lowercase and alone on their line. Unknown ones are silently dropped.
  - `<!-- pause -->` reveals the rest of the slide on the next key press.
  - `<!-- column_layout: [2, 1] -->` starts columns. `<!-- column: 0 -->` selects one (0-based). `<!-- reset_layout -->` ends them.
  - `<!-- jump_to_middle -->` centers the slide vertically and horizontally. Plain-prose bodies are centered line by line. Bodies with a list, table, quote or code are centered as one block.
  - `<!-- transition: slide|wipe|cover|dissolve|none -->` sets the transition for this slide. The front matter `transition:` sets the deck default, which is `slide` if unset. Unknown values fall back to `slide`.
- **Not supported** (silently dropped): `speaker_note`, `incremental_lists`, `new_lines`, `alignment`, `font_size`, `include`, `skip_slide`, `no_footer`.
- **`+exec`**: ```` ```bash +exec ````, `sh`, or `python`/`python3`/`py`. Any other language runs through bash.
  - Runs only when the presenter presses `e`, or with `-exec` during `-check`/`-dump`.
  - 15 s timeout. Combined stdout and stderr appears in a text block below the code.
- **Images:** `![alt](path)` alone on its line; path relative to the deck; PNG, JPEG or GIF. Scaled to fit with aspect kept: height = body rows left free on the fully revealed slide (split between stacked images, min 3), width at most the content or column width. `![image:width:50%](path)` caps the width. kitty/ghostty draw real pixels; other terminals, `-dump` and `-check` use truecolor half-blocks with the same geometry. A missing file renders as a dim `[image: alt]` box (3 rows), never an error.
- **CLI flags:** `-check`, `-dump`, `-exec` (with `-check`/`-dump`: run exec blocks first), `-w W`, `-h H` (sizes for `-check`/`-dump`), `-slide N` (start slide, cover counts), `-transition KIND` (overrides the front matter default, also after reload), `-images auto|kitty|blocks|off` (image drawing; `auto` uses kitty graphics in kitty/ghostty outside tmux).

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
- `+exec` must be harmless, offline, idempotent and fast (`echo`, `date`, `seq`, pure `python3`). Never use destructive commands (`rm`, `git push`, `sudo`, installs), network commands (`curl`, `ssh`, `ping`), or anything that prints secrets. The presenter runs it live.

## Pre-handoff checklist

- [ ] Front matter is valid, and the theme and transition values are valid.
- [ ] Every slide starts with a setext title of 60 characters or fewer. All fences are closed.
- [ ] Directives are spelled exactly: `end_slide`, `pause`, `column_layout`, `column`, `reset_layout`, `jump_to_middle`, `transition`.
- [ ] No pauses in earlier columns unless intended. Exec blocks are safe and have rows reserved for output.
- [ ] `-check` exits 0 at 100x30 and at 80x24. With `+exec`: `-exec -check` also passes if approved, otherwise output rows were budgeted by hand.
- [ ] Both `-dump` outputs reviewed: no wrapped charts, clipped columns, cut-off titles or missing lines.
- [ ] Handoff includes the path, slide count, run command, keys and exec contents.
